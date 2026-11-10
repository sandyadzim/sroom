package dto

type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" validate:"required"`
	NewPassword     string `json:"new_password" validate:"required,min=8"`
}

func ValidateChangePassword(req ChangePasswordRequest) error {
	if err := validate.Struct(req); err != nil {
		return err
	}
	if len(req.NewPassword) > 72 {
		return ErrPasswordTooLong
	}
	return nil
}
