package chatapi

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
)

type ChatAPI struct {
	rooms      map[string]*Room
	serverRoom *Room
	*sync.RWMutex
	ServerPassword string
}

//New start a new instance of the new chat api
func New(crypter AESCrypter) *ChatAPI {
	api := &ChatAPI{
		rooms:      make(map[string]*Room),
		serverRoom: CreateRoom("server", crypter),
		RWMutex:    new(sync.RWMutex),
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
	log.Println("Adding client")
	msg := NewBurpTCMessage()
	scanner := bufio.NewScanner(c)
	buf := make([]byte, 0, 8192*8192)
	scanner.Buffer(buf, 8192*8192)
	scanner.Scan()
	log.Printf("scan buffer: %s", scanner.Text())
	decryptedMessage := cAPI.serverRoom.crypter.Decrypt(scanner.Text())
	if err := json.Unmarshal([]byte(decryptedMessage), &msg); err != nil {
		log.Printf("Could not decode encrypted message, error: %s \n", err)
		c.Close()
	} else {
		log.Printf("decrypted message: %+v\n", msg)
		responseMessage := NewBurpTCMessage()
		if msg.MessageType == "LOGIN_MESSAGE" {
			writer := bufio.NewWriter(c)
			if msg.AuthenticationString == cAPI.serverRoom.crypter.aesKey {
				log.Println("login successful")
				responseMessage.AuthenticationString = "SUCCESS"
				log.Println(responseMessage)
				writer.WriteString(SendMessage(*responseMessage, cAPI.serverRoom.crypter))
				writer.Flush()
				cAPI.addClientToServer(msg, c)
			} else {
				log.Println("login failed")
				responseMessage.AuthenticationString = "FAILED"
				writer.WriteString(SendMessage(*responseMessage, cAPI.serverRoom.crypter))
				writer.Flush()
				c.Close()
			}
		}
	}
}

func (cAPI *ChatAPI) updateRooms() {
	msg := NewBurpTCMessage()
	msg.MessageType = "GET_ROOMS_MESSAGE"

	keys := make([]string, 0, len(cAPI.rooms))
	for k := range cAPI.rooms {
		keys = append(keys, k)
	}
	log.Printf("Current rooms: %s", strings.Join(keys, ","))
	msg.Data = strings.Join(keys, ",")
	for _, wc := range cAPI.serverRoom.clients {
		go func(wc chan<- string) {
			wc <- SendMessage(*msg, cAPI.serverRoom.crypter)
		}(wc.wc)
	}
}

func (cAPI *ChatAPI) moveClientToRoom(client *client, currentRoom string, newRoom string) {
	cAPI.Lock()
	defer cAPI.Unlock()
	r, ok := cAPI.rooms[newRoom]
	if !ok {
		r = CreateRoom(newRoom, cAPI.serverRoom.crypter)
		cAPI.rooms[newRoom] = r
	}
	if currentRoom != "server" {
		cAPI.rooms[currentRoom].RemoveClientSync(client.Name)
		client.changeChannel(cAPI.serverRoom.Msgch)
	} else {
		cAPI.serverRoom.RemoveClientSync(client.Name)
		client.changeChannel(r.Msgch)
	}
	r.AddExistingClient(client)
	cAPI.updateRooms()
}

func (cAPI *ChatAPI) addClientToServer(clientMsg *BurpTCMessage, c io.ReadWriteCloser) {
	cAPI.serverRoom.AddClient(c, clientMsg.SendingUser, cAPI)
}

func SendMessage(messageToSend BurpTCMessage, crypter AESCrypter) string {
	jsonMsg, _ := json.Marshal(messageToSend)
	return crypter.Encrypt(string(jsonMsg)) + "\n"
}

func (cAPI *ChatAPI) handleClient(clientMsg *BurpTCMessage, c io.ReadWriteCloser) {
	cAPI.Lock()
	defer cAPI.Unlock()
	r, ok := cAPI.rooms[clientMsg.RoomName]
	if !ok {
		r = CreateRoom(clientMsg.RoomName, *NewAESCrypter())
	}
	r.AddClient(c, clientMsg.SendingUser, cAPI)
	cAPI.rooms[clientMsg.RoomName] = r
}

func (cAPI *ChatAPI) removeClientFromRooms(c *client) {
	for room := range cAPI.rooms {
		cAPI.rooms[room].RemoveClientSync(c.Name)
	}
	cAPI.updateRooms()
}

func (cAPI *ChatAPI) removeClientFromRoom(c *client, roomName string) {
	cAPI.Lock()
	defer cAPI.Unlock()
	cAPI.rooms[roomName].RemoveClientSync(c.Name)
	log.Printf("clients: %d", len(cAPI.rooms[roomName].clients))
	if len(cAPI.rooms[roomName].clients) == 0 {
		close(cAPI.rooms[roomName].Quit)
		delete(cAPI.rooms, roomName)
	}
	cAPI.serverRoom.AddExistingClient(c)
	cAPI.updateRooms()
}
