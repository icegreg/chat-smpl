package validator

import (
	"errors"
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
)

var validate *validator.Validate

func init() {
	validate = validator.New()

	// Register custom validations
	_ = validate.RegisterValidation("role", validateRole)
	_ = validate.RegisterValidation("chat_type", validateChatType)
	_ = validate.RegisterValidation("participant_role", validateParticipantRole)
}

func validateRole(fl validator.FieldLevel) bool {
	role := fl.Field().String()
	validRoles := map[string]bool{
		"owner":     true,
		"moderator": true,
		"user":      true,
		"guest":     true,
	}
	return validRoles[role]
}

func validateChatType(fl validator.FieldLevel) bool {
	chatType := fl.Field().String()
	validTypes := map[string]bool{
		"private": true,
		"group":   true,
		"channel": true,
	}
	return validTypes[chatType]
}

func validateParticipantRole(fl validator.FieldLevel) bool {
	role := fl.Field().String()
	validRoles := map[string]bool{
		"admin":    true,
		"member":   true,
		"readonly": true,
	}
	return validRoles[role]
}

type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

type ValidationErrors []ValidationError

func (e ValidationErrors) Error() string {
	var msgs []string
	for _, err := range e {
		msgs = append(msgs, err.Error())
	}
	return strings.Join(msgs, "; ")
}

func Validate(s interface{}) error {
	err := validate.Struct(s)
	if err == nil {
		return nil
	}

	var validationErrs validator.ValidationErrors
	if errors.As(err, &validationErrs) {
		var errs ValidationErrors
		for _, e := range validationErrs {
			errs = append(errs, ValidationError{
				Field:   toSnakeCase(e.Field()),
				Message: getErrorMessage(e),
			})
		}
		return errs
	}

	return err
}

func getErrorMessage(e validator.FieldError) string {
	switch e.Tag() {
	case "required":
		return "is required"
	case "email":
		return "must be a valid email address"
	case "min":
		return fmt.Sprintf("must be at least %s characters", e.Param())
	case "max":
		return fmt.Sprintf("must be at most %s characters", e.Param())
	case "uuid":
		return "must be a valid UUID"
	case "role":
		return "must be one of: owner, moderator, user, guest"
	case "chat_type":
		return "must be one of: private, group, channel"
	case "participant_role":
		return "must be one of: admin, member, readonly"
	case "oneof":
		return fmt.Sprintf("must be one of: %s", e.Param())
	default:
		return fmt.Sprintf("failed validation: %s", e.Tag())
	}
}

func toSnakeCase(s string) string {
	var result strings.Builder
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result.WriteRune('_')
		}
		result.WriteRune(r)
	}
	return strings.ToLower(result.String())
}

func Get() *validator.Validate {
	return validate
}
