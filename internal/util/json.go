package util //nolint:nolintlint,revive

import (
	"fmt"

	"github.com/goccy/go-json"
	"github.com/tidwall/gjson"
)

var ErrGetByPathNotFound = fmt.Errorf("path not found")

func GetByPath[T any](stateData any, path string) (T, error) {
	var res T
	if stateData == nil {
		return res, fmt.Errorf("state data is required")
	}

	if path == "" {
		return res, fmt.Errorf("path is required")
	}

	stateDataBytes, err := json.Marshal(stateData)
	if err != nil {
		return res, fmt.Errorf("marshal state data: %w", err)
	}

	jsonResult := gjson.GetBytes(stateDataBytes, path)

	if !jsonResult.Exists() {
		return res, ErrGetByPathNotFound
	}

	valueBytes, err := json.Marshal(jsonResult.Value())
	if err != nil {
		return res, fmt.Errorf("marshal json value: %w", err)
	}

	if err := json.Unmarshal(valueBytes, &res); err != nil {
		return res, fmt.Errorf("unmarshal json value: %w", err)
	}

	return res, nil
}
