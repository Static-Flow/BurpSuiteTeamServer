package chatapi

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"io"
	"log"
	"math/big"
	"strings"
)

type AESCrypter struct {
	aesKey string
}

/*
We don't trust users to make secure passwords so we always generate one
*/
func NewAESCrypter() *AESCrypter {
	return &AESCrypter{
		aesKey: randString(32),
	}
}

func (a AESCrypter) Encrypt(plaintext string) string {
	block, err := aes.NewCipher([]byte(a.aesKey))
	if err != nil {
		panic(err.Error())
	}
	ciphertext := make([]byte, aes.BlockSize+len(plaintext))
	iv := ciphertext[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		panic(err)
	}

	stream := cipher.NewCFBEncrypter(block, iv)
	stream.XORKeyStream(ciphertext[aes.BlockSize:], []byte(plaintext))
	return base64.StdEncoding.EncodeToString(ciphertext)
}

func (a AESCrypter) Decrypt(ct string) string {
	encryptedData, _ := base64.StdEncoding.DecodeString(ct)
	if len(encryptedData) > aes.BlockSize {
		nonce := encryptedData[:aes.BlockSize]
		ciphertext := encryptedData[aes.BlockSize:]
		block, err := aes.NewCipher([]byte(a.aesKey))
		if err != nil {
			panic(err.Error())
		}
		mode := cipher.NewCFBDecrypter(block, nonce)
		mode.XORKeyStream(ciphertext, ciphertext)
		return string(ciphertext)
	} else {
		return ""
	}
}

func randString(length int) string {
	chars := []byte("ABCDEFGHIJKLMNOPQRSTUVWXYZ" +
		"abcdefghijklmnopqrstuvwxyz0123456789" +
		"!@#$%^&*()_+-=[]{}\\|/.,<>;:'\"")
	var b strings.Builder
	b.Grow(length)
	for i := 0; i < length; i++ {
		randInt, _ := rand.Int(rand.Reader, big.NewInt(int64(len(chars))))
		b.WriteByte(chars[randInt.Int64()])
	}
	log.Printf("This is the server key that clients need to login: %s", b.String())
	return b.String()
}
