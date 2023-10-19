package internal

import (
	"errors"
	"fmt"
	"github.com/fasthttp/websocket"
	"log"
	"strconv"
	"strings"
)

var hub *Hub

type Hub struct {
	rooms            map[string]*Room
	messages         chan *Message
	register         chan *Client
	unregister       chan *Client
	serverPassword   string
	shortenerService *ShortenedUrls
}

func NewHub(serverPassword string) *Hub {
	hub := &Hub{
		register:       make(chan *Client),
		unregister:     make(chan *Client),
		rooms:          make(map[string]*Room),
		messages:       make(chan *Message, 1024),
		serverPassword: serverPassword,
	}

	//initialize server lobby room
	hub.rooms["server"] = NewRoom("server", "")

	go hub.eventLoop()

	return hub
}

func (h *Hub) eventLoop() {
	for {
		select {
		case newSubscription := <-h.register:
			log.Printf("Registering new client %v", newSubscription)
			//when registering we add them to the server lobby default room
			h.rooms["server"].clients[newSubscription.name] = newSubscription
		case leavingSubscription := <-h.unregister:
			log.Printf("Client %v is leaving", leavingSubscription)
			//get current room members
			currentRoomMembers := h.rooms[leavingSubscription.room]
			//if the room exists
			if currentRoomMembers != nil {
				//if the current room members includes the leaving client
				if _, ok := currentRoomMembers.clients[leavingSubscription.name]; ok {
					//remove the client from the room
					delete(currentRoomMembers.clients, leavingSubscription.name)
					//close the clients send channel so no more messages are sent to them
					close(leavingSubscription.sendChannel)
					//if the room has no more members and isn't the server lobby, delete the room
					if len(currentRoomMembers.clients) == 0 && currentRoomMembers.name != "server" {
						delete(h.rooms, currentRoomMembers.name)
					} else {
						h.updateRoomMembers(currentRoomMembers.name)
					}
				}
			}
		case message := <-h.messages:
			if err := h.parseMessage(message); err != nil {
				log.Printf("Error parsing message: %s", err)
			}
		}
	}
}

func (h *Hub) sendMessageToClient(message *Message) {
	select {
	case message.sender.sendChannel <- message:
		log.Printf("Sent message %v to client %s", message, message.sender.name)
	default:
		h.unregister <- message.sender
	}
}

func (h *Hub) sendMessageToRoom(message *Message) {

	for _, roomMember := range h.rooms[message.roomName].clients {
		if message.sender != nil && roomMember.name != message.sender.name {
			if !roomMember.isGivenClientMuted(message.sender.name) {
				select {
				case roomMember.sendChannel <- message:
					log.Printf("Sent message %v to client %s", message, roomMember.name)
				default:
					h.unregister <- roomMember
				}
			}
		}
	}
}

func (h *Hub) Register(conn *websocket.Conn, clientName string) *Client {
	userNumber, err := generateRandomUserNumber()
	if err != nil {
		log.Fatalln("Why are we not generating random numbers")
	}
	client := &Client{
		conn:         conn,
		room:         "server",
		name:         fmt.Sprintf("%s#%d", clientName, userNumber),
		mutedClients: []string{},
		sendChannel:  make(chan *Message, 1024),
	}

	h.register <- client

	return client
}

