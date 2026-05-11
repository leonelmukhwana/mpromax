package utils

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"io"
	"os"
	"regexp"
	"strings"
)

// encryption
func GetEncryptionKey() ([]byte, error) {
	key := os.Getenv("ENCRYPTION_KEY")
	if len(key) != 32 {
		return nil, errors.New("ENCRYPTION_KEY must be exactly 32 characters")
	}
	return []byte(key), nil
}

func Encrypt(plainText string) (string, error) {
	key, err := GetEncryptionKey()
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	cipherText := gcm.Seal(nonce, nonce, []byte(plainText), []byte("client-profile"))
	return base64.StdEncoding.EncodeToString(cipherText), nil
}

func Decrypt(cryptoText string) (string, error) {
	key, err := GetEncryptionKey()
	if err != nil {
		return "", err
	}

	data, err := base64.StdEncoding.DecodeString(cryptoText)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", errors.New("ciphertext too short")
	}

	nonce, cipherText := data[:nonceSize], data[nonceSize:]

	plainText, err := gcm.Open(nil, nonce, cipherText, []byte("client-profile"))
	if err != nil {
		return "", err
	}

	return string(plainText), nil
}

func HashSHA256(value string) string {
	hash := sha256.Sum256([]byte(value))
	return hex.EncodeToString(hash[:])
}

// valoiadtes names
func SanitizeName(name string) (string, error) {
	name = strings.TrimSpace(name)

	if len(name) < 2 || len(name) > 100 {
		return "", errors.New("invalid name length")
	}

	// Allow letters, spaces, hyphen, apostrophe
	reg := regexp.MustCompile(`^[a-zA-Z\s'-]+$`)
	if !reg.MatchString(name) {
		return "", errors.New("name contains invalid characters")
	}

	// Normalize multiple spaces
	name = strings.Join(strings.Fields(name), " ")

	// Title case
	formatted := cases.Title(language.English).String(strings.ToLower(name))

	return formatted, nil
}

//validate phone number inputs

func ValidatePhone(phone string) (string, error) {
	// Remove spaces
	phone = strings.ReplaceAll(phone, " ", "")

	// Case 1: +254XXXXXXXXX
	if strings.HasPrefix(phone, "+254") {
		phone = "0" + phone[4:]
	}

	// Case 2: 254XXXXXXXXX
	if strings.HasPrefix(phone, "254") {
		phone = "0" + phone[3:]
	}

	// Now enforce local format: 07XXXXXXXX or 01XXXXXXXX
	reg := regexp.MustCompile(`^(07|01)\d{8}$`)
	if !reg.MatchString(phone) {
		return "", errors.New("phone must be 07XXXXXXXX or 01XXXXXXXX")
	}

	return phone, nil
}
