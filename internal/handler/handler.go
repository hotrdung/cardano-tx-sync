// internal/handler/handler.go
package handler

import (
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
	// Use a map to collect unique topics to avoid sending duplicate messages.
	topics := make(map[string]struct{})

	// Helper function to add topics from storage to our map
	addTopics := func(mappingType model.MappingType, key string) {
		newTopics, err := h.storage.GetTopicsFor(string(mappingType), key)
		if err != nil {
			h.logger.Error("failed to get topics",
				zap.String("type", string(mappingType)),
				zap.String("key", key),
				zap.Error(err))
			return
		}
		for _, topic := range newTopics {
			topics[topic] = struct{}{}
		}
	}

	// 1. Address and Policy ID mappings
	for _, output := range tx.Outputs {
		// Check for address mappings
		addTopics(model.MappingTypeAddress, output.Address)

		// Check for policy ID mappings
		for policyID, _ := range output.Value {
			addTopics(model.MappingTypePolicyID, policyID)
		}
	}

	// 2. Certificate mappings
	if len(tx.Certificates) > 0 {
		addTopics(model.MappingTypeAnyCert, "any")
		for _, cert := range tx.Certificates {
			var c map[string]interface{}
			if err := json.Unmarshal(cert, &c); err != nil {
				h.logger.Error("failed to unmarshal certificate", zap.Error(err), zap.Any("certificate", cert))
				continue
			}

			switch c["type"].(string) {
			case "stakeDelegation":
				addTopics(model.MappingTypeCertType, "stakeDelegation")
			case "stakeCredentialRegistration":
				addTopics(model.MappingTypeCertType, "stakeCredentialRegistration")
			case "stakeCredentialDeregistration":
				addTopics(model.MappingTypeCertType, "stakeCredentialDeregistration")
			case "stakePoolRegistration":
				addTopics(model.MappingTypeCertType, "stakePoolRegistration")
			case "stakePoolRetirement":
				addTopics(model.MappingTypeCertType, "stakePoolRetirement")
			case "delegateRepresentativeRegistration":
				addTopics(model.MappingTypeCertType, "delegateRepresentativeRegistration")
			case "delegateRepresentativeUpdate":
				addTopics(model.MappingTypeCertType, "delegateRepresentativeUpdate")
			case "delegateRepresentativeRetirement":
				addTopics(model.MappingTypeCertType, "delegateRepresentativeRetirement")
			case "genesisDelegation":
				addTopics(model.MappingTypeCertType, "genesisDelegation")
			case "constitutionalCommitteeDelegation":
				addTopics(model.MappingTypeCertType, "constitutionalCommitteeDelegation")
			case "constitutionalCommitteeRetirement":
				addTopics(model.MappingTypeCertType, "constitutionalCommitteeRetirement")
			}
		}
	}

	// 3. Proposal mapping
	if len(tx.Proposals) > 0 {
		addTopics(model.MappingTypeProposal, "any")
	}

	// 4. Vote mapping
	if len(tx.Votes) > 0 {
		addTopics(model.MappingTypeVote, "any")
	}

	if len(topics) > 0 {
		msg := model.TxnMessage{
			Tx:    tx,
			Block: blockDetails,
		}
		for topic := range topics {
			if err := h.producer.SendMessage(topic, msg); err != nil {
				h.logger.Error("failed to send message to kafka", zap.Error(err), zap.String("topic", topic), zap.String("tx_id", tx.ID))
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
