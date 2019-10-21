package main

import (
	"crypto/tls"
	"crypto/x509"
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
	var serverPassword = flag.String("serverPassword", "superleetsecret", "password for the server")
	flag.Parse()
	chatapi.GenCrt(*host)
	hub := chatapi.NewHub(*serverPassword)
	go hub.Run()
	var httpErr error
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		chatapi.ServeWs(hub, w, r)
	})
	if _, err := os.Stat("./burpServer.pem"); err == nil {
		fmt.Println("file ", "burpServer.pem found switching to https")
		caCert, err := ioutil.ReadFile("./burpServer.pem")
		if err != nil {
			log.Fatal(err)
		}
		caKey, err := ioutil.ReadFile("./burpServer.key")
		if err != nil {
			log.Fatal(err)
		}
		servTLSCert, err := tls.X509KeyPair(caCert, caKey)
		if err != nil {
			log.Fatalf("invalid key pair: %v", err)
		}
		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(caCert)
		// Create the TLS Config with the CA pool and enable Client certificate validation
		tlsConfig := &tls.Config{
			Certificates: []tls.Certificate{servTLSCert},
			ClientCAs:    caCertPool,
			ClientAuth:   tls.RequireAndVerifyClientCert,
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
