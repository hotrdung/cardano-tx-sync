// internal/storage/storage.go
package storage

import "cardano-ogmios-kafka-bridge/internal/model"

// Storage defines the interface for database operations.
type Storage interface {
	AddMapping(mapping model.Mapping) (int, error)
	RemoveMapping(id int) error
	GetTopicsFor(mappingType, key string) ([]string, error)
	SaveCheckpoint(checkpoint model.Checkpoint, maxCheckpoints int) error
	GetLatestCheckpoints(limit int) ([]model.Checkpoint, error)
	ClearCheckpoints() error
	Rollback(slot uint64) error
	Close() error
}
