package main

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/Static-Flow/BurpSuiteTeamServer/internal"
	"github.com/fasthttp/websocket"
	"github.com/valyala/fasthttp"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
)

var serverHub *internal.ServerHub
var upgrader = websocket.FastHTTPUpgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	var host = flag.String("host", "localhost", "host for TLS cert. Defaults to localhost")
	var port = flag.String("port", "9999", "http service address")
	var serverPassword = flag.String("serverPassword", "", "password for the server")
	var enableUrlShortener = flag.Bool("enableShortener", false, "Enables the built-in URL shortener")
	flag.Parse()

	serverNameInternal, err := os.Hostname()
	if err != nil {
		fmt.Printf("No hostname, panic: %v\n", err)
		panic(err)
	}

	serverHub = internal.NewServerHub(*serverPassword)

	internal.GenCrt(*host)
	if *enableUrlShortener {
		shortendURLs := internal.NewShortenedUrls()
		serverHub.SetShortenerService(shortendURLs)
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
					_, _ = writer.Write(burpRequestJson)
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
				if apiKey == shortendURLs.GetUrlShortenerApiKey() {
					var burpRequest = internal.BurpRequestResponse{}
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
					_, _ = writer.Write(base64Text)
				} else {
					http.Error(writer, "Wrong.", http.StatusBadRequest)
				}
			default:
				_, _ = writer.Write([]byte("Unsupported."))
			}
		})
	}
	//mux := &http.ServeMux{}
	//mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
	//	if authHeader := r.Header.Get("Auth"); authHeader == *serverPassword {
	//		if serverHub.ClientExistsInServer(r.Header.Get("Username")) {
	//			log.Println("Found duplicate name")
	//			w.WriteHeader(http.StatusConflict)
	//			_, _ = w.Write([]byte("409 - Duplicate name in server!"))
	//		} else {
	//			fmt.Println("Making websocket connection")
	//			upgrader := newUpgrader(r.Header.Get("Username"))
	//			_, _ = upgrader.Upgrade(w, r, nil)
	//			//conn,_,_, err := ws.UpgradeHTTP(r, w)
	//			//if err != nil {
	//			//	log.Println(err)
	//			//	return
	//			//}
	//			//
	//			//go func() {
	//			//	deadlinedConnection := internal.DeadlinedConnection{Conn: conn, T: *ioTimeout}
	//			//	newClient := serverHub.Register(deadlinedConnection, r.Header.Get("Username"))
	//			//	defer func() {
	//			//		_ = deadlinedConnection.Close()
	//			//		serverHub.RemoveClient(newClient)
	//			//	}()
	//			//	for {
	//			//		msg, _, err := wsutil.ReadClientData(conn)
	//			//		if err != nil {
	//			//			log.Printf("error reading Message: %v", err)
	//			//			break
	//			//		}
	//			//
	//			//		if newClient.HandleMessage(msg) != nil {
	//			//			log.Printf("error parsing Message: %v", err)
	//			//			break
	//			//		}
	//			//	}
	//			//}()
	//		}
	//
	//	} else {
	//		w.WriteHeader(http.StatusUnauthorized)
	//		_, _ = w.Write([]byte("401 - Bad Auth!"))
	//	}
	//})
	if _, err := os.Stat("./burpServer.pem"); err == nil {
		fmt.Println("file ", "burpServer.pem found switching to https")
		caCert, err := ioutil.ReadFile("./burpServer.pem")
		if err != nil {
			log.Fatal(err)
		}
		crt, err := tls.LoadX509KeyPair("./burpServer.pem", "./burpServer.key")
		if err != nil {
			log.Fatal(err)
		}
		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(caCert)
		// Create the TLS Config with the CA pool and enable Client certificate validation
		tlsConfig := &tls.Config{
			ClientCAs:    caCertPool,
			ClientAuth:   tls.VerifyClientCertIfGiven,
			MaxVersion:   tls.VersionTLS12,
			Certificates: []tls.Certificate{crt},
		}

		ln, err := net.Listen("tcp", ":"+*port)
		if err != nil {
			log.Fatal(err)
		}
		log.Fatal(fasthttp.Serve(tls.NewListener(ln, tlsConfig), func(ctx *fasthttp.RequestCtx) {
			switch string(ctx.Path()) {
			case "/":
				if authHeader := ctx.Request.Header.Peek("Auth"); bytes.Equal(authHeader, []byte(*serverPassword)) {
					username := string(ctx.Request.Header.Peek("Username"))
					if serverHub.ClientExistsInServer(username) {
						log.Println("Found duplicate name")
						ctx.Response.SetStatusCode(http.StatusConflict)
						ctx.SetBody([]byte("409 - Duplicate name in server!"))
					} else {
						if err := upgrader.Upgrade(ctx, func(conn *websocket.Conn) {
							log.Println("Opening connection")
							client := serverHub.Register(conn, username)
							log.Printf("client connection: %v+ /n", client)
							go client.Writer()
							client.Reader()

						}); err != nil {
							log.Println(err)
						}

					}
				} else {
					ctx.Response.SetStatusCode(http.StatusUnauthorized)
					ctx.SetBody([]byte("401 - Bad Auth!"))
				}
			default:
				ctx.Error("Unsupported path", fasthttp.StatusNotFound)
			}
		}))

	} else {
		log.Printf("Server running at ws://%s:%s", *host, *port)
		httpErr := http.ListenAndServe(":"+*port, nil)
		if httpErr != nil {
			log.Fatal("ListenAndServe: ", err)
		}
	}
}
