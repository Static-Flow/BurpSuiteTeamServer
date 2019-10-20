package main

import (
	"flag"
	"github.com/Static-Flow/BurpSuiteTeamServer/chatapi"
	"log"
	"net/http"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	var port = flag.String("port", ":9999", "http service address")
	var serverPassword = flag.String("serverPassword", "superleetsecret", "password for the server")
	flag.Parse()
	hub := chatapi.NewHub(*serverPassword)
	go hub.Run()
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		chatapi.ServeWs(hub, w, r)
	})
	err := http.ListenAndServe(*port, nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
