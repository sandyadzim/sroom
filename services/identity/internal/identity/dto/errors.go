package dto

import (
	"errors"
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
)

func FormatRegisterErrors(err error) map[string]string {
	details := make(map[string]string)

	if errors.Is(err, ErrPasswordTooLong) {
		details["password"] = "password must not exceed 72 bytes"
		return details
	}

	validationErrs, ok := errors.AsType[validator.ValidationErrors](err)
	if !ok {
		return details
	}

	for _, fieldErr := range validationErrs {
		field := strings.ToLower(fieldErr.Field())

		switch fieldErr.Field() {
		case "FullName":
			field = "full_name"
		}

		switch fieldErr.Tag() {
		case "required":
			details[field] = fmt.Sprintf("%s is required", field)

		case "email":
			details[field] = "email must be a valid email address"

		case "min":
			details[field] = fmt.Sprintf(
				"%s must contain at least %s characters",
				field,
				fieldErr.Param(),
			)

		case "max":
			details[field] = fmt.Sprintf(
				"%s must not exceed %s characters",
				field,
				fieldErr.Param(),
			)

		default:
			details[field] = fmt.Sprintf("%s is invalid", field)
		}
	}

	return details
}

func FormatLoginErrors(
	validationErrs validator.ValidationErrors,
) map[string]string {
	details := make(map[string]string)

	for _, fieldErr := range validationErrs {
		switch fieldErr.Field() {
		case "Email":
			switch fieldErr.Tag() {
			case "required":
				details["email"] = "email is required"
			case "email":
				details["email"] = "email must be a valid email address"
			case "max":
				details["email"] = "email must not exceed 255 characters"
			default:
				details["email"] = "email is invalid"
			}

		case "Password":
			switch fieldErr.Tag() {
			case "required":
				details["password"] = "password is required"
			default:
				details["password"] = "password is invalid"
			}
		}
	}

	return details
}
