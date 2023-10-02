package api

import (
	"github.com/go-playground/validator/v10"
	curr "github.com/jimxshaw/trivial-bank/util/currency"
)

var validCurrency validator.Func = func(fieldLevel validator.FieldLevel) bool {
	if currency, ok := fieldLevel.Field().Interface().(string); ok {
		// Check if currency is supported.
		return curr.IsSupportedCurrency(currency)
	}
	return false
}
