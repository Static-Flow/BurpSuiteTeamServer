package internal

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"github.com/fasthttp/websocket"
	"log"
	"time"
)

type Client struct {
	conn         *websocket.Conn
	room         string
	sendChannel  chan *Message
	mutedClients []string
	name         string
}

func (c *Client) isGivenClientMuted(clientName string) bool {
	for _, client := range c.mutedClients {
		if client == clientName {
			return true
		}
	}
	return false
}

func (c *Client) Reader() {
	defer func() {
		hub.unregister <- c
		_ = c.conn.Close()
	}()
	if err := c.conn.SetReadDeadline(time.Now().Add(60 * time.Second)); err != nil {
		log.Println("connection error:", err)
	}
	c.conn.SetPongHandler(func(string) error { _ = c.conn.SetReadDeadline(time.Now().Add(60 * time.Second)); return nil })
	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				log.Printf("read error: %v from client: %s", err, c.name)
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
			hub.messages <- &Message{
				msg:      newBurpMessage,
				sender:   c,
				roomName: c.room,
			}
		}
	}
}

func (c *Client) Writer() {
	ticker := time.NewTicker(50 * time.Second)
	defer func() {
		ticker.Stop()
		if c.conn != nil {
			_ = c.conn.Close()
		}
	}()
	for {
		select {
		case message, ok := <-c.sendChannel:
			if c.conn != nil {
				if !ok {
					log.Printf("Client %s is Not okay", c.name)
					//if err := c.conn.WriteMessage(websocket.CloseMessage, []byte{}); err != nil {
					//	log.Println("Error sending close message: ", err)
					//}
					return
				}

				_ = c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))

				if jsonBytes, err := json.Marshal(message.msg); err == nil {

					encodedBuf := make([]byte, base64.StdEncoding.EncodedLen(len(jsonBytes)))
					base64.StdEncoding.Encode(encodedBuf, jsonBytes)
					if err := c.conn.WriteMessage(websocket.TextMessage, encodedBuf); err != nil {
						log.Printf("Error Writing message: %s", err)
						return
					}

				} else {
					log.Println("Error marshalling json message: ", err)
				}
			} else {
				return
			}
		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
