package chatapi

import (
	"encoding/base64"
	"encoding/json"
	"log"
	"strconv"
	"strings"
)

type message struct {
	msg      *BurpTCMessage
	sender   *Client
	roomName string
	target   string
}

// Hub maintains the set of active clients and broadcasts messages to the
// clients.
type Hub struct {
	serverPassword string

	allClientNames []string

	// Registered clients.
	rooms map[string]*Room

	// Inbound messages from the clients.
	broadcast chan message

	// Register requests from the clients.
	register chan *Client

	// Unregister requests from clients.
	unregister chan *Client
}

func NewHub(password string) *Hub {
	hub := &Hub{
		allClientNames: []string{},
		serverPassword: password,
		broadcast:      make(chan message),
		register:       make(chan *Client),
		unregister:     make(chan *Client),
		rooms:          make(map[string]*Room),
	}
	hub.rooms["server"] = NewRoom("")
	return hub
}

func (h *Hub) addRoom(roomName string, room *Room) {
	h.rooms[roomName] = room
	h.updateRooms()
}

func (h *Hub) addClientToServerList(clientName string) {
	h.allClientNames = append(h.allClientNames, clientName)
}

func (h *Hub) removeClientFromServerList(clientName string) {
	if h.clientExistsInServer(clientName) {
		h.allClientNames = remove(h.allClientNames, index(h.allClientNames, clientName))
	}
}

func (h *Hub) clientExistsInServer(clientName string) bool {
	return index(h.allClientNames, clientName) != -1
}

func (h *Hub) deleteRoom(roomName string) {
	room := h.rooms[roomName]
	if room != nil && roomName != "server" {
		log.Printf("Room %s is empty. Deleting", roomName)
		delete(h.rooms, roomName)
	}
	h.updateRooms()
}

func (h *Hub) generateMessage(burpTCMessage *BurpTCMessage, sender *Client, roomName string, target string) message {
	return message{
		msg:      burpTCMessage,
		sender:   sender,
		roomName: roomName,
		target:   target,
	}
}

func (h *Hub) updateRooms() {
	msg := NewBurpTCMessage()
	msg.MessageType = "GET_ROOMS_MESSAGE"

	keys := make([]string, 0, len(h.rooms))
	for k := range h.rooms {
		if k != "server" {
			keys = append(keys, k+"::"+strconv.FormatBool(len(h.rooms[k].password) > 0))
		}
	}
	log.Printf("Current rooms: %s", strings.Join(keys, ","))
	msg.Data = strings.Join(keys, ",")
	h.broadcast <- h.generateMessage(msg, nil, "server", "Room")
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			log.Println("New Client")
			h.addClientToServerList(client.name)
			h.rooms[client.roomName].addClient(client)
		case client := <-h.unregister:
			room := h.rooms[client.roomName]
			if room != nil {
				if _, ok := room.getClient(client.name); ok {
					room.deleteClient(client.name)
					if client.roomName != "server" && len(room.clients) == 0 {
						h.deleteRoom(client.roomName)
					}
					close(client.send)
					h.removeClientFromServerList(client.name)
					log.Println("Client Leaving")
				}
			}
		case message := <-h.broadcast:
			jsonMsg, _ := json.Marshal(message.msg)
			log.Println("Sending: " + string(jsonMsg))
			encodedBuf := make([]byte, base64.StdEncoding.EncodedLen(len(jsonMsg)))
			base64.StdEncoding.Encode(encodedBuf, jsonMsg)
			room := h.rooms[message.roomName]
			if message.target == "Self" { //to yourself
				message.sender.send <- encodedBuf
			} else if message.target == "Room" { //to everyone in the room
				if message.sender != nil {
					for client := range room.clients {
						if client != message.sender.name {
							log.Printf("has client %s muted us? %t", client, room.clients[client].isGivenClientMuted(message.sender.name))
							if !room.clients[client].isGivenClientMuted(message.sender.name) {
								select {
								case room.clients[client].send <- encodedBuf:
								default:
									close(room.clients[client].send)
									delete(room.clients, client)
									if len(room.clients) == 0 {
										h.deleteRoom(message.roomName)
									}
								}
							}
						}
					}
				} else {
					for client := range room.clients {
						select {
						case room.clients[client].send <- encodedBuf:
						default:
							close(room.clients[client].send)
							delete(room.clients, client)
							if len(room.clients) == 0 {
								h.deleteRoom(message.roomName)
							}
						}
					}
				}
			} else { //to a specific person
				if !room.clients[message.target].isGivenClientMuted(message.sender.name) {
					room.clients[message.target].send <- encodedBuf
				}
			}
		}
	}
}
