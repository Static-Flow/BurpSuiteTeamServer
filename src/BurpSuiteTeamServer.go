package main

import (
	"flag"
	"github.com/Static-Flow/BurpSuiteTeamServer/src/chatapi"
	"log"
	"net"
)

func RunTCPWithExistingAPI(connection string, chat *chatapi.ChatAPI) error {
	l, err := net.Listen("tcp", connection)
	if err != nil {
		log.Println("Error connecting to chat client", err)
		return err
	}
	defer l.Close()
	for {
		conn, err := l.Accept()
		if err != nil {
			break
		}
		go func(c net.Conn) {
			chat.AddClient(c)
		}(conn)
	}

	return err
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	tcpAddr := flag.String("tcp", "0.0.0.0:8989", "Address for the TCP chat server to listen on")
	flag.Parse()
	api := chatapi.New()
	if err := RunTCPWithExistingAPI(*tcpAddr, api); err != nil {
		log.Fatalf("Could not listen on %s, error %s \n", *tcpAddr, err)
	}
}
