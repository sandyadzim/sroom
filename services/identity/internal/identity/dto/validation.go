package dto

import (
	"errors"

	"github.com/go-playground/validator/v10"
)

var validate = validator.New()

var ErrPasswordTooLong = errors.New(
	"password must not exceed 72 bytes",
)

var ErrEmptyUpdate = errors.New("at least one field is required")

func ValidateCreateAdmin(req CreateAdminRequest) error {
	if err := validate.Struct(req); err != nil {
		return err
	}
	return ValidateRegister(RegisterRequest{FullName: req.FullName, Email: req.Email, Password: req.Password})
}

func ValidateUpdateAdmin(req UpdateAdminRequest) error {
	if req.FullName == nil && req.Email == nil && req.Password == nil && req.Status == nil {
		return ErrEmptyUpdate
	}
	if err := validate.Struct(req); err != nil {
		return err
	}
	if req.Password != nil && len(*req.Password) > 72 {
		return ErrPasswordTooLong
	}
	return nil
}

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
