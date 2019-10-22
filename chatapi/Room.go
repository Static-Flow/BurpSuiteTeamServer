package chatapi

import "sync"

type Room struct {
	scope   string
	clients map[string]*Client
	*sync.RWMutex
	comments Comments
}

func NewRoom() *Room {
	return &Room{
		"",
		make(map[string]*Client),
		new(sync.RWMutex),
		NewComments(),
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
	delete(r.clients, clientName)
}
