package api

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"github.com/OverlayFox/VRC-Stream-Haven/logger"
	"io"
	"os"
)

var Key []byte

func init() {
	evaluationKey := os.Getenv("PASSPHRASE")

	if len(evaluationKey) < 10 {
		logger.HavenLogger.Fatal().Msg("PASSPHRASE not set or shorter than 10 characters.")
	}

	Key = []byte(evaluationKey)
}

// Encrypt encrypts a plaintext with a passphrase and return a base64 encoded string
func Encrypt(plainText string) (string, error) {
	block, err := aes.NewCipher(Key)
	if err != nil {
		return "", err
	}

	ciphertext := make([]byte, aes.BlockSize+len(plainText))
	iv := ciphertext[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return "", err
	}

	stream := cipher.NewCFBEncrypter(block, iv)
	stream.XORKeyStream(ciphertext[aes.BlockSize:], []byte(plainText))
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt decrypts a base64 encoded string with a passphrase and return the plaintext
func Decrypt(encryptedText string) (string, error) {
	ciphertext, err := base64.StdEncoding.DecodeString(encryptedText)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(Key)
	if err != nil {
		return "", err
	}

	if len(ciphertext) < aes.BlockSize {
		return "", errors.New("ciphertext is too short")
	}
	iv := ciphertext[:aes.BlockSize]
	ciphertext = ciphertext[aes.BlockSize:]

	stream := cipher.NewCFBDecrypter(block, iv)
	stream.XORKeyStream(ciphertext, ciphertext)

	return string(ciphertext), nil
}
