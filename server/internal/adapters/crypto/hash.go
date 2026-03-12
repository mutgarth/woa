package crypto

import (
	"crypto/sha256"
	"encoding/hex"

	"golang.org/x/crypto/bcrypt"
)

type HashService struct{}

func NewHashService() *HashService { return &HashService{} }

func (h *HashService) HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

func (h *HashService) CheckPassword(hash, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}

func (h *HashService) HashAPIKey(key string) string {
	h256 := sha256.Sum256([]byte(key))
	return hex.EncodeToString(h256[:])
}
