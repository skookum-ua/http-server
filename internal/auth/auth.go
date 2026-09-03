package auth

import (
	"log"
	"runtime"

	"github.com/alexedwards/argon2id"
)

func HashPassword(password string) (string, error) {
	params := &argon2id.Params{
		Memory:      128 * 1024,
		Iterations:  4,
		Parallelism: uint8(runtime.NumCPU()),
		SaltLength:  16,
		KeyLength:   32,
	}
	hash, err := argon2id.CreateHash(password, params)
	if err != nil {
		log.Printf("Can't hash password: %v", err)
		return "", err
	}
	return hash, nil
}

func CheckPasswordHash(password, hash string) (bool, error) {
	return argon2id.ComparePasswordAndHash(password, hash)
}
