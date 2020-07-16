package main

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/Static-Flow/BurpSuiteTeamServer/chatapi"
	"io/ioutil"
	"log"
	"net/http"
	"os"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	var host = flag.String("host", "localhost", "host for TLS cert. Defaults to localhost")
	var port = flag.String("port", "9999", "http service address")
	var serverPassword = flag.String("serverPassword", "", "password for the server")
	var enableUrlShortener = flag.Bool("enableShortener", false, "Enables the built-in URL shortener")
	serverNameInternal, err := os.Hostname()
	if err != nil {
		fmt.Printf("No hostname, panic: %v\n", err)
		panic(err)
	}
	flag.Parse()
	chatapi.GenCrt(*host)
	hub := chatapi.NewHub(*serverPassword)
	shortendURLs := chatapi.NewShortenedUrls()
	go hub.Run()
	var httpErr error
	if *enableUrlShortener {
		hub.SetUrlShortenerApiKey(shortendURLs.GenString() + shortendURLs.GenString())
		http.HandleFunc("/shortener", func(writer http.ResponseWriter, request *http.Request) {
			switch request.Method {
			case http.MethodGet:

				shortId, ok := request.URL.Query()["id"]

				if !ok || len(shortId[0]) < 1 {
					log.Println("Url Param 'id' is missing")
					http.Error(writer, "Improper id", http.StatusBadRequest)
					return
				}
				id := shortId[0]

				if burpRequest := shortendURLs.GetShortenedURL(id); burpRequest != nil {
					burpRequestJson, err := json.Marshal(burpRequest)
					if err != nil {
						http.Error(writer, err.Error(), http.StatusInternalServerError)
						return
					}

					writer.Header().Set("Content-Type", "application/json")
					writer.Write(burpRequestJson)
				} else {
					http.Error(writer, "Bad Id", http.StatusBadRequest)
				}

			case http.MethodPost:

				key, ok := request.URL.Query()["key"]

				if !ok || len(key[0]) < 1 {
					log.Println("Url Param 'key' is missing")
					http.Error(writer, "Improper key", http.StatusBadRequest)
					return
				}
				apiKey := key[0]
				log.Println("User supplied key: " + apiKey)
				if apiKey == hub.GetUrlShortenerApiKey() {
					var burpRequest = chatapi.BurpRequestResponse{}
					dec := json.NewDecoder(request.Body)
					dec.DisallowUnknownFields()
					err := dec.Decode(&burpRequest)
					if err != nil {
						http.Error(writer, "Improper JSON", http.StatusBadRequest)
						return
					}
					newId := shortendURLs.AddNewShortenURL(burpRequest)
					log.Println("POST: " + newId)
					//encoder := base64.NewEncoder(base64.StdEncoding, writer)
					log.Println("POST: " + base64.StdEncoding.EncodeToString([]byte("https://"+serverNameInternal+":"+*port+"/shortener?id="+newId)))
					accessURL := "https://" + serverNameInternal + ":" + *port + "/shortener?id=" + newId
					base64Text := make([]byte, base64.StdEncoding.EncodedLen(len(accessURL)))
					base64.StdEncoding.Encode(base64Text, []byte(accessURL))
					writer.Write(base64Text)
				} else {
					http.Error(writer, "Wrong.", http.StatusBadRequest)
				}
			default:
				writer.Write([]byte("Unsupported."))
			}
		})
	}
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		chatapi.ServeWs(hub, w, r)
	})
	if _, err := os.Stat("./burpServer.pem"); err == nil {
		fmt.Println("file ", "burpServer.pem found switching to https")
		caCert, err := ioutil.ReadFile("./burpServer.pem")
		if err != nil {
			log.Fatal(err)
		}
		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(caCert)
		// Create the TLS Config with the CA pool and enable Client certificate validation
		tlsConfig := &tls.Config{
			ClientCAs:  caCertPool,
			ClientAuth: tls.VerifyClientCertIfGiven,
			MaxVersion: tls.VersionTLS12,
		}
		tlsConfig.BuildNameToCertificate()
		server := &http.Server{
			Addr:      ":" + *port,
			TLSConfig: tlsConfig,
		}
		log.Printf("Server running at wss://%s:%s", *host, *port)
		if httpErr = server.ListenAndServeTLS("burpServer.pem", "burpServer.key"); httpErr != nil {
			log.Fatal("The process exited with https error: ", httpErr.Error())
		}

	} else {
		log.Printf("Server running at ws://%s:%s", *host, *port)
		httpErr := http.ListenAndServe(":"+*port, nil)
		if httpErr != nil {
			log.Fatal("ListenAndServe: ", err)
		}
	}
}
