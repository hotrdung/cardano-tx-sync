// internal/encoder/encoder.go
package encoder

import (
	"cardano-tx-sync/internal/model"
	"fmt"
	"strings"
)

// Encoder defines the interface for message encoders.
type Encoder interface {
	Encode(message model.TxnMessage) ([]byte, error)
}

// GetEncoder returns an encoder instance by name.
func GetEncoder(name string) (Encoder, error) {
	switch strings.ToUpper(name) {
	case "DEFAULT":
		return &DefaultEncoder{}, nil
	case "SIMPLE":
		return &SimpleEncoder{}, nil
	case "DANOGO":
		return &DanogoEncoder{}, nil
	default:
		return nil, fmt.Errorf("unknown encoder: %s", name)
	}
}
