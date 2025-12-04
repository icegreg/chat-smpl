package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRole_IsValid(t *testing.T) {
	tests := []struct {
		role     Role
		expected bool
	}{
		{RoleOwner, true},
		{RoleModerator, true},
		{RoleUser, true},
		{RoleGuest, true},
		{Role("invalid"), false},
		{Role(""), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.role), func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.role.IsValid())
		})
	}
}

func TestRole_CanWrite(t *testing.T) {
	tests := []struct {
		role     Role
		expected bool
	}{
		{RoleOwner, true},
		{RoleModerator, true},
		{RoleUser, true},
		{RoleGuest, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.role), func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.role.CanWrite())
		})
	}
}

func TestRole_CanModerate(t *testing.T) {
	tests := []struct {
		role     Role
		expected bool
	}{
		{RoleOwner, true},
		{RoleModerator, true},
		{RoleUser, false},
		{RoleGuest, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.role), func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.role.CanModerate())
		})
	}
}

func TestRole_IsAdmin(t *testing.T) {
	tests := []struct {
		role     Role
		expected bool
	}{
		{RoleOwner, true},
		{RoleModerator, false},
		{RoleUser, false},
		{RoleGuest, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.role), func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.role.IsAdmin())
		})
	}
}
