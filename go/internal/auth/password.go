package auth

import "golang.org/x/crypto/bcrypt"

// HashPassword returns a bcrypt hash of the password (used by cmd/hash-password to
// pre-compute values for the seed SQL).
func HashPassword(password string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(b), err
}

// VerifyPassword reports whether password matches the bcrypt hash.
func VerifyPassword(password, hash string) (bool, error) {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil, nil
}
