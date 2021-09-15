package internal

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"github.com/fasthttp/websocket"
	"log"
	"reflect"
	"strconv"
	"strings"
	"time"
)

type Client struct {
	conn         *websocket.Conn
	roomName     string
	sendChannel  chan Message
	mutedClients []string
	name         string
	serverHub    *ServerHub
}

func (c *Client) isGivenClientMuted(clientName string) bool {
	for _, client := range c.mutedClients {
		if client == clientName {
			return true
		}
	}
	return false
}

func (c *Client) parseMessage(message *BurpTCMessage) error {
	//log.Printf("Got message: %+v from client: %s\n",*message,c.name)
	switch message.MessageType {
	case "SYNC_SCOPE_MESSAGE":
		if message.MessageTarget != "Self" {
			log.Printf("received new scope from %s", c.name)
			c.serverHub.rooms[c.roomName].scope = message.Data
		} else {
			log.Printf("%s requesting scope", c.name)
			message.Data = c.serverHub.rooms[c.roomName].scope
		}
		c.serverHub.messages <- generateMessage(message, c, c.roomName, message.MessageTarget)
	case "JOIN_ROOM_MESSAGE":
		if len(c.serverHub.rooms[message.MessageTarget].password) > 0 {
			if c.serverHub.rooms[message.MessageTarget].password == message.Data {
				log.Printf("%s joining room: %s", c.name, message.MessageTarget)
				c.serverHub.rooms[c.roomName].deleteClient(c.name)
				c.serverHub.rooms[message.MessageTarget].addClient(c)
				c.serverHub.rooms[c.roomName].sendRoomMessagesToNewRoomMember(c.name)
				message.MessageType = "GOOD_PASSWORD_MESSAGE"
				c.serverHub.messages <- generateMessage(message, c, c.roomName, "Self")
			} else {
				//bad password
				message.MessageType = "BAD_PASSWORD_MESSAGE"
				c.serverHub.messages <- generateMessage(message, c, c.roomName, "Self")
			}

		} else {
			log.Printf("%s joining room: %s", c.name, message.MessageTarget)
			c.serverHub.rooms[c.roomName].deleteClient(c.name)
			c.serverHub.rooms[message.MessageTarget].addClient(c)
			c.serverHub.rooms[c.roomName].sendRoomMessagesToNewRoomMember(c.name)

		}
	case "LEAVE_ROOM_MESSAGE":
		log.Printf("%s leaving room: %s", c.name, c.roomName)
		c.serverHub.rooms[c.roomName].deleteClient(c.name)
		if len(c.serverHub.rooms[c.roomName].clients) == 0 {
			c.serverHub.deleteRoom(c.roomName)
		}
		c.serverHub.rooms["server"].addClient(c)
	case "ADD_ROOM_MESSAGE":
		log.Printf("creating new room: " + message.MessageTarget)
		c.serverHub.rooms[c.roomName].deleteClient(c.name)
		c.serverHub.addRoom(message.Data, message.MessageTarget)
		c.serverHub.rooms[message.MessageTarget].addClient(c)
	case "MUTE_MESSAGE":
		if message.MessageTarget == "All" {
			keys := reflect.ValueOf(c.serverHub.rooms[c.roomName].clients).MapKeys()
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
			keys := reflect.ValueOf(c.serverHub.rooms[c.roomName].clients).MapKeys()
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
		rooms := c.serverHub.rooms
		keys := make([]string, 0, len(rooms))
		for k := range rooms {
			if k != "server" {
				keys = append(keys, k+"::"+strconv.FormatBool(len(rooms[k].password) > 0))
			}
		}
		message.Data = strings.Join(keys, ",")
		//log.Printf("Rooms message: %+v\n",message)
		c.serverHub.messages <- generateMessage(message, c, c.roomName, message.MessageTarget)
	case "COMMENT_MESSAGE":

		//If there are already comments in the room
		if len(c.serverHub.rooms[c.roomName].comments.getRequestWithComments(message.Data).Comments) > 0 {
			//If the incoming request with comments has less comments than the old one we are deleting a comment
			if len(c.serverHub.rooms[c.roomName].comments.getRequestWithComments(message.Data).Comments) > len(message.BurpRequestResponse.Comments) {
				/*
					oldComments : A, B
					newComments: A
				*/
				log.Println("Deleting Comments")
				if len(c.serverHub.rooms[c.roomName].comments.getRequestWithComments(message.Data).Comments) > 1 {
					for _, oldComment := range c.serverHub.rooms[c.roomName].comments.getRequestWithComments(message.Data).Comments {
						for _, newComment := range message.BurpRequestResponse.Comments {
							if oldComment != newComment {
								if oldComment.UserWhoCommented == c.name {
									c.serverHub.rooms[c.roomName].updateRequestResponseComments(message)
								} else {
									log.Printf("User " + c.name + " cannot delete comment by " + oldComment.UserWhoCommented)
								}
							}
						}
					}
				} else {
					//if the last comment for the request is authored by the sender they can delete it
					if c.serverHub.rooms[c.roomName].comments.getRequestWithComments(message.Data).Comments[0].UserWhoCommented == c.name {
						c.serverHub.rooms[c.roomName].updateRequestResponseComments(message)
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
				for _, oldComment := range c.serverHub.rooms[c.roomName].comments.getRequestWithComments(message.Data).Comments {
					for _, newComment := range message.BurpRequestResponse.Comments {
						if oldComment != newComment {
							if newComment.UserWhoCommented == c.name {
								c.serverHub.rooms[c.roomName].updateRequestResponseComments(message)
							} else {
								log.Printf("User " + c.name + " cannot add comment by " + newComment.UserWhoCommented)
							}
						}
					}
				}
			}
		} else {
			c.serverHub.rooms[c.roomName].updateRequestResponseComments(message)
		}
	case "GET_CONFIG_MESSAGE":
		message.MessageType = "GET_CONFIG_MESSAGE"
		if c.serverHub.shortenerService != nil {
			message.Data = c.serverHub.shortenerService.apiKey
		}
		c.serverHub.messages <- generateMessage(message, c, c.roomName, message.MessageTarget)
	case "COOKIE_MESSAGE":
		fallthrough
	case "SCAN_ISSUE_MESSAGE":
		fallthrough
	case "REPEATER_MESSAGE":
		fallthrough
	case "INTRUDER_MESSAGE":
		fallthrough
	case "BURP_MESSAGE":
		c.serverHub.messages <- generateMessage(message, c, c.roomName, message.MessageTarget)
	default:
		return errors.New("ERROR: unknown message type")
	}
	return nil
}

func (c *Client) Reader() {
	defer func() {
		c.serverHub.RemoveClient(c)
		_ = c.conn.Close()
	}()
	_ = c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error { _ = c.conn.SetReadDeadline(time.Now().Add(60 * time.Second)); return nil })
	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
			break
		}
		newBurpMessage := NewBurpTCMessage()
		decodedBytes := make([]byte, base64.StdEncoding.DecodedLen(len(message)))
		_, err = base64.StdEncoding.Decode(decodedBytes, message)
		if err != nil {
			log.Printf("error decoding base64: %v", err)
		}
		if err := json.Unmarshal(bytes.Trim(decodedBytes, "\x00"), &newBurpMessage); err != nil {
			log.Printf("Could not unmarshal BurpTCMessage, error: %s \n", err)
		} else {
			if err = c.parseMessage(newBurpMessage); err != nil {
				log.Printf("Could not parse BurpTCMessage, error: %s \n", err)

			}
		}
	}
}

func (c *Client) Writer() {
	ticker := time.NewTicker(50 * time.Second)
	defer func() {
		ticker.Stop()
		_ = c.conn.Close()
	}()
	for {
		select {
		case message, ok := <-c.sendChannel:
			//fmt.Printf("Message: %+v to send to client: %s\n",*message.msg,c.name)
			if c.conn != nil {
				_ = c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
				if !ok {
					// The hub closed the channel.
					_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
					return
				}

				w, err := c.conn.NextWriter(websocket.TextMessage)
				if err != nil {
					return
				}
				jsonBytes, err := json.Marshal(message.msg)
				encodedBuf := make([]byte, base64.StdEncoding.EncodedLen(len(jsonBytes)))
				base64.StdEncoding.Encode(encodedBuf, jsonBytes)
				//mw := io.MultiWriter(os.Stdout,w)
				//
				//if _,err := mw.Write(encodedBuf); err != nil {
				//	log.Printf("Error Writing message: %s\n",err)
				//}
				if _, err := w.Write(encodedBuf); err != nil {
					log.Printf("Error Writing message: %s\n", err)
				}
				if err := w.Close(); err != nil {
					return
				}
			}
		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