func (h *Hub) parseMessage(message *Message) error {
	log.Printf("Got message type: %s from client: %v to room %s", message.msg.MessageType, message.sender, message.roomName)
	switch message.msg.MessageType {
	case "NEW_MEMBER_MESSAGE":
		if h.rooms[message.roomName].clients != nil {
			for _, roomMember := range h.rooms[message.roomName].clients {
				log.Printf("Sending message type %s to client %s", message.msg.MessageType, roomMember.name)
				select {
				case roomMember.sendChannel <- message:
				default:
					h.unregister <- roomMember
				}
			}
		}
	case "SET_SCOPE_MESSAGE":
		log.Printf("received new scope from %s", message.sender.name)
		h.rooms[message.sender.room].scope = message.msg.Data
	case "GET_SCOPE_MESSAGE":
		log.Printf("%s requesting scope", message.sender.name)
		message.msg.Data = h.rooms[message.sender.room].scope
		h.sendMessageToClient(message)
	case "JOIN_ROOM_MESSAGE":
		roomTargetData := strings.Split(message.msg.Data, ":")
		roomServerPassword := h.rooms[roomTargetData[0]].password
		if len(roomServerPassword) > 0 {
			if roomServerPassword == roomTargetData[1] {
				h.clientRoomChangeHandler(message.sender, roomTargetData[0])
				//change response message type so client knows auth succeeded
				message.msg.MessageType = "GOOD_PASSWORD_MESSAGE"
				//send to client
				h.sendMessageToClient(message)
			} else {
				//bad password
				message.msg.MessageType = "BAD_PASSWORD_MESSAGE"
				h.sendMessageToClient(message)
			}
		} else {
			h.clientRoomChangeHandler(message.sender, roomTargetData[0])
		}
	case "LEAVE_ROOM_MESSAGE":
		log.Printf("%s leaving room: %s", message.sender.name, message.sender.room)
		h.clientRoomChangeHandler(message.sender, "server")
	case "ADD_ROOM_MESSAGE":
		roomTargetData := strings.Split(message.msg.Data, ":")
		//log.Printf("creating new room: %s with password : %s", roomTargetData[0], roomTargetData[1])
		if _, ok := h.rooms[roomTargetData[0]]; ok {
			message.msg.MessageType = "ROOM_EXISTS_MESSAGE"
			h.sendMessageToClient(message)
		} else {
			if len(roomTargetData) > 1 {
				h.rooms[roomTargetData[0]] = NewRoom(roomTargetData[0], roomTargetData[1])
			} else {
				h.rooms[roomTargetData[0]] = NewRoom(roomTargetData[0], "")
			}
			h.clientRoomChangeHandler(message.sender, roomTargetData[0])
			h.announceNewRooms()
		}
	case "MUTE_MESSAGE":
		if message.msg.Data == "All" {
			for _, roomMember := range h.rooms[message.sender.room].clients {
				if roomMember.name != message.sender.name {
					message.sender.mutedClients = append(message.sender.mutedClients, roomMember.name)
				}
			}
		} else {
			if message.msg.Data != message.sender.name {
				message.sender.mutedClients = append(message.sender.mutedClients, message.msg.Data)
			}
		}
		log.Printf("%s muted these clients %s", message.sender.name, message.sender.mutedClients)
	case "UNMUTE_MESSAGE":
		if message.msg.Data == "All" {
			for _, roomMember := range h.rooms[message.sender.room].clients {
				if roomMember.name != message.sender.name {
					message.sender.mutedClients = remove(message.sender.mutedClients, index(message.sender.mutedClients, roomMember.name))
				}
			}
		} else {
			if message.msg.Data != message.sender.name {
				message.sender.mutedClients = remove(message.sender.mutedClients, index(message.sender.mutedClients, message.msg.Data))
			}
		}
		log.Printf("%s unmuted %s", message.sender.name, message.msg.Data)
	case "GET_ROOMS_MESSAGE":

		rooms := h.rooms
		keys := make([]string, 0, len(rooms))
		for k := range rooms {
			if k != "server" {
				keys = append(keys, k+"::"+strconv.FormatBool(len(rooms[k].password) > 0))
			}
		}
		message.msg.Data = strings.Join(keys, ",")

		if message.sender == nil {
			//server announcing new rooms to lobby
			for _, lobbyMember := range h.rooms["server"].clients {
				lobbyMember.sendChannel <- message
			}
		} else {
			h.sendMessageToClient(message)
		}
	case "GET_CONFIG_MESSAGE":
		if h.shortenerService != nil {
			message.msg.Data = h.shortenerService.apiKey
		}
		h.sendMessageToClient(message)
	case "COOKIE_MESSAGE":
		fallthrough
	case "SCAN_ISSUE_MESSAGE":
		fallthrough
	case "REPEATER_MESSAGE":
		fallthrough
	case "INTRUDER_MESSAGE":
		fallthrough
	case "BURP_MESSAGE":
		h.sendMessageToRoom(message)
	default:
		return errors.New("ERROR: unknown message type")
	}
	return nil
}

func (h *Hub) announceNewRooms() {
	msg := NewBurpTCMessage()
	msg.MessageType = "GET_ROOMS_MESSAGE"
	h.messages <- generateMessage(msg, nil, "server")
}

func (h *Hub) updateRoomMembers(roomName string) {
	if roomName == "server" {
		return
	}
	msg := NewBurpTCMessage()
	msg.MessageType = "NEW_MEMBER_MESSAGE"

	keys := make([]string, 0, len(h.rooms[roomName].clients))
	for k := range h.rooms[roomName].clients {
		keys = append(keys, k)
	}

	if len(keys) > 0 {
		msg.Data = strings.Join(keys, ",")
		log.Printf("Current room (%s) members: %s", roomName, msg.Data)
		h.messages <- generateMessage(msg, nil, roomName)
	} else {
		log.Println("no room members to update")
		delete(h.rooms, roomName)
	}

}

func (h *Hub) clientRoomChangeHandler(clientChangingRooms *Client, newRoom string) {
	log.Printf("%s joining room: %s", clientChangingRooms.name, newRoom)
	//remove client from previous room
	delete(h.rooms[clientChangingRooms.room].clients, clientChangingRooms.name)
	//notify remaining room clients of leaving member
	h.updateRoomMembers(clientChangingRooms.room)
	//add them to the new room
	clientChangingRooms.room = newRoom
	h.rooms[newRoom].clients[clientChangingRooms.name] = clientChangingRooms
	//notify current room clients of new member
	h.updateRoomMembers(newRoom)
}

func (h *Hub) SetShortenerService(shortenerService *ShortenedUrls) {
	h.shortenerService = shortenerService
}
