package tron

import (
	"crypto/ecdsa"
	"crypto/sha256"
	"fmt"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	"google.golang.org/protobuf/proto"
)

// SignTransaction signs the transaction with the given private key
func (t *Tron) SignTransaction(tx *core.Transaction, privateKey *ecdsa.PrivateKey) error {
	if tx == nil {
		return fmt.Errorf("empty tron tx")
	}

	rawData, err := proto.Marshal(tx.GetRawData())
	if err != nil {
		return err
	}

	h256h := sha256.New()
	if _, err := h256h.Write(rawData); err != nil {
		return err
	}

	signature, err := crypto.Sign(h256h.Sum(nil), privateKey)
	if err != nil {
		return err
	}

	tx.Signature = append(tx.Signature, signature)

	return nil
}
