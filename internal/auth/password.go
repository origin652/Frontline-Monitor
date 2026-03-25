package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
)

const (
	MinPasswordLength = 8
	hashIterations    = 120000
	hashSaltBytes     = 16
	hashKeyBytes      = 32
)

func HashPassword(password string) (string, error) {
	salt := make([]byte, hashSaltBytes)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("generate salt: %w", err)
	}
	key := derivePBKDF2SHA256([]byte(password), salt, hashIterations, hashKeyBytes)
	return fmt.Sprintf(
		"pbkdf2-sha256$%d$%s$%s",
		hashIterations,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(key),
	), nil
}

func ComparePasswordHash(encoded, password string) bool {
	parts := strings.Split(encoded, "$")
	if len(parts) != 4 || parts[0] != "pbkdf2-sha256" {
		return false
	}
	iterations, err := strconv.Atoi(parts[1])
	if err != nil || iterations <= 0 {
		return false
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[2])
	if err != nil {
		return false
	}
	expected, err := base64.RawStdEncoding.DecodeString(parts[3])
	if err != nil {
		return false
	}
	actual := derivePBKDF2SHA256([]byte(password), salt, iterations, len(expected))
	return subtle.ConstantTimeCompare(actual, expected) == 1
}

func GenerateSessionID() (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("generate session id: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

func derivePBKDF2SHA256(password, salt []byte, iterations, keyLen int) []byte {
	hLen := 32
	blocks := (keyLen + hLen - 1) / hLen
	output := make([]byte, 0, blocks*hLen)

	for block := 1; block <= blocks; block++ {
		u := pbkdf2Block(password, salt, iterations, block)
		output = append(output, u...)
	}

	return output[:keyLen]
}

func pbkdf2Block(password, salt []byte, iterations, block int) []byte {
	mac := hmac.New(sha256.New, password)
	mac.Write(salt)
	mac.Write([]byte{
		byte(block >> 24),
		byte(block >> 16),
		byte(block >> 8),
		byte(block),
	})
	sum := mac.Sum(nil)
	result := append([]byte(nil), sum...)
	for i := 1; i < iterations; i++ {
		mac = hmac.New(sha256.New, password)
		mac.Write(sum)
		sum = mac.Sum(nil)
		for j := range result {
			result[j] ^= sum[j]
		}
	}
	return result
}
