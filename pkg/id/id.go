package id

import (
	"crypto/rand"
	"math/big"
	"strings"
)

const charset = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"

func Generate() string {
	return block() + "-" + block() + "-" + block()
}

func block() string {
	b := make([]byte, 3)
	for i := range b {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			b[i] = charset[0]
		} else {
			b[i] = charset[n.Int64()]
		}
	}
	return string(b)
}

func Valid(id string) bool {
	parts := strings.Split(id, "-")
	if len(parts) != 3 {
		return false
	}
	for _, p := range parts {
		if len(p) != 3 {
			return false
		}
		for _, c := range p {
			if !strings.ContainsRune(charset, c) {
				return false
			}
		}
	}
	return true
}
