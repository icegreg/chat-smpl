package validator

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testStruct struct {
	Email    string `validate:"required,email"`
	Username string `validate:"required,min=3,max=50"`
	Role     string `validate:"required,role"`
}

func TestValidate_Success(t *testing.T) {
	s := testStruct{
		Email:    "test@example.com",
		Username: "testuser",
		Role:     "user",
	}

	err := Validate(s)
	assert.NoError(t, err)
}

func TestValidate_RequiredFields(t *testing.T) {
	s := testStruct{}

	err := Validate(s)
	require.Error(t, err)

	var validationErrs ValidationErrors
	require.ErrorAs(t, err, &validationErrs)
	assert.Len(t, validationErrs, 3)
}

func TestValidate_InvalidEmail(t *testing.T) {
	s := testStruct{
		Email:    "invalid-email",
		Username: "testuser",
		Role:     "user",
	}

	err := Validate(s)
	require.Error(t, err)

	var validationErrs ValidationErrors
	require.ErrorAs(t, err, &validationErrs)
	assert.Len(t, validationErrs, 1)
	assert.Equal(t, "email", validationErrs[0].Field)
	assert.Contains(t, validationErrs[0].Message, "valid email")
}

func TestValidate_UsernameTooShort(t *testing.T) {
	s := testStruct{
		Email:    "test@example.com",
		Username: "ab",
		Role:     "user",
	}

	err := Validate(s)
	require.Error(t, err)

	var validationErrs ValidationErrors
	require.ErrorAs(t, err, &validationErrs)
	assert.Len(t, validationErrs, 1)
	assert.Equal(t, "username", validationErrs[0].Field)
}

func TestValidate_InvalidRole(t *testing.T) {
	s := testStruct{
		Email:    "test@example.com",
		Username: "testuser",
		Role:     "invalid",
	}

	err := Validate(s)
	require.Error(t, err)

	var validationErrs ValidationErrors
	require.ErrorAs(t, err, &validationErrs)
	assert.Len(t, validationErrs, 1)
	assert.Equal(t, "role", validationErrs[0].Field)
}

func TestValidate_ValidRoles(t *testing.T) {
	validRoles := []string{"owner", "moderator", "user", "guest"}

	for _, role := range validRoles {
		t.Run(role, func(t *testing.T) {
			s := testStruct{
				Email:    "test@example.com",
				Username: "testuser",
				Role:     role,
			}
			err := Validate(s)
			assert.NoError(t, err)
		})
	}
}

type chatTypeStruct struct {
	Type string `validate:"required,chat_type"`
}

func TestValidate_ChatType(t *testing.T) {
	validTypes := []string{"private", "group", "channel"}

	for _, chatType := range validTypes {
		t.Run(chatType, func(t *testing.T) {
			s := chatTypeStruct{Type: chatType}
			err := Validate(s)
			assert.NoError(t, err)
		})
	}

	t.Run("invalid", func(t *testing.T) {
		s := chatTypeStruct{Type: "invalid"}
		err := Validate(s)
		assert.Error(t, err)
	})
}

type participantRoleStruct struct {
	Role string `validate:"required,participant_role"`
}

func TestValidate_ParticipantRole(t *testing.T) {
	validRoles := []string{"admin", "member", "readonly"}

	for _, role := range validRoles {
		t.Run(role, func(t *testing.T) {
			s := participantRoleStruct{Role: role}
			err := Validate(s)
			assert.NoError(t, err)
		})
	}

	t.Run("invalid", func(t *testing.T) {
		s := participantRoleStruct{Role: "invalid"}
		err := Validate(s)
		assert.Error(t, err)
	})
}

func TestToSnakeCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Username", "username"},
		{"UserID", "user_i_d"},
		{"firstName", "first_name"},
		{"simple", "simple"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := toSnakeCase(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestValidationErrors_Error(t *testing.T) {
	errs := ValidationErrors{
		{Field: "email", Message: "is required"},
		{Field: "username", Message: "is required"},
	}

	expected := "email: is required; username: is required"
	assert.Equal(t, expected, errs.Error())
}

func TestGet(t *testing.T) {
	v := Get()
	assert.NotNil(t, v)
}
