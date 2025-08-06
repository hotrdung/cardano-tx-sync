package chainsync

import (
	"bytes"
	"cardano-tx-sync/config"
	"cardano-tx-sync/internal/handler"
	"cardano-tx-sync/internal/model"
	"cardano-tx-sync/internal/storage"
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/SundaeSwap-finance/ogmigo"
	"github.com/SundaeSwap-finance/ogmigo/ouroboros/chainsync"
	"go.uber.org/zap"
)

// Syncer manages the chain synchronization with Ogmios.
type Syncer struct {
	ogmigoClient *ogmigo.Client
	handler      *handler.BlockHandler
	storage      storage.Storage
	logger       *zap.Logger
	cfg          config.ChainSyncConfig
	startPoint   *model.Checkpoint
	mu           sync.Mutex
	closer       *ogmigo.ChainSync
}

// NewSyncer creates a new Syncer.
func NewSyncer(client *ogmigo.Client, handler *handler.BlockHandler, storage storage.Storage, logger *zap.Logger, cfg config.ChainSyncConfig) *Syncer {
	return &Syncer{
		ogmigoClient: client,
		handler:      handler,
		storage:      storage,
		logger:       logger,
		cfg:          cfg,
	}
}

// SetStartPoint sets a new point to start syncing from.
func (s *Syncer) SetStartPoint(point model.Checkpoint) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.logger.Info("setting new start point", zap.Uint64("slot", point.Slot), zap.String("hash", point.Hash))
	if err := s.storage.ClearCheckpoints(); err != nil {
		return fmt.Errorf("could not clear checkpoints: %w", err)
	}
	s.startPoint = &point

	// If a sync is in progress, close it to restart from the new point
	if s.closer != nil {
		s.closer.Close()
	}

	return nil
}

// Start begins the chain synchronization process.
func (s *Syncer) Start(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			err := s.runSync(ctx)
			if err != nil {
				s.logger.Error("chain sync error", zap.Error(err))
				// Exponential backoff or similar could be implemented here
				time.Sleep(5 * time.Second)
			}
		}
	}
}

func (s *Syncer) runSync(ctx context.Context) error {
	s.mu.Lock()
	var points []chainsync.Point
	if s.startPoint != nil {
		points = []chainsync.Point{chainsync.PointStruct{Slot: s.startPoint.Slot, ID: s.startPoint.Hash}.Point()}
		s.startPoint = nil // Consume the start point
	} else {
		checkpoints, err := s.storage.GetLatestCheckpoints(s.cfg.MaxCheckpointsToKeep)
		if err != nil {
			s.mu.Unlock()
			return fmt.Errorf("failed to get latest checkpoints: %w", err)
		}
		if len(checkpoints) > 0 {
			for _, cp := range checkpoints {
				points = append(points, chainsync.PointStruct{Slot: cp.Slot, ID: cp.Hash}.Point())
			}
			s.logger.Info("resuming from latest checkpoints", zap.Any("points", points))
		} else {
			s.logger.Info("no checkpoints found, starting from origin")
			points = []chainsync.Point{chainsync.Origin}
		}
	}
	s.mu.Unlock()

	// Define the callback function that will handle incoming messages.
	var callback ogmigo.ChainSyncFunc = func(ctx context.Context, data []byte) error {
		// Use a decoder for stricter JSON parsing, as requested.
		// This ensures that the response from Ogmios matches our expected struct.
		decoder := json.NewDecoder(bytes.NewReader(data))
		decoder.DisallowUnknownFields()

		var response chainsync.ResponsePraos
		if err := decoder.Decode(&response); err != nil {
			// If decoding fails, it might be an unknown response type or malformed JSON.
			// We log it but don't return an error to the websocket client,
			// as this would terminate the connection. We want to continue processing subsequent messages.
			s.logger.Error("failed to decode chainsync response",
				zap.Error(err),
				zap.String("data", string(data)))
			return nil
		}

		// Pass the successfully decoded response to the handler logic.
		return s.handleChainSyncResponse(&response)
	}

	// Start the chainsync process with the Ogmigo client.
	closer, err := s.ogmigoClient.ChainSync(ctx, callback, ogmigo.WithPoints(points...))
	if err != nil {
		return fmt.Errorf("failed to start chainsync: %w", err)
	}
	s.mu.Lock()
	s.closer = closer
	s.mu.Unlock()

	s.logger.Info("chainsync started")
	// Wait until the closer is done, which indicates the connection has been closed.
	<-closer.Done()
	s.logger.Info("chainsync connection closed")
	select {
	case err := <-closer.Err():
		return err
	default:
		return nil
	}
}

// handleChainSyncResponse processes the decoded response from Ogmios.
func (s *Syncer) handleChainSyncResponse(response *chainsync.ResponsePraos) error {
	switch response.Method {
	case chainsync.FindIntersectionMethod:
		findIntersectResult := response.MustFindIntersectResult()
		if findIntersectResult.Intersection != nil {
			// This is an informational message confirming the starting point of the sync.
			s.logger.Info("intersection found",
				zap.Any("intersection", findIntersectResult.Intersection),
			)
		} else {
			// This indicates a problem with the provided start points.
			s.logger.Warn("intersection not found, client should restart with different points")
			// A more robust implementation might trigger an automatic restart from origin
			// by clearing checkpoints.
		}

	case chainsync.NextBlockMethod:
		nextBlockResult := response.MustNextBlockResult()
		if nextBlockResult.Block != nil {
			// Handle a new block by passing it to the block handler.
			return s.handler.HandleRollForward(*nextBlockResult.Block, s.cfg.MaxCheckpointsToKeep)
		} else {
			// Handle a blockchain rollback.
			return s.handler.HandleRollBackward(nextBlockResult.Point)
		}

	default:
		s.logger.Warn("received an unknown methodname in chainsync response", zap.String("method", response.Method))
	}

	return nil
}
