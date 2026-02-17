package processedincidents

import "errors"

var (
	errBlockchainEmpty = errors.New("blockchain is empty")
	errIncidentIDEmpty = errors.New("incident ID is empty")
)
