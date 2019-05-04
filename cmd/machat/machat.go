package main

import (
	"flag"
	"github.com/Static-Flow/BurpSuiteTeamServer"
	"github.com/Static-Flow/BurpSuiteTeamServer/chatapi"
	"log"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	tcpAddr := flag.String("tcp", "0.0.0.0:8989", "Address for the TCP chat server to listen on")
	flag.Parse()
	api := chatapi.New()
	if err := machat.RunTCPWithExistingAPI(*tcpAddr, api); err != nil {
		log.Fatalf("Could not listen on %s, error %s \n", *tcpAddr, err)
	}
}
