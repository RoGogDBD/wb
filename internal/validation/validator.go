package validation

import (
	"regexp"

	"github.com/go-playground/validator/v10"
)

var (
	phoneRU = regexp.MustCompile(`^(\+7|8)\d{10}$`)
	zipRU   = regexp.MustCompile(`^\d{6}$`)
)

func New() *validator.Validate {
	v := validator.New()
	_ = v.RegisterValidation("phone_ru", func(fl validator.FieldLevel) bool {
		return phoneRU.MatchString(fl.Field().String())
	})
	_ = v.RegisterValidation("zip_ru", func(fl validator.FieldLevel) bool {
		return zipRU.MatchString(fl.Field().String())
	})
	return v
}
