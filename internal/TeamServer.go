package internal

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"github.com/fasthttp/websocket"
	"github.com/valyala/fasthttp"
	"io/ioutil"
	"log"
	"net"
	"os"
)

var upgrader = websocket.FastHTTPUpgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func StartServer(serverPassword *string, host *string, enableUrlShortener *bool, shortenerPort *string, port *string) *Hub {
	hub = NewHub(*serverPassword)

	GenCrt(*host)
	if *enableUrlShortener {
		shortendURLs := NewShortenedUrls(*shortenerPort, *host)
		hub.SetShortenerService(shortendURLs)
	}

	if _, err := os.Stat("./burpServer.pem"); err == nil {
		fmt.Println("file burpServer.pem found switching to https")
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

		ln, err := net.Listen("tcp", *host+":"+*port)
		if err != nil {
			log.Fatal(err)
		}
		log.Fatal(fasthttp.Serve(tls.NewListener(ln, tlsConfig), func(ctx *fasthttp.RequestCtx) {
			switch string(ctx.Path()) {
			case "/":
				if authHeader := ctx.Request.Header.Peek("Auth"); bytes.Equal(authHeader, []byte(*serverPassword)) {
					username := string(ctx.Request.Header.Peek("Username"))

					if err := upgrader.Upgrade(ctx, func(conn *websocket.Conn) {
						log.Println("Opening connection")
						client := hub.Register(conn, username)
						log.Printf("client connection: %v", client)
						go client.Writer()
						client.Reader()

					}); err != nil {
						log.Println("Socket upgrade error:", err)
					}

				} else {
					ctx.Response.SetStatusCode(fasthttp.StatusUnauthorized)
					ctx.SetBody([]byte("401 - Bad Auth!"))
				}
			default:
				ctx.Error("Unsupported path", fasthttp.StatusNotFound)
			}
		}))

	} else {
		log.Printf("Server running at ws://%s:%s", *host, *port)
		httpErr := fasthttp.ListenAndServe(":"+*port, nil)
		if httpErr != nil {
			log.Fatal("ListenAndServe: ", err)
		}
	}
	return hub
}
