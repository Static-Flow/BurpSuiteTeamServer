package chatapi

import (
	"io"
	"log"
	"reflect"
	"strings"
	"sync"
)

//Room type represents a chat room
type Room struct {
	scope   []string `json:"scope"`
	name    string   `json:"name"`
	Msgch   chan BurpTCMessage
	clients map[string]*client
	//signals the quitting of the chat room
	Quit chan struct{}
	*sync.RWMutex
}

//CreateRoom starts a new chat room with name rname
func CreateRoom(rname string) *Room {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	r := &Room{
		name:    rname,
		Msgch:   make(chan BurpTCMessage),
		RWMutex: new(sync.RWMutex),
		clients: make(map[string]*client),
		Quit:    make(chan struct{}),
	}
	r.Run()
	return r
}

//AddClient adds a new client to the chat room
func (r *Room) AddClient(c io.ReadWriteCloser, clientname string, serverPassword string) {
	r.Lock()
	defer r.Unlock()
	if _, ok := r.clients[clientname]; ok {
		log.Printf("Client %s already exist in chat room %s, existing...", clientname, r.name)
		return
	} else {
		log.Printf("Adding client %s \n", clientname)
		wc, done := StartClient(serverPassword, r.Msgch, c, r.name)
		r.clients[clientname] = wc
		go func() {
			<-done
			r.RemoveClientSync(clientname)
		}()
		r.updateRoomMembers()
	}
}

func (r *Room) updateRoomMembers() {
	msg := NewBurpTCMessage()
	msg.MessageType = "NEW_MEMBER_MESSAGE"
	keys := reflect.ValueOf(r.clients).MapKeys()
	strkeys := make([]string, len(keys))
	for i := 0; i < len(keys); i++ {
		strkeys[i] = keys[i].String()
	}
	log.Printf("Current clients: %s", strings.Join(strkeys, ","))
	msg.Data = strings.Join(strkeys, ",")
	for _, wc := range r.clients {
		go func(wc chan<- string) {
			wc <- SendMessage(*msg)
		}(wc.wc)
	}
}

//ClCount returns the number of clients in a chat room
func (r *Room) ClCount() int {
	return len(r.clients)
}

//RemoveClientSync removes a client from the chat room. This is a blocking call
func (r *Room) RemoveClientSync(name string) {
	log.SetFlags(log.Ltime | log.Lmicroseconds)
	r.Lock()
	defer r.Unlock()
	delete(r.clients, name)
	r.updateRoomMembers()
}

//Run runs a chat room
func (r *Room) Run() {
	log.Println("Starting chat room", r.name)
	//handle the chat room BurpSuiteTeamServer message channel
	go func() {
		for msg := range r.Msgch {
			r.broadcastMsg(msg)
		}
	}()

	//handle when the quit channel is triggered
	go func() {
		<-r.Quit
		r.CloseChatRoomSync()
	}()
}

//CloseChatRoomSync closes a chat room. This is a blocking call
func (r *Room) CloseChatRoomSync() {
	r.Lock()
	defer r.Unlock()
	close(r.Msgch)
	for name := range r.clients {
		delete(r.clients, name)
	}
}

//fan out is used to distribute the chat message
func (r *Room) broadcastMsg(msg BurpTCMessage) {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	r.RLock()
	defer r.RUnlock()
	log.Printf("Sending message from %s to %s", msg.SendingUser, msg.MessageTarget)
	if msg.MessageTarget != "room" {
		r.clients[msg.MessageTarget].wc <- SendMessage(msg)
	} else {
		for clientName, wc := range r.clients {
			if index(wc.mutedClients, msg.SendingUser) == -1 {
				if msg.SendingUser != clientName {
					go func(wc chan<- string) {
						wc <- SendMessage(msg)
					}(wc.wc)
				}
			}
		}
	}
}
