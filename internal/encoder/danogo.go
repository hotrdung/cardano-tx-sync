// internal/encoder/danogo.go
package encoder

import (
	"cardano-tx-sync/internal/model"
	"cardano-tx-sync/internal/pb"
	"cardano-tx-sync/internal/transformer"
	"encoding/json"
	"time"

	"fmt"

	"github.com/gogo/protobuf/types"
)

// DanogoEncoder encodes the message into the custom CardanoTransaction protobuf format.
type DanogoEncoder struct{}

// Encode implements the Encoder interface.
func (e *DanogoEncoder) Encode(message model.TxnMessage) ([]byte, error) {
	// Transform the Ogmigo transaction model to our target protobuf model.
	pbTx, err := transformer.ToCardanoTransaction(message)
	if err != nil {
		return nil, fmt.Errorf("failed to transform transaction for danogo encoder: %w", err)
	}

	// Marshal the protobuf message into its binary format.
	// return proto.Marshal(MapMsgType2Event(42, pbTx))

	return json.Marshal(MapMsgType2Event(42, pbTx))
}

// func MapMsgType2Event(networkMagic int32, tx *pb.CardanoTransaction) proto.Message {
func MapMsgType2Event(networkMagic int32, tx *pb.CardanoTransaction) interface{} {
	// get current timestamp
	now := time.Now()
	ts, _ := types.TimestampProto(now)
	return &pb.CardanoTransactionEvent{
		NetworkMagic: networkMagic,
		Transaction:  tx,
		Timestamp:    ts,
	}
}
