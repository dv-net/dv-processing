package utils

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

func SpecialKey(bufSize int) (string, error) {
	buff := make([]byte, bufSize)
	if _, err := rand.Read(buff); err != nil {
		return "", fmt.Errorf("key random generate: %w", err)
	}
	return hex.EncodeToString(buff), nil
}
