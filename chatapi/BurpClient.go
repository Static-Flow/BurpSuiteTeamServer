package chatapi

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"reflect"
	"strings"
	"time"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 8192 * 8192
)

var (
	newline = []byte{'\n'}
	space   = []byte{' '}
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  8192 * 8192,
	WriteBufferSize: 8192 * 8192,
}

// Client is a middleman between the websocket connection and the hub.
type Client struct {
	hub *Hub

	mutedClients []string

	name string

	// The websocket connection.
	conn *websocket.Conn

	// Buffered channel of outbound messages.
	send chan []byte
}

func (c *Client) parseMessage(message *BurpTCMessage, sub *subscription) {
	switch message.MessageType {
	case "SYNC_SCOPE_MESSAGE":
		if message.MessageTarget != "Self" {
			log.Printf("received new scope from %s", c.name)
			c.hub.rooms[sub.roomName].scope = message.Data
		} else {
			log.Printf("%s requesting scope", c.name)
			message.Data = c.hub.rooms[sub.roomName].scope
		}
		c.hub.broadcast <- c.hub.generateMessage(message, c, sub.roomName, message.MessageTarget)
	case "JOIN_ROOM_MESSAGE":
		log.Printf("%s joining room: %s", c.name, message.MessageTarget)
		c.hub.rooms[sub.roomName].deleteClient(c.name)
		c.hub.rooms[message.MessageTarget].addClient(sub.conn)
		sub.roomName = message.MessageTarget
		if len(c.hub.rooms[sub.roomName].clients) == 0 {
			c.hub.deleteRoom(sub.roomName)
		} else {
			c.updateRoomMembers(sub.roomName)
		}
	case "LEAVE_ROOM_MESSAGE":
		log.Printf("leaving room: %s", sub.roomName)
		c.hub.rooms[sub.roomName].deleteClient(c.name)
		c.hub.rooms["server"].addClient(sub.conn)
		if len(c.hub.rooms[sub.roomName].clients) == 0 {
			c.hub.deleteRoom(sub.roomName)
		} else {
			c.updateRoomMembers(sub.roomName)
		}
		sub.roomName = "server"
	case "ADD_ROOM_MESSAGE":
		log.Printf("creating new room: " + message.MessageTarget)
		c.hub.rooms[sub.roomName].deleteClient(c.name)
		c.hub.addRoom(message.MessageTarget, NewRoom())
		c.hub.rooms[message.MessageTarget].addClient(sub.conn)
		sub.roomName = message.MessageTarget
		c.updateRoomMembers(message.MessageTarget)
	case "MUTE_MESSAGE":
		if message.MessageTarget == "All" {
			keys := reflect.ValueOf(c.hub.rooms[sub.roomName].clients).MapKeys()
			log.Printf("Clients to mute %s", keys)
			for i := 0; i < len(keys); i++ {
				key := keys[i].String()
				if key != c.name {
					c.mutedClients = append(c.mutedClients, key)
				}
			}
		} else {
			log.Printf("Client to mute %s", message.MessageTarget)
			c.mutedClients = append(c.mutedClients, message.MessageTarget)
		}
		log.Printf("%s muted these clients %s", c.name, c.mutedClients)
	case "UNMUTE_MESSAGE":
		if message.MessageTarget == "All" {
			keys := reflect.ValueOf(c.hub.rooms[sub.roomName].clients).MapKeys()
			for i := 0; i < len(keys); i++ {
				key := keys[i].String()
				if key != c.name {
					c.mutedClients = remove(c.mutedClients, index(c.mutedClients, keys[i].String()))
				}
			}
		} else {
			c.mutedClients = remove(c.mutedClients, index(c.mutedClients, message.MessageTarget))
		}
		log.Printf("%s unmuted %s", c.name, message.MessageTarget)
	case "GET_ROOMS_MESSAGE":
		rooms := c.hub.rooms
		keys := make([]string, 0, len(rooms))
		for k := range rooms {
			keys = append(keys, k)
		}
		message.Data = strings.Join(keys, ",")
		c.hub.broadcast <- c.hub.generateMessage(message, c, sub.roomName, message.MessageTarget)
	case "COOKIE_MESSAGE":
		fallthrough
	case "SCAN_ISSUE_MESSAGE":
		fallthrough
	case "REPEATER_MESSAGE":
		fallthrough
	case "INTRUDER_MESSAGE":
		fallthrough
	case "BURP_MESSAGE":
		c.hub.broadcast <- c.hub.generateMessage(message, c, sub.roomName, message.MessageTarget)
	default:
		log.Println("ERROR: unknown message type")
	}
}

