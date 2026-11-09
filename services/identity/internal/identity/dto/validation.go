package dto

import (
	"errors"

	"github.com/go-playground/validator/v10"
)

var validate = validator.New()

var ErrPasswordTooLong = errors.New(
	"password must not exceed 72 bytes",
)

func ValidateRegister(req RegisterRequest) error {
	if err := validate.Struct(req); err != nil {
		return err
	}

	if len(req.Password) > 72 {
		return ErrPasswordTooLong
	}

	return nil
}

func ValidateLogin(req LoginRequest) error {
	return validate.Struct(req)
}
