package main

import (
	"crypto/tls"
	"crypto/x509"
	"flag"
	"fmt"
	"github.com/Static-Flow/BurpSuiteTeamServer/chatapi"
	"github.com/chyeh/pubip"
	"github.com/gorilla/mux"
	"github.com/valyala/fasthttp"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/http/pprof"
	"os"
)

func main() {
	go func() {
		r := mux.NewRouter()
		r.Path("/").HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			_, _ = fmt.Fprintf(writer, "Hello")
		})
		r.PathPrefix("/debug/").Handler(http.DefaultServeMux)

		r.HandleFunc("/debug/pprof/", pprof.Index)
		r.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
		r.HandleFunc("/debug/pprof/profile", pprof.Profile)
		r.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
		r.HandleFunc("/debug/pprof/trace", pprof.Trace)
		log.Println(http.ListenAndServe(":6060", r))
	}()
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	var host = flag.String("host", "localhost", "host for TLS cert. Defaults to localhost")
	var port = flag.String("port", "9999", "http service address")
	var shortenerServicePort = flag.String("shortenerPort", "8888", "shortener service port")
	var serverPassword = flag.String("serverPassword", "", "password for the server")
	var enableUrlShortener = flag.Bool("enableShortener", false, "Enables the built-in URL shortener")
	var localUrlShortener = flag.Bool("localShortener", false, "debug flag for local URL shortener")

	flag.Parse()
	chatapi.GenCrt(*host)
	ip, err := pubip.Get()
	if err != nil {
		fmt.Println("Couldn't Get public IP", err)
		return
	}
	var hub *chatapi.Hub
	if *localUrlShortener {
		hub = chatapi.NewHub(*serverPassword, *shortenerServicePort, "localhost")
	} else {
		hub = chatapi.NewHub(*serverPassword, *shortenerServicePort, ip.String())
	}
	go hub.Run()

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
		ln, err := net.Listen("tcp4", ":"+*port)
		if err != nil {
			panic(err)
		}

		go func() {
			var requestHandler fasthttp.RequestHandler
			requestHandler = func(ctx *fasthttp.RequestCtx) {
				switch string(ctx.Path()) {
				case "/auth":
					chatapi.AuthPackage(ctx, hub)
				default:
					ctx.Error("Unsupported path", fasthttp.StatusNotFound)
				}
			}
			if *enableUrlShortener {
				requestHandler = func(ctx *fasthttp.RequestCtx) {
					switch string(ctx.Path()) {
					case "/auth":
						chatapi.AuthPackage(ctx, hub)
					case "/shortener":
						chatapi.HandleShortUrl(ctx, hub)
					default:
						ctx.Error("Unsupported path", fasthttp.StatusNotFound)
					}
				}
			}
			fasthttp.ListenAndServeTLS(":"+*shortenerServicePort, "./burpServer.pem", "./burpServer.key", requestHandler)

		}()
		requestHandler := func(ctx *fasthttp.RequestCtx) {
			switch string(ctx.Path()) {
			case "/":
				chatapi.ServeWs(ctx, hub)
			default:
				ctx.Error("Unsupported path", fasthttp.StatusNotFound)
			}
		}
		server := fasthttp.Server{
			Handler: requestHandler,
		}
		lnTls := tls.NewListener(ln, tlsConfig)
		log.Printf("Server running at wss://%s:%s", *host, *port)
		if httpErr := server.Serve(lnTls); httpErr != nil {
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
