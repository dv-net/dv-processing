package util //nolint:nolintlint,revive

import (
	"crypto/sha256"
	"encoding/hex"
)

func SHA256Signature(data []byte, secretKey string) string {
	sign := sha256.New()
	sign.Write(append(data, []byte(secretKey)...))
	return hex.EncodeToString(sign.Sum(nil))
}
