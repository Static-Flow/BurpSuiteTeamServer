package main

import (
	"flag"
	"github.com/Static-Flow/BurpSuiteTeamServer/internal"
	"log"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	var host = flag.String("host", "localhost", "host for TLS cert. Defaults to localhost")
	var port = flag.String("port", "9999", "http service address")
	var serverPassword = flag.String("serverPassword", "", "password for the server")
	var enableUrlShortener = flag.Bool("enableShortener", false, "Enables the built-in URL shortener")
	var shortenerPort = flag.String("shortPort", "4444", "Sets the built-in URL shortener port")
	flag.Parse()

	internal.StartServer(serverPassword, host, enableUrlShortener, shortenerPort, port)
}
