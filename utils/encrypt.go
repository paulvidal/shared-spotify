package utils

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"github.com/shared-spotify/logger"
	"io"
)



var encryptionError = errors.New("Encryption failed")
var decryptionError = errors.New("Decryption failed")

func createHash(key string) string {
	hasher := md5.New()
	hasher.Write([]byte(key))
	return hex.EncodeToString(hasher.Sum(nil))
}

func Encrypt(data []byte, encryptionKey string) ([]byte, error) {
	block, _ := aes.NewCipher([]byte(createHash(encryptionKey)))

	gcm, err := cipher.NewGCM(block)

	if err != nil {
		logger.Logger.Error("Encryption error ", err)
		return nil, encryptionError
	}

	nonce := make([]byte, gcm.NonceSize())

	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		logger.Logger.Error("Encryption error ", err)
		return nil, encryptionError
	}

	ciphertext := gcm.Seal(nonce, nonce, data, nil)

	return ciphertext, nil
}

func Decrypt(data []byte, encryptionKey string) ([]byte, error) {
	key := []byte(createHash(encryptionKey))

	block, err := aes.NewCipher(key)

	if err != nil {
		logger.Logger.Error("Decryption error ", err)
		return nil, decryptionError
	}

	gcm, err := cipher.NewGCM(block)

	if err != nil {
		logger.Logger.Error("Decryption error ", err)
		return nil, decryptionError
	}

	nonceSize := gcm.NonceSize()
	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)

	if err != nil {
		logger.Logger.Error("Decryption error ", err)
		return nil, decryptionError
	}

	return plaintext, nil
}
