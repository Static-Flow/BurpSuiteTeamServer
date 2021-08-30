package chatapi

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
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
	serverPassword []byte

	allClientNames [][]byte

	// Registered clients.
	rooms map[string]*Room

	// Inbound messages from the clients.
	broadcast chan message

	// Register requests from the clients.
	register chan *Client

	shortenerApiKey string

	shortenedUrls *ShortenedUrls

	hostname string

	serverPort string

	authPackageDownloaded bool
}

func NewHub(password string, port string, hostname string) *Hub {
	hub := &Hub{
		serverPassword:        []byte(password),
		broadcast:             make(chan message),
		register:              make(chan *Client),
		rooms:                 make(map[string]*Room),
		shortenerApiKey:       "",
		shortenedUrls:         NewShortenedUrls(),
		serverPort:            port,
		hostname:              hostname,
		authPackageDownloaded: false,
	}

	hub.SetUrlShortenerApiKey(hub.shortenedUrls.GenString() + hub.shortenedUrls.GenString())
	hub.rooms["server"] = NewRoom("")
	return hub
}

func (h *Hub) GetServerPort() string {
	return h.serverPort
}

func (h *Hub) GetServerHostname() string {
	return h.hostname
}

func (h *Hub) GetShortenerUrl(id string) string {
	return fmt.Sprintf("https://%s:%s/shortener?id=%s", h.hostname, h.serverPort, id)
}

func (h *Hub) GetUrlShortener() *ShortenedUrls {
	return h.shortenedUrls
}

func (h *Hub) SetUrlShortenerApiKey(key string) {
	h.shortenerApiKey = key
}

func (h *Hub) GetUrlShortenerApiKey() string {
	return h.shortenerApiKey
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

func (h *Hub) updateRoomMembers(roomName string) {
	h.rooms[roomName].Lock()
	defer h.rooms[roomName].Unlock()
	msg := NewBurpTCMessage()
	msg.MessageType = "NEW_MEMBER_MESSAGE"

	clientNames := make([]string, 0, len(h.rooms[roomName].clients))
	for k := range h.rooms[roomName].clients {
		fmt.Printf("%+v\n", k)
		clientNames = append(clientNames, k.name)
	}
	if len(clientNames) > 0 {
		msg.Data = strings.Join(clientNames, ",")
		h.broadcast <- h.generateMessage(msg, nil, roomName, "Room")
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
			h.rooms[client.roomName].addClient(client)
		case message := <-h.broadcast:
			jsonMsg, _ := json.Marshal(message.msg)
			log.Println("Sending: " + string(jsonMsg))
			encodedBuf := make([]byte, base64.StdEncoding.EncodedLen(len(jsonMsg)))
			base64.StdEncoding.Encode(encodedBuf, jsonMsg)
			room := h.rooms[message.roomName]
			if message.target == "Self" { //to yourself
				message.sender.send <- encodedBuf
			} else if message.target == "Room" { //to everyone in the room
				fmt.Println("Sending message to room: " + message.roomName)
				if message.sender != nil { //this message is from a client to other clients in a room
					for client := range room.clients {
						fmt.Println("Attempting to send to client: " + client.name)
						if client.name != message.sender.name {
							isMuted := client.isGivenClientMuted(message.sender.name)
							log.Printf("has client %s muted us? %t", client.name, isMuted)
							if !isMuted {
								select {
								case client.send <- encodedBuf:
								default:
									close(client.send)
									delete(room.clients, client)
									if len(room.clients) == 0 {
										h.deleteRoom(message.roomName)
									}
								}
							}
						}
					}
				} else { //this is a message from the server to clients in a room
					for client := range room.clients {
						select {
						case client.send <- encodedBuf:
						default:
							close(client.send)
							delete(room.clients, client)
							if len(room.clients) == 0 {
								h.deleteRoom(message.roomName)
							}
						}
					}
				}
			} else { //to a specific person
				clientToMessage, exists := room.getClient(message.target)
				if exists && !clientToMessage.isGivenClientMuted(message.sender.name) {
					clientToMessage.send <- encodedBuf
				}
			}
		}
	}
}
