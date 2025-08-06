// internal/pb/event.pb.go
package pb

import (
	"github.com/gogo/protobuf/types"
)

// CardanoTransactionEvent corresponds to the CardanoTransaction message in the proto file.
type CardanoTransactionEvent struct {
	Transaction  *CardanoTransaction `protobuf:"bytes,1,opt,name=transaction,proto3" json:"transaction,omitempty"`
	Timestamp    *types.Timestamp    `protobuf:"bytes,2,opt,name=timestamp,proto3" json:"timestamp,omitempty"`
	NetworkMagic int32               `protobuf:"varint,3,opt,name=network_magic,json=networkMagic,proto3" json:"network_magic,omitempty"`
}

// CardanoTransaction corresponds to the CardanoTransaction message in the proto file.
type CardanoTransaction struct {
	TransactionID string            `protobuf:"bytes,1,opt,name=transaction_id,json=transactionId,proto3" json:"transaction_id,omitempty"`
	HeaderHash    string            `protobuf:"bytes,2,opt,name=header_hash,json=headerHash,proto3" json:"header_hash,omitempty"`
	Slot          uint64            `protobuf:"varint,3,opt,name=slot,proto3" json:"slot,omitempty"`
	Redeemers     *types.Struct     `protobuf:"bytes,4,opt,name=redeemers,proto3" json:"redeemers,omitempty"`
	Datums        map[string][]byte `protobuf:"bytes,5,rep,name=datums,proto3" json:"datums,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
	Body          *TxBody           `protobuf:"bytes,6,opt,name=body,proto3" json:"body,omitempty"`
	Cbor          string            `protobuf:"bytes,7,opt,name=cbor,proto3" json:"cbor,omitempty"`
	Metadata      *types.Struct     `protobuf:"bytes,8,opt,name=metadata,proto3" json:"metadata,omitempty"`
	Certificates  []*types.Struct   `protobuf:"bytes,9,rep,name=certificates,proto3" json:"certificates,omitempty"`
	Votes         []*types.Struct   `protobuf:"bytes,10,rep,name=votes,proto3" json:"votes,omitempty"`
	Proposals     []*types.Struct   `protobuf:"bytes,11,rep,name=proposals,proto3" json:"proposals,omitempty"`
	Signatures    map[string]string `protobuf:"bytes,12,rep,name=signatures,proto3" json:"signatures,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
}

// TxBody corresponds to the TxBody message in the proto file.
type TxBody struct {
	Inputs                []*TxInput                  `protobuf:"bytes,1,rep,name=inputs,proto3" json:"inputs,omitempty"`
	Outputs               []*TxOutput                 `protobuf:"bytes,2,rep,name=outputs,proto3" json:"outputs,omitempty"`
	Mint                  *MintValue                  `protobuf:"bytes,3,opt,name=mint,proto3" json:"mint,omitempty"`
	ReferenceInputs       []*Reference                `protobuf:"bytes,4,rep,name=reference_inputs,json=referenceInputs,proto3" json:"reference_inputs,omitempty"`
	ValidityIntervalStart int64                       `protobuf:"varint,5,opt,name=validity_interval_start,json=validityIntervalStart,proto3" json:"validity_interval_start,omitempty"`
	ValidityIntervalEnd   int64                       `protobuf:"varint,5,opt,name=validity_interval_end,json=validityIntervalEnd,proto3" json:"validity_interval_end,omitempty"`
	Withdrawals           map[string]map[string]int64 `protobuf:"bytes,6,rep,name=withdrawals,proto3" json:"withdrawals,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"varint,2,opt,name=value,proto3"`
}

// TxInput corresponds to the TxInput message in the proto file.
type TxInput struct {
	TransactionID string `protobuf:"bytes,1,opt,name=transaction_id,json=transactionId,proto3" json:"transaction_id,omitempty"`
	Index         int32  `protobuf:"varint,2,opt,name=index,proto3" json:"index,omitempty"`
}

// TxOutput corresponds to the TxOutput message in the proto file.
type TxOutput struct {
	Address   string        `protobuf:"bytes,1,opt,name=address,proto3" json:"address,omitempty"`
	Datum     string        `protobuf:"bytes,2,opt,name=datum,proto3" json:"datum,omitempty"`
	DatumHash string        `protobuf:"bytes,3,opt,name=datum_hash,json=datumHash,proto3" json:"datum_hash,omitempty"`
	Value     *OutputValue  `protobuf:"bytes,4,opt,name=value,proto3" json:"value,omitempty"`
	Script    *types.Struct `protobuf:"bytes,5,opt,name=script,proto3" json:"script,omitempty"`
}

// OutputValue corresponds to the OutputValue message in the proto file.
type OutputValue struct {
	Coins  int64            `protobuf:"varint,1,opt,name=coins,proto3" json:"coins,omitempty"`
	Assets map[string]int64 `protobuf:"bytes,2,rep,name=assets,proto3" json:"assets,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"varint,2,opt,name=value,proto3"`
}

// MintValue corresponds to the MintValue message in the proto file.
type MintValue struct {
	Coins  int64            `protobuf:"varint,1,opt,name=coins,proto3" json:"coins,omitempty"`
	Assets map[string]int64 `protobuf:"bytes,2,rep,name=assets,proto3" json:"assets,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"varint,2,opt,name=value,proto3"`
}

// Reference corresponds to the Reference message in the proto file.
type Reference struct {
	TransactionID string `protobuf:"bytes,1,opt,name=transaction_id,json=transactionId,proto3" json:"transaction_id,omitempty"`
	Index         int32  `protobuf:"varint,2,opt,name=index,proto3" json:"index,omitempty"`
}
