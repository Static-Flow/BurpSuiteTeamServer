package internal

import (
	"encoding/base64"
	"encoding/json"
	"github.com/valyala/fasthttp"
	"log"
	"math/rand"
	"net/http"
	"time"
)

const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

type ShortenedUrls struct {
	urls       map[string]BurpRequestResponse
	seededRand *rand.Rand
	apiKey     string
}

func HandleShortUrl(ctx *fasthttp.RequestCtx, hub *Hub) {
	switch string(ctx.Method()) {
	case http.MethodGet:
		shortId := string(ctx.QueryArgs().Peek("id"))

		if len(shortId) < 1 {
			log.Println("Url Param 'id' is missing")
			ctx.Error("Improper id", fasthttp.StatusBadRequest)
			return
		}

		if burpRequest := hub.GetUrlShortener().GetShortenedURL(shortId); burpRequest != nil {
			burpRequestJson, err := json.Marshal(burpRequest)
			if err != nil {
				ctx.Error(err.Error(), http.StatusInternalServerError)
				return
			}

			ctx.Response.Header.Add("Content-Type", "application/json")
			ctx.Write(burpRequestJson)
		} else {
			ctx.Error("Bad Id", fasthttp.StatusBadRequest)
		}

	case http.MethodPost:

		key := string(ctx.QueryArgs().Peek("key"))

		if len(key) < 1 {
			log.Println("Url Param 'key' is missing")
			ctx.Error("Improper key", fasthttp.StatusBadRequest)
			return
		}

		log.Println("User supplied key: " + key)
		if key == hub.GetUrlShortenerApiKey() {
			var burpRequest = BurpRequestResponse{}

			if err := json.Unmarshal(ctx.PostBody(), &burpRequest); err != nil {
				ctx.Error("Improper JSON", fasthttp.StatusBadRequest)
			}
			newId := hub.GetUrlShortener().AddNewShortenURL(burpRequest)
			shortenedUrl := hub.GetShortenerUrl(newId)
			log.Println("POST: " + shortenedUrl)
			base64Text := make([]byte, base64.StdEncoding.EncodedLen(len(shortenedUrl)))
			base64.StdEncoding.Encode(base64Text, []byte(shortenedUrl))
			ctx.Write(base64Text)
		} else {
			ctx.Error("Wrong.", http.StatusBadRequest)
		}
	default:
		ctx.Error("Unsupported method", fasthttp.StatusMethodNotAllowed)
	}
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
