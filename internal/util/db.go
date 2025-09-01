package util //nolint:nolintlint,revive

func ConvertListToAny[T any](list []T) []any {
	result := make([]any, len(list))
	for idx, v := range list {
		result[idx] = v
	}
	return result
}
