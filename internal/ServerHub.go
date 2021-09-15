package internal

import (
	"github.com/fasthttp/websocket"
	"log"
	"strconv"
	"strings"
	"sync"
)

type ServerHub struct {
	mu               sync.Mutex
	allClients       map[string]*Client
	rooms            map[string]*Room
	messages         chan Message
	serverPassword   string
	shortenerService *ShortenedUrls
}

func NewServerHub(serverPassword string) *ServerHub {
	serverHub := &ServerHub{
		allClients:     make(map[string]*Client),
		rooms:          make(map[string]*Room),
		messages:       make(chan Message, 1),
		serverPassword: serverPassword,
	}

	serverHub.rooms["server"] = NewRoom(serverHub, "", "server")
	go serverHub.writer()

	return serverHub
}

func (serverHub *ServerHub) GetClientByName(clientName string) *Client {
	return serverHub.allClients[clientName]
}

func (serverHub *ServerHub) SetShortenerService(shortenerService *ShortenedUrls) {
	serverHub.shortenerService = shortenerService
}

func (serverHub *ServerHub) writer() {
	for message := range serverHub.messages {
		room := serverHub.rooms[message.roomName]
		if message.target == "Self" { //echo message to client who sent it
			message.sender.sendChannel <- message
		} else if message.target == "Room" { //to everyone in the room
			if message.sender != nil {
				//sent from user to other room members
				for name, client := range room.clients {
					if name != message.sender.name {
						log.Printf("has client %v+ muted us? %t", client, room.clients[name].isGivenClientMuted(message.sender.name))
						if !room.clients[name].isGivenClientMuted(message.sender.name) {
							serverHub.sendMessage(client, message, room)
						}
					}
				}
			} else {
				//sent from server to room
				for _, client := range room.clients {
					serverHub.sendMessage(client, message, room)
				}
			}
		} else { //to a specific person
			if !room.clients[message.target].isGivenClientMuted(message.sender.name) {
				clientToMessage, exists := room.getClient(message.target)
				if exists && !clientToMessage.isGivenClientMuted(message.sender.name) {
					serverHub.sendMessage(clientToMessage, message, room)
				}
			}
		}
	}
}

func (serverHub *ServerHub) sendMessage(clientToMessage *Client, message Message, room *Room) {
	select {
	case clientToMessage.sendChannel <- message:
	default:
		close(clientToMessage.sendChannel)
		delete(room.clients, clientToMessage.name)
		if len(room.clients) == 0 {
			serverHub.deleteRoom(message.roomName)
		}
	}
}

func (serverHub *ServerHub) deleteRoom(roomName string) {
	if roomName != "server" {
		log.Printf("Room %s is empty. Deleting", roomName)
		delete(serverHub.rooms, roomName)
	}
	serverHub.updateRooms()
}

func (serverHub *ServerHub) updateRooms() {
	msg := NewBurpTCMessage()
	msg.MessageType = "GET_ROOMS_MESSAGE"

	keys := make([]string, 0, len(serverHub.rooms))
	for k := range serverHub.rooms {
		if k != "server" {
			keys = append(keys, k+"::"+strconv.FormatBool(len(serverHub.rooms[k].password) > 0))
		}
	}
	log.Printf("Current rooms: %s", strings.Join(keys, ","))
	msg.Data = strings.Join(keys, ",")
	serverHub.messages <- generateMessage(msg, nil, "server", "Room")
}

func (serverHub *ServerHub) ClientExistsInServer(clientName string) bool {
	for name := range serverHub.allClients {
		if name == clientName {
			return true
		}
	}
	return false
}

func (serverHub *ServerHub) Register(conn *websocket.Conn, clientName string) *Client {
	client := &Client{serverHub: serverHub, conn: conn,
		name: clientName, roomName: "server", mutedClients: []string{}, sendChannel: make(chan Message, 1024)}

	serverHub.mu.Lock()
	{
		serverHub.allClients[clientName] = client
		serverHub.rooms["server"].addClient(client)
	}
	serverHub.mu.Unlock()

	return client
}

func (serverHub *ServerHub) RemoveClient(client *Client) {
	serverHub.mu.Lock()
	delete(serverHub.allClients, client.name)
	serverHub.rooms[client.roomName].deleteClient(client.name)
	if client.roomName != "server" && len(serverHub.rooms[client.roomName].clients) == 0 {
		serverHub.deleteRoom(client.roomName)
	}
	close(client.sendChannel)
	serverHub.mu.Unlock()
}

func (serverHub *ServerHub) addRoom(roomPassword string, roomName string) {
	serverHub.mu.Lock()
	defer serverHub.mu.Unlock()
	serverHub.rooms[roomName] = NewRoom(serverHub, roomPassword, roomName)
	serverHub.updateRooms()
}
