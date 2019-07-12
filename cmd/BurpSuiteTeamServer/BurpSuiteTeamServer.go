package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/Static-Flow/BurpSuiteTeamServer/authentication"
	"github.com/Static-Flow/BurpSuiteTeamServer/chatapi"
	"github.com/gorilla/mux"
	"log"
	"net"
	"net/http"
	"os"
)

func RunTCPWithExistingAPI(connection string, chat *chatapi.ChatAPI) error {
	l, err := net.Listen("tcp", connection)
	if err != nil {
		log.Println("Error connecting to chat client", err)
		return err
	}
	log.Println("Awaiting Clients...")
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

func handleRooms(api *chatapi.ChatAPI) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		roomMap := api.GetRooms()
		keys := make([]string, 0, len(roomMap))
		for k := range roomMap {
			keys = append(keys, k)
		}
		w.Header().Set("Content-Type", "application/json")
		jsonResponse, _ := json.Marshal(keys)
		fmt.Fprintln(w, string(jsonResponse))
	}
}

func handleGetScope(api *chatapi.ChatAPI) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		roomMap := api.GetRooms()
		keys := make([]string, 0, len(roomMap))
		for k := range roomMap {
			keys = append(keys, k)
		}
		fmt.Fprintln(w, keys)
	}
}

func handleRoomMembers(api *chatapi.ChatAPI) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		roomMap := api.GetRoomMembers(vars["name"])
		keys := make([]string, 0, len(roomMap))
		for k := range roomMap {
			keys = append(keys, k)
		}
		fmt.Fprintln(w, keys)
	}
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	port := os.Getenv("PORT")
	if port == "" {
		port = "8989"
	}
	tcpAddr := flag.String("tcp", "0.0.0.0:"+port, "Address for the TCP chat server to listen on")
	serverPassword := flag.String("serverPassword", "", "Server Password, default is none")
	serverUsername := flag.String("serverUsername", "", "Server Username, default is none")
	flag.Parse()
	api := chatapi.New(*serverPassword)
	authenticationWrapper := authentication.New(*serverUsername, *serverPassword)
	go func() {
		r := mux.NewRouter()
		r.Handle("/rooms/{name}", authenticationWrapper.WrapHandler(handleRoomMembers(api)))
		r.Handle("/rooms", authenticationWrapper.WrapHandler(handleRooms(api)))
		r.PathPrefix("/").Handler(http.FileServer(http.Dir("../../static")))
		http.ListenAndServe(":8888", r)
	}()
	if err := RunTCPWithExistingAPI(*tcpAddr, api); err != nil {
		log.Fatalf("Could not listen on %s, error %s \n", *tcpAddr, err)
	}
}
