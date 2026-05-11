package utils

import (
	"errors"
	"regexp"
	"strings"
)

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

func CleanEmail(email string) (string, error) {
	email = strings.TrimSpace(email)
	email = strings.ToLower(email) // Standardize to lowercase
	if !emailRegex.MatchString(email) {
		return "", errors.New("invalid email format")
	}
	return email, nil
}

func ValidatePassword(pass string) error {
	if len(pass) < 8 {
		return errors.New("password must be at least 8 characters long")
	}
	return nil
}
