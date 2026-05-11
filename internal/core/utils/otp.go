package utils

import (
	"crypto/rand"
	"fmt"
	"math/big"
)

func GenerateOTP() string {
	// Generates a random number between 000000 and 999999
	n, _ := rand.Int(rand.Reader, big.NewInt(900000))
	return fmt.Sprintf("%06d", n.Add(n, big.NewInt(100000)))
}
