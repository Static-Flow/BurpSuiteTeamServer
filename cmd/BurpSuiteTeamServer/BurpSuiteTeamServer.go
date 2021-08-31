package main

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
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
	var shortenerPort = flag.String("shortPort", "4444", "Sets the built-in URL shortener port")
	flag.Parse()

	serverHub = internal.NewServerHub(*serverPassword)

	internal.GenCrt(*host)
	if *enableUrlShortener {
		shortendURLs := internal.NewShortenedUrls(*shortenerPort, *host)
		serverHub.SetShortenerService(shortendURLs)
	}

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
