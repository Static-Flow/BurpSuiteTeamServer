package chatapi

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/fasthttp/websocket"
	"github.com/valyala/fasthttp"
	"log"
	"net/http"
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

var upgrader = websocket.FastHTTPUpgrader{
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
				c.hub.rooms[c.roomName].deleteClient(c)
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
			c.hub.rooms[c.roomName].deleteClient(c)
			c.hub.rooms[message.MessageTarget].addClient(c)
			c.roomName = message.MessageTarget
			c.hub.updateRoomMembers(c.roomName)
			c.sendRoomMessages()

		}
	case "LEAVE_ROOM_MESSAGE":
		log.Printf("leaving room: %s", c.roomName)
		c.hub.rooms[c.roomName].deleteClient(c)
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
		c.hub.rooms[c.roomName].deleteClient(c)
		c.hub.addRoom(message.MessageTarget, NewRoom(message.Data))
		c.hub.rooms[message.MessageTarget].addClient(c)
		c.roomName = message.MessageTarget
		c.hub.updateRoomMembers(c.roomName)
		c.sendRoomMessages()
	case "MUTE_MESSAGE":
		if message.MessageTarget == "All" {
			for roomClient := range c.hub.rooms[c.roomName].clients {
				if roomClient != c {
					c.mutedClients = append(c.mutedClients, roomClient.name)
				}
			}
		} else {
			log.Printf("Client to mute %s", message.MessageTarget)
			c.mutedClients = append(c.mutedClients, message.MessageTarget)
		}
		log.Printf("%s muted these clients %s", c.name, c.mutedClients)
	case "UNMUTE_MESSAGE":
		if message.MessageTarget == "All" {
			c.mutedClients = c.mutedClients[:0]
		} else {
			for i, mutedClient := range c.mutedClients {
				if mutedClient == message.MessageTarget {
					c.mutedClients = append(c.mutedClients[:i], c.mutedClients[i+1:]...)
					break
				}
			}
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

		//If there are already comments in the room
		if len(c.hub.rooms[c.roomName].comments.getRequestWithComments(message.Data).Comments) > 0 {
			//If the incoming request with comments has less comments than the old one we are deleting a comment
			if len(c.hub.rooms[c.roomName].comments.getRequestWithComments(message.Data).Comments) > len(message.BurpRequestResponse.Comments) {
				/*
					oldComments : A, B
					newComments: A
				*/
				log.Println("Deleting Comments")
				if len(c.hub.rooms[c.roomName].comments.getRequestWithComments(message.Data).Comments) > 1 {
					for _, oldComment := range c.hub.rooms[c.roomName].comments.getRequestWithComments(message.Data).Comments {
						for _, newComment := range message.BurpRequestResponse.Comments {
							if oldComment != newComment {
								if oldComment.UserWhoCommented == c.name {
									c.updateRequestResponseComments(message)
								} else {
									log.Printf("User " + c.name + " cannot delete comment by " + oldComment.UserWhoCommented)
								}
							}
						}
					}
				} else {
					//if the last comment for the request is authored by the sender they can delete it
					if c.hub.rooms[c.roomName].comments.getRequestWithComments(message.Data).Comments[0].UserWhoCommented == c.name {
						c.updateRequestResponseComments(message)
					} else {
						log.Printf("User " + c.name + " cannot delete comment by another user")
					}
				}
			} else {
				//If the incoming request with comments has more comments than the old one we are adding a comment
				/*
					oldComments : A
					newComments: A, B
				*/
				log.Println("Adding Comments")
				for _, oldComment := range c.hub.rooms[c.roomName].comments.getRequestWithComments(message.Data).Comments {
					for _, newComment := range message.BurpRequestResponse.Comments {
						if oldComment != newComment {
							if newComment.UserWhoCommented == c.name {
								c.updateRequestResponseComments(message)
							} else {
								log.Printf("User " + c.name + " cannot add comment by " + newComment.UserWhoCommented)
							}
						}
					}
				}
			}
		} else {
			c.updateRequestResponseComments(message)
		}

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
		message.Data = fmt.Sprintf("%s,%s", c.hub.GetUrlShortenerApiKey(), c.hub.GetServerPort())
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

func (c *Client) updateRequestResponseComments(tcMessage *BurpTCMessage) {
	log.Printf("Got comment message: " + tcMessage.String())
	c.hub.rooms[c.roomName].comments.setRequestWithComments(tcMessage.Data, *tcMessage.BurpRequestResponse)
	log.Printf("%d comments in room", len(c.hub.rooms[c.roomName].comments.requestsWithComments))
	c.hub.broadcast <- c.hub.generateMessage(tcMessage, c, c.roomName, tcMessage.MessageTarget)
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
		c.hub.rooms[c.roomName].deleteClient(c)
		if c.roomName != "server" {
			if len(c.hub.rooms[c.roomName].clients) == 0 {
				c.hub.deleteRoom(c.roomName)
			} else {
				c.hub.updateRoomMembers(c.roomName)
			}
			c.roomName = "server"
		}
		close(c.send)
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
			} else {
				log.Printf("All Errors: %v", err)
			}
			break
		}
		newBurpMessage := NewBurpTCMessage()
		decodedBytes := make([]byte, base64.StdEncoding.DecodedLen(len(msg)))
		_, err = base64.StdEncoding.Decode(decodedBytes, msg)
		if err != nil {
			log.Printf("error decoding base64: %v", err)
		}
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
func ServeWs(ctx *fasthttp.RequestCtx, hub *Hub) {
	if authHeader := ctx.Request.Header.Peek("Auth"); bytes.Equal(authHeader, hub.serverPassword) {
		user := string(ctx.Request.Header.Peek("Username"))
		err := upgrader.Upgrade(ctx, func(conn *websocket.Conn) {
			fmt.Printf("Client joining: %s\n", user)
			client := &Client{hub: hub, conn: conn, send: make(chan []byte, 1024),
				name: user, roomName: "server", authenticated: false}
			//s := subscription{client, "server"}
			hub.register <- client

			// Allow collection of memory referenced by the caller by doing all work in
			// new goroutines.
			go client.writePump()
			client.readPump()
		})
		if err != nil {
			log.Println(err)
			return
		}

	} else {
		ctx.Response.SetStatusCode(http.StatusUnauthorized)
		ctx.SetBody([]byte("401 - Bad Auth!"))
	}
}
