package utils

import (
	"errors"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type Claims struct {
	UserID uuid.UUID `json:"user_id"`
	Role   string    `json:"role"`
	jwt.RegisteredClaims
}

// GenerateTokens creates both the 2-hour access token and a 7-day refresh token
func GenerateTokens(userID uuid.UUID, role string) (string, string, error) {
	secret := []byte(os.Getenv("JWT_SECRET"))

	// 1. Access Token - Set to exactly 2 hours for your inactivity requirement
	atClaims := Claims{
		UserID: userID,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(2 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	accessToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, atClaims).SignedString(secret)
	if err != nil {
		return "", "", err
	}

	// 2. Refresh Token - Used to get a new access token without re-entering password
	rtClaims := jwt.RegisteredClaims{
		Subject:   userID.String(),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(7 * 24 * time.Hour)),
	}
	refreshToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, rtClaims).SignedString(secret)

	return accessToken, refreshToken, err
}

// ValidateToken - Make sure the 'V' is CAPITALIZED to fix your compiler error
func ValidateToken(tokenStr string) (*Claims, error) {
	secret := []byte(os.Getenv("JWT_SECRET"))
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		return secret, nil
	})

	if err != nil || !token.Valid {
		return nil, errors.New("session expired")
	}

	claims, ok := token.Claims.(*Claims)
	if !ok {
		return nil, errors.New("invalid claims")
	}
	return claims, nil
}
