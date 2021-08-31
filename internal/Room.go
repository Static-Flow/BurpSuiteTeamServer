package internal

import (
	"encoding/json"
	"log"
	"strings"
	"sync"
)

type Room struct {
	scope   string
	clients map[string]*Client
	*sync.RWMutex
	comments  Comments
	password  string
	serverHub *ServerHub
	name      string
}

func NewRoom(serverHub *ServerHub, password string, roomName string) *Room {
	return &Room{
		"",
		make(map[string]*Client),
		new(sync.RWMutex),
		NewComments(),
		password,
		serverHub,
		roomName,
	}
}

func (r *Room) getAllComments() Comments {
	return r.comments
}

func (r *Room) getClient(clientName string) (*Client, bool) {
	r.Lock()
	defer r.Unlock()
	if client, ok := r.clients[clientName]; ok {
		return client, ok
	}
	return nil, false
}

func (r *Room) addClient(c *Client) {
	r.Lock()
	defer r.Unlock()
	r.clients[c.name] = c
}

func (r *Room) deleteClient(clientName string) {
	r.Lock()
	defer r.Unlock()
	if _, ok := r.clients[clientName]; ok {
		delete(r.clients, clientName)
	}
}

func (r *Room) updateRoomMembers() {
	r.Lock()
	defer r.Unlock()
	msg := NewBurpTCMessage()
	msg.MessageType = "NEW_MEMBER_MESSAGE"

	keys := make([]string, 0, len(r.clients))
	for k := range r.clients {
		keys = append(keys, k)
	}
	if len(keys) > 0 {
		log.Printf("Current room members: %s", strings.Join(keys, ","))
		msg.Data = strings.Join(keys, ",")
		r.serverHub.messages <- generateMessage(msg, nil, r.name, "Room")
	} else {
		log.Println("no room members to update")
	}

}

func (r *Room) updateRequestResponseComments(tcMessage *BurpTCMessage) {
	log.Printf("Got comment message: " + tcMessage.String())
	r.comments.setRequestWithComments(tcMessage.Data, *tcMessage.BurpRequestResponse)
	log.Printf("%d comments in room", len(r.comments.requestsWithComments))
	r.serverHub.messages <- generateMessage(tcMessage, nil, r.name, tcMessage.MessageTarget)
}

func (r *Room) sendRoomMessagesToNewRoomMember(clientName string) {
	if len(r.comments.requestsWithComments) != 0 {
		r.Lock()
		defer r.Unlock()
		client := r.clients[clientName]
		msg := NewBurpTCMessage()
		msg.MessageType = "GET_COMMENTS_MESSAGE"

		burpMessagesWithComments := make([]BurpRequestResponse, len(r.comments.requestsWithComments))
		idx := 0
		for _, value := range r.comments.requestsWithComments {
			burpMessagesWithComments[idx] = value
			log.Printf(value.String())
			idx++
		}

		log.Printf("Sending %d current room messages to %s", len(burpMessagesWithComments), clientName)
		jsonData, err := json.Marshal(burpMessagesWithComments)
		if err != nil {
			log.Println(err)
		} else {
			msg.Data = string(jsonData)
			r.serverHub.messages <- generateMessage(msg, client, r.name, "Self")
		}
	}
}
