package chatapi

import (
	"encoding/base64"
	"encoding/json"
	"log"
	"strings"
)

type message struct {
	msg      *BurpTCMessage
	sender   *Client
	roomName string
	target   string
}

type subscription struct {
	conn     *Client
	roomName string
}

// Hub maintains the set of active clients and broadcasts messages to the
// clients.
type Hub struct {
	serverPassword string

	// Registered clients.
	rooms map[string]*Room

	// Inbound messages from the clients.
	broadcast chan message

	// Register requests from the clients.
	register chan *subscription

	// Unregister requests from clients.
	unregister chan *subscription
}

func NewHub(password string) *Hub {
	hub := &Hub{
		serverPassword: password,
		broadcast:      make(chan message),
		register:       make(chan *subscription),
		unregister:     make(chan *subscription),
		rooms:          make(map[string]*Room),
	}
	hub.rooms["server"] = NewRoom()
	return hub
}

func (h *Hub) addRoom(roomName string, room *Room) {
	h.rooms[roomName] = room
	h.updateRooms()
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
		keys = append(keys, k)
	}
	log.Printf("Current rooms: %s", strings.Join(keys, ","))
	msg.Data = strings.Join(keys, ",")
	h.broadcast <- h.generateMessage(msg, nil, "server", "Room")
}

func (h *Hub) Run() {
	for {
		select {
		case subscription := <-h.register:
			log.Println("New Client")
			h.rooms[subscription.roomName].addClient(subscription.conn)
		case subscription := <-h.unregister:
			room := h.rooms[subscription.roomName]
			if room != nil {
				if _, ok := room.getClient(subscription.conn.name); ok {
					room.deleteClient(subscription.conn.name)
					if subscription.roomName != "server" && len(room.clients) == 0 {
						h.deleteRoom(subscription.roomName)
					}
					close(subscription.conn.send)
					log.Println("Client Leaving")
				}
			}
		case message := <-h.broadcast:
			jsonMsg, _ := json.Marshal(message.msg)
			log.Println(string(jsonMsg))
			encodedBuf := make([]byte, base64.StdEncoding.EncodedLen(len(jsonMsg)))
			base64.StdEncoding.Encode(encodedBuf, jsonMsg)
			room := h.rooms[message.roomName]
			if message.target == "Self" { //to yourself
				message.sender.send <- encodedBuf
			} else if message.target == "Room" { //to everyone in the room
				if message.sender != nil {
					for client := range room.clients {
						if client != message.sender.name {
							log.Printf("is client %s muted %t", client, room.clients[client].isGivenClientMuted(message.sender.name))
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
