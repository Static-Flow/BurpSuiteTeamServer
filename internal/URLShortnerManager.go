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
	port       string
	host       string
}

func (shortenedUrls *ShortenedUrls) HandleShortUrl(ctx *fasthttp.RequestCtx) {
	switch string(ctx.Method()) {
	case http.MethodGet:
		shortId := string(ctx.QueryArgs().Peek("id"))

		if len(shortId) < 1 {
			log.Println("Url Param 'id' is missing")
			ctx.Error("Improper id", fasthttp.StatusBadRequest)
			return
		}

		if burpRequest := shortenedUrls.getShortenedURL(shortId); burpRequest != nil {
			burpRequestJson, err := json.Marshal(burpRequest)
			if err != nil {
				ctx.Error(err.Error(), http.StatusInternalServerError)
				return
			}

			ctx.Response.Header.Add("Content-Type", "application/json")
			_, _ = ctx.Write(burpRequestJson)
		} else {
			ctx.Error("Bad Id", fasthttp.StatusBadRequest)
		}

	case http.MethodPost:

		key := ctx.QueryArgs().Peek("key")

		if key == nil {
			log.Println("Url Param 'key' is missing")
			ctx.Response.SetStatusCode(http.StatusBadRequest)
			ctx.SetBody([]byte("Improper key"))
			return
		}
		log.Printf("User supplied key: %s\n", key)
		if string(key) == shortenedUrls.apiKey {
			var burpRequest = BurpRequestResponse{}
			if err := json.Unmarshal(ctx.PostBody(), &burpRequest); err != nil {
				ctx.Response.SetStatusCode(http.StatusBadRequest)
				ctx.SetBody([]byte("Improper JSON"))
				return
			}
			newId := shortenedUrls.addNewShortenURL(burpRequest)
			accessURL := "https://" + shortenedUrls.host + ":" + shortenedUrls.port + "/shortener?id=" + newId
			log.Println("POST: " + accessURL)
			base64Text := make([]byte, base64.StdEncoding.EncodedLen(len(accessURL)))
			base64.StdEncoding.Encode(base64Text, []byte(accessURL))
			ctx.Response.SetStatusCode(http.StatusOK)
			ctx.SetBody(base64Text)
			return
		} else {
			ctx.Response.SetStatusCode(http.StatusBadRequest)
			ctx.SetBody([]byte("No."))
		}
	default:
		ctx.Error("Unsupported method", fasthttp.StatusMethodNotAllowed)
	}
}

func NewShortenedUrls(port string, host string) *ShortenedUrls {
	manager := &ShortenedUrls{
		make(map[string]BurpRequestResponse),
		rand.New(
			rand.NewSource(time.Now().UnixNano())),
		"",
		port,
		host,
	}
	manager.apiKey = manager.genString()

	go func() {
		if err := fasthttp.ListenAndServe(":"+port, manager.HandleShortUrl); err != nil {
			log.Printf("could not start shortener service: %s\n", err)
		}
	}()

	return manager
}

func (shortenedUrls *ShortenedUrls) genString() string {
	b := make([]byte, 20)
	for i := range b {
		b[i] = charset[shortenedUrls.seededRand.Intn(len(charset))]
	}
	return string(b)
}

func (shortenedUrls *ShortenedUrls) addNewShortenURL(response BurpRequestResponse) string {
	id := shortenedUrls.genString()
	shortenedUrls.urls[id] = response
	return id
}

func (shortenedUrls *ShortenedUrls) getShortenedURL(id string) *BurpRequestResponse {
	if burpRequest, ok := shortenedUrls.urls[id]; ok {
		return &burpRequest
	}
	return nil
}

func (shortenedUrls *ShortenedUrls) setUrlShortenerApiKey(key string) {
	shortenedUrls.apiKey = key
}

func (shortenedUrls *ShortenedUrls) getUrlShortenerApiKey() string {
	return shortenedUrls.apiKey
}
