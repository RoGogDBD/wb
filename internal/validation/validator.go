// Package validation содержит валидаторы входных данных.
package validation

import (
	"fmt"
	"regexp"

	"github.com/go-playground/validator/v10"
)

var (
	phoneRU = regexp.MustCompile(`^(\+7|8)\d{10}$`)
	zipRU   = regexp.MustCompile(`^\d{6}$`)
)

// New создает валидатор с пользовательскими правилами.
func New() (*validator.Validate, error) {
	v := validator.New()
	if err := v.RegisterValidation("phone_ru", func(fl validator.FieldLevel) bool {
		return phoneRU.MatchString(fl.Field().String())
	}); err != nil {
		return nil, fmt.Errorf("register phone_ru validation: %w", err)
	}
	if err := v.RegisterValidation("zip_ru", func(fl validator.FieldLevel) bool {
		return zipRU.MatchString(fl.Field().String())
	}); err != nil {
		return nil, fmt.Errorf("register zip_ru validation: %w", err)
	}
	return v, nil
}

// MustNew возвращает валидатор или паникует при ошибке инициализации.
func MustNew() *validator.Validate {
	v, err := New()
	if err != nil {
		panic(err)
	}
	return v
}
