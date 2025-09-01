package madmin

import (
	"fmt"
	"reflect"
)

func (o *Service) validateReq(request any) error {
	if reflect.TypeOf(request).Kind() != reflect.Ptr && reflect.TypeOf(request).Elem().Kind() != reflect.Struct {
		return fmt.Errorf("invalid request type. must be a pointer to a struct")
	}
	if err := o.validator.Struct(request); err != nil {
		return fmt.Errorf("request validation error: %w", err)
	}
	return nil
}
