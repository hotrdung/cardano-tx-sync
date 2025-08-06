// internal/storage/storage.go
package storage

import "cardano-tx-sync/internal/model"

// Storage defines the interface for database operations.
type Storage interface {
	AddMapping(mapping model.Mapping) (int, error)
	RemoveMapping(id int) error
	GetMappingsFor(mappingType, key string) ([]model.Mapping, error)
	SaveCheckpoint(checkpoint model.Checkpoint, maxCheckpoints int) error
	GetLatestCheckpoints(limit int) ([]model.Checkpoint, error)
	ClearCheckpoints() error
	Rollback(slot uint64) error
	Close() error
}
