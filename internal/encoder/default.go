// internal/encoder/default.go
package encoder

import (
	"cardano-tx-sync/internal/model"
	"encoding/json"
)

// DefaultEncoder encodes the message as a full JSON object.
type DefaultEncoder struct{}

// Encode implements the Encoder interface.
func (e *DefaultEncoder) Encode(message model.TxnMessage) ([]byte, error) {
	return json.Marshal(message)
}
