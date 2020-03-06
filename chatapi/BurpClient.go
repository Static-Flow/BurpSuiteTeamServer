package chatapi

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"reflect"
	"strconv"
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

var upgrader = websocket.Upgrader{
	ReadBufferSize:  8192 * 8192,
	WriteBufferSize: 8192 * 8192,
}

// Client is a middleman between the websocket connection and the hub.
type Client struct {
	hub *Hub

	authenticated bool

	mutedClients []string

	roomName string

	name string

	// The websocket connection.
	conn *websocket.Conn

	// Buffered channel of outbound messages.
	send chan []byte
}

func (c *Client) parseMessage(message *BurpTCMessage) {
	switch message.MessageType {
	case "SYNC_SCOPE_MESSAGE":
		if message.MessageTarget != "Self" {
			log.Printf("received new scope from %s", c.name)
			c.hub.rooms[c.roomName].scope = message.Data
		} else {
			log.Printf("%s requesting scope", c.name)
			message.Data = c.hub.rooms[c.roomName].scope
		}
		c.hub.broadcast <- c.hub.generateMessage(message, c, c.roomName, message.MessageTarget)
	case "JOIN_ROOM_MESSAGE":
		if len(c.hub.rooms[message.MessageTarget].password) > 0 {
			if c.authenticated {
				log.Printf("%s joining room: %s", c.name, message.MessageTarget)
				c.hub.rooms[c.roomName].deleteClient(c.name)
				c.hub.rooms[message.MessageTarget].addClient(c)
				c.roomName = message.MessageTarget
				c.hub.updateRoomMembers(c.roomName)
				c.sendRoomMessages()
			} else {
				//bad password
				message.MessageType = "BAD_PASSWORD_MESSAGE"
				c.hub.broadcast <- c.hub.generateMessage(message, c, c.roomName, "Self")
			}

		} else {
			log.Printf("%s joining room: %s", c.name, message.MessageTarget)
			c.hub.rooms[c.roomName].deleteClient(c.name)
			c.hub.rooms[message.MessageTarget].addClient(c)
			c.roomName = message.MessageTarget
			c.hub.updateRoomMembers(c.roomName)
			c.sendRoomMessages()

		}
	case "LEAVE_ROOM_MESSAGE":
		log.Printf("leaving room: %s", c.roomName)
		c.hub.rooms[c.roomName].deleteClient(c.name)
		c.hub.rooms["server"].addClient(c)
		if len(c.hub.rooms[c.roomName].clients) == 0 {
			c.hub.deleteRoom(c.roomName)
		} else {
			c.hub.updateRoomMembers(c.roomName)
		}
		c.hub.updateRooms()
		c.authenticated = false
		c.roomName = "server"
	case "ADD_ROOM_MESSAGE":
		log.Printf("creating new room: " + message.MessageTarget)
		c.hub.rooms[c.roomName].deleteClient(c.name)
		c.hub.addRoom(message.MessageTarget, NewRoom(message.Data))
		c.hub.rooms[message.MessageTarget].addClient(c)
		c.roomName = message.MessageTarget
		c.hub.updateRoomMembers(c.roomName)
		c.sendRoomMessages()
	case "MUTE_MESSAGE":
		if message.MessageTarget == "All" {
			keys := reflect.ValueOf(c.hub.rooms[c.roomName].clients).MapKeys()
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
			keys := reflect.ValueOf(c.hub.rooms[c.roomName].clients).MapKeys()
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
			if k != "server" {
				keys = append(keys, k+"::"+strconv.FormatBool(len(rooms[k].password) > 0))
			}
		}
		message.Data = strings.Join(keys, ",")
		c.hub.broadcast <- c.hub.generateMessage(message, c, c.roomName, message.MessageTarget)
	case "COMMENT_MESSAGE":
		log.Printf("Got comment message: " + message.String())
		c.hub.rooms[c.roomName].comments.setRequestWithComments(message.Data, *message.BurpRequestResponse)
		log.Printf("%d comments in room", len(c.hub.rooms[c.roomName].comments.requestsWithComments))
		c.hub.broadcast <- c.hub.generateMessage(message, c, c.roomName, message.MessageTarget)
	case "CHECK_PASSWORD_MESSAGE":
		if c.hub.rooms[message.MessageTarget].password == message.Data {
			log.Printf("%s supplied orrect password for room: %s", c.name, message.MessageTarget)
			c.authenticated = true
			message.MessageType = "GOOD_PASSWORD_MESSAGE"
			c.hub.broadcast <- c.hub.generateMessage(message, c, c.roomName, "Self")
		} else {
			//bad password
			message.MessageType = "BAD_PASSWORD_MESSAGE"
			c.hub.broadcast <- c.hub.generateMessage(message, c, c.roomName, "Self")
		}
	case "GET_CONFIG_MESSAGE":
		message.MessageType = "GET_CONFIG_MESSAGE"
		message.Data = c.hub.GetUrlShortenerApiKey()
		c.hub.broadcast <- c.hub.generateMessage(message, c, c.roomName, message.MessageTarget)
	case "COOKIE_MESSAGE":
		fallthrough
	case "SCAN_ISSUE_MESSAGE":
		fallthrough
	case "REPEATER_MESSAGE":
		fallthrough
	case "INTRUDER_MESSAGE":
		fallthrough
	case "BURP_MESSAGE":
		c.hub.broadcast <- c.hub.generateMessage(message, c, c.roomName, message.MessageTarget)
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
func (c *Client) readPump() {
	defer func() {
		c.hub.rooms[c.roomName].deleteClient(c.name)
		if c.roomName != "server" {
			if len(c.hub.rooms[c.roomName].clients) == 0 {
				c.hub.deleteRoom(c.roomName)
			} else {
				c.hub.updateRoomMembers(c.roomName)
			}
			c.roomName = "server"
		}
		close(c.send)
		c.hub.removeClientFromServerList(c.name)
		log.Println("Client Leaving")
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
		newBurpMessage := NewBurpTCMessage()
		decodedBytes := make([]byte, base64.StdEncoding.DecodedLen(len(msg)))
		base64.StdEncoding.Decode(decodedBytes, msg)
		log.Printf(string(decodedBytes))
		if err := json.Unmarshal(bytes.Trim(decodedBytes, "\x00"), &newBurpMessage); err != nil {
			log.Printf("Could not unmarshal BurpTCMessage, error: %s \n", err)
		} else {
			c.parseMessage(newBurpMessage)
		}
	}
}

func (c *Client) write(mt int, payload []byte) error {
	c.conn.SetWriteDeadline(time.Now().Add(writeWait))
	return c.conn.WriteMessage(mt, payload)
}

func (c *Client) sendRoomMessages() {
	if len(c.hub.rooms[c.roomName].comments.requestsWithComments) != 0 {
		c.hub.rooms[c.roomName].Lock()
		defer c.hub.rooms[c.roomName].Unlock()
		msg := NewBurpTCMessage()
		msg.MessageType = "GET_COMMENTS_MESSAGE"

		burpMessagesWithComments := make([]BurpRequestResponse, len(c.hub.rooms[c.roomName].comments.requestsWithComments))
		idx := 0
		for _, value := range c.hub.rooms[c.roomName].comments.requestsWithComments {
			burpMessagesWithComments[idx] = value
			log.Printf(value.String())
			idx++
		}

		log.Printf("Sending %d current room messages to %s", len(burpMessagesWithComments), c.name)
		jsonData, err := json.Marshal(burpMessagesWithComments)
		if err != nil {
			log.Println(err)
		} else {
			msg.Data = string(jsonData)
			c.hub.broadcast <- c.hub.generateMessage(msg, c, c.roomName, "Self")
		}
	}
}

// writePump pumps messages from the hub to the websocket connection.
//
// A goroutine running writePump is started for each connection. The
// application ensures that there is at most one writer to a connection by
// executing all writes from this goroutine.
func (c *Client) writePump() {
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
		if hub.clientExistsInServer(r.Header.Get("Username")) {
			log.Println("Found duplicate name")
			w.WriteHeader(http.StatusConflict)
			w.Write([]byte("409 - Duplicate name in server!"))
		} else {
			conn, err := upgrader.Upgrade(w, r, nil)
			if err != nil {
				log.Println(err)
				return
			}
			client := &Client{hub: hub, conn: conn, send: make(chan []byte, 8196*8196),
				name: r.Header.Get("Username"), roomName: "server", authenticated: false}
			//s := subscription{client, "server"}
			hub.register <- client

			// Allow collection of memory referenced by the caller by doing all work in
			// new goroutines.
			go client.writePump()
			go client.readPump()
		}

	} else {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("401 - Bad Auth!"))
	}
}
