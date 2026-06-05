package auth

import (
	"golang.org/x/crypto/bcrypt"
)

// BcryptCost is the work factor used for all password hashing in the app.
// 12 gives meaningfully better resistance against GPU cracking than the
// bcrypt default (10) while staying well under the 250ms budget per login.
const BcryptCost = 12

func HashPassword(password string) (string, error) {
	if password == "" {
		return "", nil
	}
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), BcryptCost)
	return string(bytes), err
}

func CheckPassword(password, hash string) bool {
	if hash == "" {
		return false // Empty hash is never valid; callers should check HasPassword first
	}
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}
