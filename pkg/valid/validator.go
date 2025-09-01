package valid

import (
	"errors"
	"reflect"
	"regexp"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/shopspring/decimal"
)

func New(opts ...validator.Option) *validator.Validate {
	vl := validator.New(opts...)
	if err := vl.RegisterValidation("ISO8601date", isISO8601Date); err != nil {
		panic(err)
	}
	if err := vl.RegisterValidation("nulluuid4", nullUUID); err != nil {
		panic(err)
	}
	if err := vl.RegisterValidation("pgtimestamp", timestamp); err != nil {
		panic(err)
	}
	vl.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		if name == "-" {
			return fld.Name
		}
		return name
	})
	vl.RegisterCustomTypeFunc(func(field reflect.Value) any {
		if valuer, ok := field.Interface().(decimal.Decimal); ok {
			return valuer.String()
		}
		return nil
	}, decimal.Decimal{})
	if err := vl.RegisterValidation("dgt", func(fl validator.FieldLevel) bool {
		data, ok := fl.Field().Interface().(string)
		if !ok {
			return false
		}
		value, err := decimal.NewFromString(data)
		if err != nil {
			return false
		}
		baseValue, err := decimal.NewFromString(fl.Param())
		if err != nil {
			return false
		}
		return value.GreaterThan(baseValue)
	}); err != nil {
		panic(err)
	}
	if err := vl.RegisterValidation("dgte", func(fl validator.FieldLevel) bool {
		data, ok := fl.Field().Interface().(string)
		if !ok {
			return false
		}
		value, err := decimal.NewFromString(data)
		if err != nil {
			return false
		}
		baseValue, err := decimal.NewFromString(fl.Param())
		if err != nil {
			return false
		}
		return value.GreaterThanOrEqual(baseValue)
	}); err != nil {
		panic(err)
	}
	if err := vl.RegisterValidation("dlt", func(fl validator.FieldLevel) bool {
		data, ok := fl.Field().Interface().(string)
		if !ok {
			return false
		}
		value, err := decimal.NewFromString(data)
		if err != nil {
			return false
		}
		baseValue, err := decimal.NewFromString(fl.Param())
		if err != nil {
			return false
		}
		return value.LessThan(baseValue)
	}); err != nil {
		panic(err)
	}
	if err := vl.RegisterValidation("dlte", func(fl validator.FieldLevel) bool {
		data, ok := fl.Field().Interface().(string)
		if !ok {
			return false
		}
		value, err := decimal.NewFromString(data)
		if err != nil {
			return false
		}
		baseValue, err := decimal.NewFromString(fl.Param())
		if err != nil {
			return false
		}
		return value.LessThanOrEqual(baseValue)
	}); err != nil {
		panic(err)
	}

	return vl
}

func isISO8601Date(fl validator.FieldLevel) bool {
	ISO8601DateRegexString := "^(?:[1-9]\\d{3}-(?:(?:0[1-9]|1[0-2])-(?:0[1-9]|1\\d|2[0-8])|(?:0[13-9]|1[0-2])-(?:29|30)|(?:0[13578]|1[02])-31)|(?:[1-9]\\d(?:0[48]|[2468][048]|[13579][26])|(?:[2468][048]|[13579][26])00)-02-29)T(?:[01]\\d|2[0-3]):[0-5]\\d:[0-5]\\d(?:\\.\\d{1,9})?(?:Z|[+-][01]\\d:[0-5]\\d)$"
	ISO8601DateRegex := regexp.MustCompile(ISO8601DateRegexString)
	return ISO8601DateRegex.MatchString(fl.Field().String())
}

func nullUUID(fl validator.FieldLevel) bool {
	if valuer, ok := fl.Field().Interface().(uuid.NullUUID); ok {
		return valuer.Valid
	}

	return false
}

func timestamp(fl validator.FieldLevel) bool {
	if valuer, ok := fl.Field().Interface().(pgtype.Timestamp); ok {
		if valuer.Time.IsZero() {
			return false
		}

		return valuer.Valid
	}

	return false
}

// GetErrorsPath return error paths wuth tag
//
// Example:
//
//	full_name => required
//	params.age => gt=18
func GetErrorsPath(err error) map[string]string {
	res := make(map[string]string)
	errs := new(validator.ValidationErrors)
	if ok := errors.As(err, errs); ok {
		for _, err := range *errs {
			res[strings.Join(strings.Split(err.Namespace(), ".")[1:], ".")] = err.ActualTag()
		}
	}
	return res
}