func remove(s []string, i int) []string {
	s[i] = s[len(s)-1]
	return s[:len(s)-1]
}

func index(vs []string, t string) int {

	for i, v := range vs {
		if v == t {
			return i
		}
	}
	return -1
}

func (c *Client) isGivenClientMuted(clientName string) bool {
	for _, client := range c.mutedClients {
		if client == clientName {
			return true
		}
	}
	return false
}

// readPump pumps messages from the websocket connection to the hub.
//
// The application runs readPump in a per-connection goroutine. The application
// ensures that there is at most one reader on a connection by executing all
// reads from this goroutine.
func (s *subscription) readPump() {
	c := s.conn
	defer func() {
		c.hub.unregister <- s
		c.conn.Close()
	}()
	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error { c.conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	for {
		_, msg, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
			break
		}
		log.Println(msg)
		newBurpMessage := NewBurpTCMessage()
		decodedBytes := make([]byte, base64.StdEncoding.DecodedLen(len(msg)))
		base64.StdEncoding.Decode(decodedBytes, msg)
		log.Println(string(decodedBytes))
		if err := json.Unmarshal(bytes.Trim(decodedBytes, "\x00"), &newBurpMessage); err != nil {
			log.Printf("Could not unmarshal BurpTCMessage, error: %s \n", err)
		} else {
			c.parseMessage(newBurpMessage, s)
		}
	}
}

func (c *Client) write(mt int, payload []byte) error {
	c.conn.SetWriteDeadline(time.Now().Add(writeWait))
	return c.conn.WriteMessage(mt, payload)
}

func (c *Client) updateRoomMembers(roomName string) {
	c.hub.rooms[roomName].Lock()
	defer c.hub.rooms[roomName].Unlock()
	msg := NewBurpTCMessage()
	msg.MessageType = "NEW_MEMBER_MESSAGE"

	keys := make([]string, 0, len(c.hub.rooms[roomName].clients))
	for k := range c.hub.rooms[roomName].clients {
		keys = append(keys, k)
	}
	log.Printf("Current room members: %s", strings.Join(keys, ","))
	msg.Data = strings.Join(keys, ",")
	c.hub.broadcast <- c.hub.generateMessage(msg, nil, roomName, "Room")
}

// writePump pumps messages from the hub to the websocket connection.
//
// A goroutine running writePump is started for each connection. The
// application ensures that there is at most one writer to a connection by
// executing all writes from this goroutine.
func (s *subscription) writePump() {
	c := s.conn
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()
	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The hub closed the channel.
				c.write(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.write(websocket.TextMessage, message); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.write(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// serveWs handles websocket requests from the peer.
func ServeWs(hub *Hub, w http.ResponseWriter, r *http.Request) {
	if authHeader := r.Header.Get("Auth"); authHeader == hub.serverPassword {
		if _, ok := hub.rooms["server"].getClient(r.Header.Get("Username")); ok {
			log.Println("Found duplicate name")
			w.WriteHeader(http.StatusConflict)
			w.Write([]byte("409 - Duplicate name in server!"))
		} else {
			conn, err := upgrader.Upgrade(w, r, nil)
			if err != nil {
				log.Println(err)
				return
			}
			client := &Client{hub: hub, conn: conn, send: make(chan []byte, 8196*8196), name: r.Header.Get("Username")}
			s := subscription{client, "server"}
			hub.register <- &s

			// Allow collection of memory referenced by the caller by doing all work in
			// new goroutines.
			go s.writePump()
			go s.readPump()
		}

	} else {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("401 - Bad Auth!"))
	}
}
