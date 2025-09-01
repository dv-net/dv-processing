package doge

import "errors"

var (
	ErrInputAlreadyUsed    = errors.New("input already used")
	ErrOutputAlreadyExists = errors.New("output already exists")
)
