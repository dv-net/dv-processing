package constants

type TronTransferKind string

const (
	TronTransferKindBurnTRX       TronTransferKind = "burntrx"
	TronTransferKindResources     TronTransferKind = "resources"
	TronTransferKindCloudDelegate TronTransferKind = "cloud_delegate"
)

func (t TronTransferKind) String() string { return string(t) }

// Validate
func (t TronTransferKind) Valid() bool {
	switch t {
	case TronTransferKindBurnTRX,
		TronTransferKindResources,
		TronTransferKindCloudDelegate:
		return true
	}

	return false
}
