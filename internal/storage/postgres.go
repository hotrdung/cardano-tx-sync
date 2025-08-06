// internal/storage/postgres.go
package storage

import (
	"cardano-tx-sync/config"
	"cardano-tx-sync/internal/model"
	"database/sql"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/patrickmn/go-cache"
)

// PostgresStorage implements the Storage interface for PostgreSQL.
type PostgresStorage struct {
	db    *sqlx.DB
	cache *cache.Cache
}

// NewPostgresStorage creates a new PostgresStorage instance.
func NewPostgresStorage(cfg config.PostgresConfig) (*PostgresStorage, error) {
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName, cfg.SSLMode)

	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		return nil, err
	}

	// Initialize cache with a 5-minute default expiration and 10-minute cleanup interval
	c := cache.New(5*time.Minute, 10*time.Minute)

	s := &PostgresStorage{
		db:    db,
		cache: c,
	}

	if err := s.initSchema(); err != nil {
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return s, nil
}

// initSchema creates the necessary tables if they don't exist.
func (s *PostgresStorage) initSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS mapping_groups (
		id SERIAL PRIMARY KEY,
		name TEXT NOT NULL UNIQUE,
		description TEXT
	);
	
	CREATE TABLE IF NOT EXISTS mappings (
		id SERIAL PRIMARY KEY,
		group_id INTEGER REFERENCES mapping_groups(id) ON DELETE CASCADE,
		type TEXT NOT NULL, -- 'address', 'policy_id', 'cert', 'proposal', 'vote'
		key TEXT NOT NULL,
		topic TEXT NOT NULL,
		encoder TEXT NOT NULL DEFAULT 'DEFAULT',
		UNIQUE(type, key, topic)
	);

	CREATE TABLE IF NOT EXISTS checkpoints (
		id SERIAL PRIMARY KEY,
		slot BIGINT NOT NULL,
		hash TEXT NOT NULL,
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);
	`
	_, err := s.db.Exec(schema)
	return err
}

// Close closes the database connection.
func (s *PostgresStorage) Close() error {
	return s.db.Close()
}

// AddMapping adds a new mapping to the database.
func (s *PostgresStorage) AddMapping(mapping model.Mapping) (int, error) {
	var id int
	query := `INSERT INTO mappings (group_id, type, key, topic, encoder) VALUES ($1, $2, $3, $4, $5) RETURNING id`
	err := s.db.QueryRow(query, mapping.GroupID, mapping.Type, mapping.Key, mapping.Topic, mapping.Encoder).Scan(&id)
	if err != nil {
		return 0, err
	}
	s.cache.Flush() // Invalidate cache
	return id, nil
}

// RemoveMapping removes a mapping from the database.
func (s *PostgresStorage) RemoveMapping(id int) error {
	query := `DELETE FROM mappings WHERE id = $1`
	_, err := s.db.Exec(query, id)
	if err != nil {
		return err
	}
	s.cache.Flush() // Invalidate cache
	return nil
}

// GetMappingsFor finds mappings for a given mapping type and key.
func (s *PostgresStorage) GetMappingsFor(mappingType, key string) ([]model.Mapping, error) {
	cacheKey := fmt.Sprintf("mappings:%s:%s", mappingType, key)
	if cached, found := s.cache.Get(cacheKey); found {
		return cached.([]model.Mapping), nil
	}

	var mappings []model.Mapping
	query := `SELECT id, group_id, type, key, topic, encoder FROM mappings WHERE type = $1 AND key = $2`
	err := s.db.Select(&mappings, query, mappingType, key)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	s.cache.Set(cacheKey, mappings, cache.DefaultExpiration)
	return mappings, nil
}

// SaveCheckpoint saves a new checkpoint.
func (s *PostgresStorage) SaveCheckpoint(checkpoint model.Checkpoint, maxCheckpoints int) error {
	tx, err := s.db.Beginx()
	if err != nil {
		return err
	}

	query := `INSERT INTO checkpoints (slot, hash) VALUES ($1, $2)`
	_, err = tx.Exec(query, checkpoint.Slot, checkpoint.Hash)
	if err != nil {
		tx.Rollback()
		return err
	}

	// Trim old checkpoints
	trimQuery := `
		DELETE FROM checkpoints
		WHERE id NOT IN (
			SELECT id FROM checkpoints ORDER BY slot DESC, id DESC LIMIT $1
		)`
	_, err = tx.Exec(trimQuery, maxCheckpoints)
	if err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

// GetLatestCheckpoints retrieves the latest checkpoints.
func (s *PostgresStorage) GetLatestCheckpoints(limit int) ([]model.Checkpoint, error) {
	var checkpoints []model.Checkpoint
	query := `SELECT slot, hash FROM checkpoints ORDER BY slot DESC, id DESC LIMIT $1`
	err := s.db.Select(&checkpoints, query, limit)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	return checkpoints, nil
}

// ClearCheckpoints removes all checkpoints.
func (s *PostgresStorage) ClearCheckpoints() error {
	query := `DELETE FROM checkpoints`
	_, err := s.db.Exec(query)
	return err
}

// Rollback deletes checkpoints after a given slot.
func (s *PostgresStorage) Rollback(slot uint64) error {
	query := `DELETE FROM checkpoints WHERE slot > $1`
	_, err := s.db.Exec(query, slot)
	return err
}
