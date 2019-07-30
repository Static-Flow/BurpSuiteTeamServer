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
	Scope   string `json:"scope"`
	Name    string `json:"name"`
	Msgch   chan BurpTCMessage
	clients map[string]*client
	//signals the quitting of the chat room
	Quit chan struct{}
	*sync.RWMutex
	crypter AESCrypter
}

//CreateRoom starts a new chat room with name rname
func CreateRoom(rname string, aesCrypter AESCrypter) *Room {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	r := &Room{
		Name:    rname,
		Msgch:   make(chan BurpTCMessage),
		RWMutex: new(sync.RWMutex),
		clients: make(map[string]*client),
		Quit:    make(chan struct{}),
		crypter: aesCrypter,
		Scope:   "",
	}
	r.Run()
	return r
}

//AddClient adds a new client to the chat room
func (r *Room) AddClient(c io.ReadWriteCloser, clientname string, chatApi *ChatAPI) {
	r.Lock()
	defer r.Unlock()
	if _, ok := r.clients[clientname]; ok {
		log.Printf("Client %s already exist in chat room %s, existing...", clientname, r.Name)
		return
	} else {
		log.Printf("Adding client %s \n", clientname)
		wc, done := StartClient(clientname, chatApi, r.Msgch, c, r.Name)
		r.clients[clientname] = wc
		go func() {
			<-done
			r.RemoveClientSync(clientname)
		}()
	}
}

func (r *Room) AddExistingClient(client *client) {
	r.Lock()
	defer r.Unlock()
	r.clients[client.Name] = client
	if r.Name != "server" {
		r.updateRoomMembers()
	}
}

func (r *Room) updateRoomMembers() {
	if r.Name != "server" {
		msg := NewBurpTCMessage()
		msg.MessageType = "NEW_MEMBER_MESSAGE"
		keys := reflect.ValueOf(r.clients).MapKeys()
		strkeys := make([]string, len(keys))
		for i := 0; i < len(keys); i++ {
			strkeys[i] = keys[i].String()
		}
		log.Printf("Current clients in room %s: %s", r.Name, strings.Join(strkeys, ","))
		msg.Data = strings.Join(strkeys, ",")
		for _, wc := range r.clients {
			go func(wc chan<- string) {
				wc <- SendMessage(*msg, r.crypter)
			}(wc.wc)
		}
	}
}

//RemoveClientSync removes a client from the chat room. This is a blocking call
func (r *Room) RemoveClientSync(name string) {
	r.Lock()
	defer r.Unlock()
	delete(r.clients, name)
	r.updateRoomMembers()
}

//Run runs a chat room
func (r *Room) Run() {
	log.Println("Starting chat room", r.Name)
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
	log.Printf("Closing room: %s", r.Name)
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
	log.Printf("Sending message from %s to %s in room: %s", msg.SendingUser, msg.MessageTarget, r.Name)
	if msg.MessageTarget == "me" {
		r.clients[msg.SendingUser].wc <- SendMessage(msg, r.crypter)
	} else if msg.MessageTarget != "room" {
		r.clients[msg.MessageTarget].wc <- SendMessage(msg, r.crypter)
	} else {
		for clientName, wc := range r.clients {
			if index(wc.mutedClients, msg.SendingUser) == -1 {
				if msg.SendingUser != clientName {
					go func(wc chan<- string) {
						wc <- SendMessage(msg, r.crypter)
					}(wc.wc)
				}
			}
		}
	}
}
