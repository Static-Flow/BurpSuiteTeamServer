package chatapi

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

type ChatAPI struct {
	rooms map[string]*Room
	*sync.RWMutex
	ServerPassword string
}

type clientInfo struct {
	Room     string `json:"room"`
	Name     string `json:"name"`
	Password string `json:"password"`
}

//New start a new instance of the new chat api
func New(serverPassword string) *ChatAPI {
	api := &ChatAPI{
		rooms:          make(map[string]*Room),
		RWMutex:        new(sync.RWMutex),
		ServerPassword: serverPassword,
	}
	//handle shutdown
	go func() {
		// Handle SIGINT and SIGTERM.
		ch := make(chan os.Signal)
		signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
		<-ch
		fmt.Println("Closing tcp connection")
		api.RLock()
		defer api.RUnlock()
		for name, r := range api.rooms {
			log.Printf("Closing room %s \n", name)
			close(r.Quit)
		}
		os.Exit(0)
	}()
	return api
}

func (cAPI *ChatAPI) GetRoomMembers(roomName string) map[string]*client {
	if val, ok := cAPI.rooms[roomName]; ok {
		return val.clients
	} else {
		return make(map[string]*client)
	}
}

func (cAPI *ChatAPI) GetRooms() map[string]*Room {
	return cAPI.rooms
}

//AddClient adds a new client to the chat server. Expects a JSON file
func (cAPI *ChatAPI) AddClient(c io.ReadWriteCloser) {
	msg := NewBurpTCMessage()
	responseMessage := NewBurpTCMessage()
	if err := json.NewDecoder(c).Decode(&msg); err != nil {
		log.Printf("Could not decode message, error: %s \n", err)
	} else if msg.MessageType == "LOGIN_MESSAGE" {
		writer := bufio.NewWriter(c)
		log.Println(msg)
		if msg.AuthenticationString == cAPI.ServerPassword {
			log.Println("login successful")
			responseMessage.AuthenticationString = "SUCCESS"
			log.Println(responseMessage)
			writer.WriteString(SendMessage(*responseMessage))
			writer.Flush()
			cAPI.handleClient(msg, c)
		} else {
			log.Println("login failed")
			responseMessage.AuthenticationString = "FAILED"
			writer.WriteString(SendMessage(*responseMessage))
			writer.Flush()
			c.Close()
		}
	}
}

func SendMessage(messageToSend BurpTCMessage) string {
	jsonMsg, _ := json.Marshal(messageToSend)
	return string(jsonMsg) + "\n"
}

func (cAPI *ChatAPI) handleClient(clientMsg *BurpTCMessage, c io.ReadWriteCloser) {
	cAPI.Lock()
	defer cAPI.Unlock()
	r, ok := cAPI.rooms[clientMsg.RoomName]
	if !ok {
		r = CreateRoom(clientMsg.RoomName)
	}
	r.AddClient(c, clientMsg.SendingUser, cAPI.ServerPassword)
	cAPI.rooms[clientMsg.RoomName] = r
}
