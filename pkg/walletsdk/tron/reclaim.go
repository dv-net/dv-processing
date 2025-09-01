package tron

import (
	"context"
	"fmt"

	"github.com/dv-net/dv-processing/pkg/avalidator"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/api"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
)

type ReclaimResourceParams struct {
	FromAddress  string
	FromSequence uint32
	Mnemonic     string
	PassPhrase   string
	ToAddress    string
	ResourceType core.ResourceCode
}

// Validate
func (p ReclaimResourceParams) Validate() error {
	if p.FromAddress == "" || !avalidator.ValidateTronAddress(p.FromAddress) {
		return fmt.Errorf("from address is not valid")
	}

	if p.ToAddress == "" || !avalidator.ValidateTronAddress(p.ToAddress) {
		return fmt.Errorf("to address is not valid")
	}

	if p.Mnemonic == "" {
		return fmt.Errorf("mnemonic is not valid")
	}

	// if p.PassPhrase == "" {
	// 	return fmt.Errorf("pass phrase is not valid")
	// }

	if p.ResourceType != core.ResourceCode_BANDWIDTH && p.ResourceType != core.ResourceCode_ENERGY {
		return fmt.Errorf("resource type is not valid")
	}

	return nil
}

func (t *Tron) ReclaimResource(ctx context.Context, params ReclaimResourceParams) (*api.TransactionExtention, error) {
	if err := params.Validate(); err != nil {
		return nil, fmt.Errorf("validate params: %w", err)
	}

	_, frozenResources, err := t.StakedResources(ctx, params.FromAddress)
	if err != nil {
		return nil, fmt.Errorf("get staked resources: %w", err)
	}

	address, priv, _, err := WalletPubKeyHash(params.Mnemonic, params.PassPhrase, params.FromSequence)
	if err != nil {
		return nil, err
	}

	if address != params.FromAddress {
		return nil, fmt.Errorf("address %s is not valid, expected %s", address, params.FromAddress)
	}

	var tx *api.TransactionExtention
	for _, frozenResource := range frozenResources {
		if frozenResource.DelegateTo != params.ToAddress ||
			frozenResource.Type != params.ResourceType ||
			frozenResource.Amount <= 0 {
			continue
		}

		tx, err = t.Node().UnDelegateResource(params.FromAddress, params.ToAddress, params.ResourceType, frozenResource.Amount)
		if err != nil {
			return nil, fmt.Errorf("undelegate resource: %w", err)
		}

		if !tx.Result.Result {
			return nil, fmt.Errorf("create reclaim tx error: %s", string(tx.Result.Message))
		}

		if err := t.SignTransaction(tx.GetTransaction(), priv.ToECDSA()); err != nil {
			return nil, fmt.Errorf("sign transaction: %w", err)
		}

		if _, err := t.Node().Broadcast(tx.GetTransaction()); err != nil {
			return nil, fmt.Errorf("broadcast transaction: %w", err)
		}
	}

	if tx == nil {
		return nil, fmt.Errorf("no frozen resources found for address %s", params.ToAddress)
	}

	return tx, nil
}
