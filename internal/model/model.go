package model

import "github.com/SundaeSwap-finance/ogmigo/ouroboros/chainsync"

// MappingType defines the type of a mapping.
type MappingType string

const (
	// MappingTypeAddress maps a specific address.
	MappingTypeAddress MappingType = "address"
	// MappingTypePolicyID maps a specific policy ID.
	MappingTypePolicyID MappingType = "policy_id"
	// MappingTypeCert maps transactions with certificates. Key can be a specific cert type or "*" for any.
	MappingTypeCert MappingType = "cert"
	// MappingTypeProposal maps transactions with any governance proposal. Key should be "*".
	MappingTypeProposal MappingType = "proposal"
	// MappingTypeVote maps transactions with any governance vote. Key should be "*".
	MappingTypeVote MappingType = "vote"
)

// Mapping represents a filter-to-Kafka-topic mapping.
type Mapping struct {
	ID      int         `json:"id" db:"id"`
	GroupID *int        `json:"group_id,omitempty" db:"group_id"`
	Type    MappingType `json:"type" db:"type"`
	Key     string      `json:"key" db:"key"`
	Topic   string      `json:"topic" db:"topic"`
	Encoder string      `json:"encoder,omitempty" db:"encoder"`
}

// Checkpoint represents a point in the blockchain to sync from.
type Checkpoint struct {
	Slot uint64 `json:"slot" db:"slot"`
	Hash string `json:"hash" db:"hash"`
}

// TxnMessage is the message format for publishing to Kafka
type TxnMessage struct {
	Tx          chainsync.Tx `json:"tx"`
	Block       BlockDetails `json:"block"`
	Invalidated bool         `json:"invalidated,omitempty"`
}

// BlockDetails contains metadata about the block
type BlockDetails struct {
	Hash string `json:"hash"`
	Slot uint64 `json:"slot"`
	Era  string `json:"era"`
}

// RollbackMessage is the message sent to Kafka to indicate a rollback
type RollbackMessage struct {
	RollbackTo struct {
		Slot uint64 `json:"slot"`
		Hash string `json:"hash"`
	} `json:"rollbackTo"`
}
