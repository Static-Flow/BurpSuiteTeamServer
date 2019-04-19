package main

import (
	"flag"
	"log"

	"github.com/minaandrawos/machat"
	"github.com/minaandrawos/machat/chatapi"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	tcpAddr := flag.String("tcp", "localhost:8989", "Address for the TCP chat server to listen on")
	flag.Parse()
	api := chatapi.New()
	if err := machat.RunTCPWithExistingAPI(*tcpAddr, api); err != nil {
		log.Fatalf("Could not listen on %s, error %s \n", *tcpAddr, err)
	}
}
