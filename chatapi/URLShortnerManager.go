package chatapi

import (
	"math/rand"
	"time"
)

const charset = "abcdefghijklmnopqrstuvwxyz" +
	"ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

const length = 10

type ShortenedUrls struct {
	urls       map[string]BurpRequestResponse
	seededRand *rand.Rand
}

func NewShortenedUrls() *ShortenedUrls {
	return &ShortenedUrls{
		make(map[string]BurpRequestResponse),
		rand.New(
			rand.NewSource(time.Now().UnixNano())),
	}
}

func (shortener *ShortenedUrls) GenString() string {
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[shortener.seededRand.Intn(len(charset))]
	}
	return string(b)
}

func (shortener *ShortenedUrls) AddNewShortenURL(response BurpRequestResponse) string {
	id := shortener.GenString()
	shortener.urls[id] = response
	return id
}

func (shortener *ShortenedUrls) GetShortenedURL(id string) *BurpRequestResponse {
	if burpRequest, ok := shortener.urls[id]; ok {
		return &burpRequest
	}
	return nil
}
