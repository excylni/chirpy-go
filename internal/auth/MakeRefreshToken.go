package auth

import (
	"crypto/rand"
	"encoding/hex"
)

func MakeRefreshToken() string {
	c := make([]byte, 32)
	_, err := rand.Read(c)
	if err != nil {
		panic("crypto/rand failed: " + err.Error())
	}
	return hex.EncodeToString((c))
}