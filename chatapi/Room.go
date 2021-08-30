package chatapi

import (
	"sync"
)

type Room struct {
	scope   string
	clients map[*Client]bool
	*sync.RWMutex
	comments Comments
	password string
}

func NewRoom(password string) *Room {
	return &Room{
		"",
		make(map[*Client]bool),
		new(sync.RWMutex),
		NewComments(),
		password,
	}
}

func (r *Room) getAllComments() Comments {
	return r.comments
}

func (r *Room) getClient(clientName string) (*Client, bool) {
	r.Lock()
	defer r.Unlock()
	for client := range r.clients {
		if client.name == clientName {
			return client, true
		}
	}
	return nil, false
}

func (r *Room) addClient(c *Client) {
	r.Lock()
	defer r.Unlock()
	r.clients[c] = true
}

func (r *Room) deleteClient(client *Client) {
	r.Lock()
	defer r.Unlock()
	if _, ok := r.clients[client]; ok {
		delete(r.clients, client)
	}
}
