package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"io"
	mathRand "math/rand"
	"time"
)

const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

// GenerateKey Generates a random key
func GenerateKey() string {
	seededRand := mathRand.New(mathRand.NewSource(time.Now().UnixNano()))
	key := make([]byte, 32)
	for i := range key {
		key[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(key)
}

func deriveSHA256Key(key []byte) []byte {
	hash := sha256.Sum256(key)
	return hash[:]
}

// Encrypt encrypts a plaintext with a passphrase and return a base64 encoded string
func Encrypt(plainText string, key []byte) (string, error) {
	block, err := aes.NewCipher(deriveSHA256Key(key))
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
func Decrypt(encryptedText string, key []byte) (string, error) {
	ciphertext, err := base64.StdEncoding.DecodeString(encryptedText)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(deriveSHA256Key(key))
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
