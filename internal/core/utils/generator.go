package utils

import (
	"crypto/rand"
	"math/big"
)

func GenerateJobRef() (string, error) {
	// Excludes O, 0, I, 1 to prevent human typing errors during payment
	const charset = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	length := 6
	result := make([]byte, length)
	for i := 0; i < length; i++ {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			return "", err
		}
		result[i] = charset[num.Int64()]
	}
	return string(result), nil
}
