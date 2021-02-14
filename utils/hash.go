package utils

import (
	"math/rand"
	"time"
)

const minCountToBeStrong = 4
const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

var seededRand = rand.New(rand.NewSource(time.Now().UnixNano()))

func GenerateStrongHash() string {
	return stringWithCharset(minCountToBeStrong, charset)
}

func GenerateHash(length int) string {
	return stringWithCharset(length, charset)
}

func stringWithCharset(length int, charset string) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}
