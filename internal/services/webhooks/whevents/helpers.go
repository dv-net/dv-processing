package whevents

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// rawMessage returns a buffer with the payload.
func rawMessage(p any) (*bytes.Buffer, error) {
	body, err := json.Marshal(p)
	if err != nil {
		return nil, err
	}

	payloadBuf := bytes.NewBuffer([]byte{})
	if err := json.Compact(payloadBuf, body); err != nil {
		return nil, fmt.Errorf("json compact error: %w", err)
	}

	return payloadBuf, nil
}
