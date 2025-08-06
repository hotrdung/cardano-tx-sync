// internal/handler/handler.go
package handler

import (
	"cardano-tx-sync/internal/encoder"
	"cardano-tx-sync/internal/kafka"
	"cardano-tx-sync/internal/model"
	"cardano-tx-sync/internal/storage"
	"encoding/json"
	"sync"

	"github.com/SundaeSwap-finance/ogmigo/ouroboros/chainsync"
	"go.uber.org/zap"
)

// BlockHandler processes blocks from Ogmios.
type BlockHandler struct {
	storage  storage.Storage
	producer *kafka.Producer
	logger   *zap.Logger
}

// NewBlockHandler creates a new BlockHandler.
func NewBlockHandler(storage storage.Storage, producer *kafka.Producer, logger *zap.Logger) *BlockHandler {
	return &BlockHandler{
		storage:  storage,
		producer: producer,
		logger:   logger,
	}
}

// HandleRollForward processes a new block.
func (h *BlockHandler) HandleRollForward(block chainsync.Block, maxCheckpoints int) error {
	blockDetails, txs, err := h.parseBlock(block)
	if err != nil {
		h.logger.Error("failed to parse block", zap.Error(err))
		return nil // Continue
	}

	h.logger.Info("processing block", zap.Uint64("slot", blockDetails.Slot), zap.String("hash", blockDetails.Hash), zap.Int("tx_count", len(txs)))

	var wg sync.WaitGroup
	for _, tx := range txs {
		wg.Add(1)
		go func(tx chainsync.Tx) {
			defer wg.Done()
			h.processTx(tx, blockDetails)
		}(tx)
	}
	wg.Wait()

	// Save checkpoint
	checkpoint := model.Checkpoint{
		Slot: blockDetails.Slot,
		Hash: blockDetails.Hash,
	}
	if err := h.storage.SaveCheckpoint(checkpoint, maxCheckpoints); err != nil {
		h.logger.Error("failed to save checkpoint", zap.Error(err))
		// This is a critical error, might need to stop the service
		return err
	}

	return nil
}

// HandleRollBackward processes a rollback.
func (h *BlockHandler) HandleRollBackward(rb *chainsync.Point) error {
	pointStruct, ok := rb.PointStruct()
	if !ok {
		h.logger.Warn("rollback requested - empty point")
		return nil
	}

	h.logger.Warn("rollback requested", zap.Any("point:", pointStruct))

	if err := h.storage.Rollback(pointStruct.Slot); err != nil {
		h.logger.Error("failed to perform rollback in storage", zap.Error(err))
		return err
	}

	// For simplicity, we broadcast the rollback event to a general topic.
	// A more advanced implementation might track which topics were affected.
	rollbackMsg := model.RollbackMessage{
		RollbackTo: struct {
			Slot uint64 `json:"slot"`
			Hash string `json:"hash"`
		}{
			Slot: pointStruct.Slot,
			Hash: pointStruct.ID,
		},
	}
	if err := h.producer.SendMessage("cardano.rollbacks", rollbackMsg); err != nil {
		h.logger.Error("failed to send rollback message to kafka", zap.Error(err))
	}

	return nil
}

func (h *BlockHandler) processTx(tx chainsync.Tx, blockDetails model.BlockDetails) {
	// topicsByEncoder groups topics by the required encoder name.
	// map[encoderName]map[topicName]struct{}
	topicsByEncoder := make(map[string]map[string]struct{})

	// addMapping finds all relevant mappings and groups their topics by encoder.
	addMapping := func(mappingType model.MappingType, key string) {
		mappings, err := h.storage.GetMappingsFor(string(mappingType), key)
		if err != nil {
			h.logger.Error("failed to get mappings",
				zap.String("type", string(mappingType)),
				zap.String("key", key),
				zap.Error(err))
			return
		}
		for _, m := range mappings {
			if _, ok := topicsByEncoder[m.Encoder]; !ok {
				topicsByEncoder[m.Encoder] = make(map[string]struct{})
			}
			topicsByEncoder[m.Encoder][m.Topic] = struct{}{}
		}
	}

	// 1. Address and Policy ID mappings
	addMapping(model.MappingTypeAddress, "*")

	for _, output := range tx.Outputs {
		// Check for address mappings
		addMapping(model.MappingTypeAddress, output.Address)

		// Check for policy ID mappings
		for policyID, _ := range output.Value {
			addMapping(model.MappingTypePolicyID, policyID)
		}
	}

	// 2. Certificate mappings
	if len(tx.Certificates) > 0 {
		addMapping(model.MappingTypeCert, "*") // For any certificate

		for _, cert := range tx.Certificates {
			var c map[string]interface{}
			if err := json.Unmarshal(cert, &c); err != nil {
				h.logger.Error("failed to unmarshal certificate", zap.Error(err), zap.Any("certificate", cert))
				continue
			}
			if certType, ok := c["type"].(string); ok {
				addMapping(model.MappingTypeCert, certType)
			} else {
				h.logger.Warn("certificate 'type' field is missing or not a string", zap.Any("certificate", cert))
			}
		}
	}

	// 3. Proposal mapping
	if len(tx.Proposals) > 0 {
		addMapping(model.MappingTypeProposal, "*")
	}

	// 4. Vote mapping
	if len(tx.Votes) > 0 {
		addMapping(model.MappingTypeVote, "*")
	}

	// If any mappings were matched, encode and send the message.
	if len(topicsByEncoder) > 0 {
		txnMsg := model.TxnMessage{Tx: tx, Block: blockDetails}
		for encoderName, topics := range topicsByEncoder {
			// Get the appropriate encoder
			enc, err := encoder.GetEncoder(encoderName)
			if err != nil {
				h.logger.Error("could not find encoder", zap.String("encoder", encoderName), zap.Error(err))
				continue
			}

			// Encode the message
			encodedMsg, err := enc.Encode(txnMsg)
			if err != nil {
				h.logger.Error("failed to encode message", zap.String("encoder", encoderName), zap.Error(err))
				continue
			}

			// Send to all topics for this encoder
			for topic := range topics {
				if err := h.producer.SendMessage(topic, encodedMsg); err != nil {
					h.logger.Error("failed to send message to kafka",
						zap.Error(err),
						zap.String("topic", topic),
						zap.String("encoder", encoderName),
						zap.String("tx_id", tx.ID))
				}
			}
		}
	}
}

func (h *BlockHandler) parseBlock(block chainsync.Block) (model.BlockDetails, []chainsync.Tx, error) {
	// Note: The `chainsync.Block` struct in ogmigo actually has `ID` for hash and `Slot` for slot.
	// The `Transactions` field holds the list of transactions.
	blockDetails := model.BlockDetails{
		Hash: block.ID,
		Slot: block.Slot,
		Era:  block.Era, // This will be "Alonzo", "Babbage", etc., or empty if not set.
	}
	return blockDetails, block.Transactions, nil
}
