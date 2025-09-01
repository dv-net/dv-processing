package utils //nolint:revive,nolintlint

import "github.com/goccy/go-json"

// Pointer return pointer for value
func Pointer[T any](v T) *T { return &v }

func JSONToStruct[T any](data any) (T, error) {
	var res T
	result, err := json.Marshal(data)
	if err != nil {
		return res, err
	}

	if err := json.Unmarshal(result, &res); err != nil {
		return res, err
	}

	return res, nil
}

func BytesToStruct[T any](data []byte) (T, error) {
	var res T
	if err := json.Unmarshal(data, &res); err != nil {
		return res, err
	}
	return res, nil
}

func StructToBytes[T any](data T) ([]byte, error) {
	return json.Marshal(data)
}
