// internal/encoder/simple.go
package encoder

import (
	"cardano-tx-sync/internal/model"
	"encoding/json"
)

// SimpleEncoder encodes the message with only the transaction ID.
type SimpleEncoder struct{}

// Encode implements the Encoder interface.
func (e *SimpleEncoder) Encode(message model.TxnMessage) ([]byte, error) {
	simpleMsg := struct {
		TxID string `json:"txId"`
	}{
		TxID: message.Tx.ID,
	}
	return json.Marshal(simpleMsg)
}
