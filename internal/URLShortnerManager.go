package internal

import (
	"math/rand"
	"time"
)

const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

type ShortenedUrls struct {
	urls       map[string]BurpRequestResponse
	seededRand *rand.Rand
	apiKey     string
}

func NewShortenedUrls() *ShortenedUrls {
	manager := &ShortenedUrls{
		make(map[string]BurpRequestResponse),
		rand.New(
			rand.NewSource(time.Now().UnixNano())),
		"",
	}
	manager.apiKey = manager.GenString()
	return manager
}

func (shortener *ShortenedUrls) GenString() string {
	b := make([]byte, 20)
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

func (shortener *ShortenedUrls) SetUrlShortenerApiKey(key string) {
	shortener.apiKey = key
}

func (shortener *ShortenedUrls) GetUrlShortenerApiKey() string {
	return shortener.apiKey
}
