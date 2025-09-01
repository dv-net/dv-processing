package tron

const TrxAssetIdentifier = "trx"

const (
	BlackHoleAddress = "T9yD14Nj9j7xAB4dbGeiX9h8unkKHxuWwb"
)

type MethodSignature string

func (o MethodSignature) String() string { return string(o) }

const (
	ActivatorActivateMethodSignature MethodSignature = "1c5a9d9c"
)
