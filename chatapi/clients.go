package chatapi

import (
	"bufio"
	"encoding/json"
	"io"
	"log"
	"reflect"
	"strings"
	//"strings"
)

//client type represents a chat client
type client struct {
	*bufio.Reader
	*bufio.Writer
	wc            chan string
	mutedClients  []string
	outputChannel chan<- BurpTCMessage
	Name          string
}

/*
	StartClient starts a chat client. This function uses the channel generator pattern
	first argument is the BurpSuiteTeamServer channel of chat room which the client belongs to. The client will be sending messages to this channel
	second argument is a readwritercloser representing a connection at which the client is communicating
	third argument is a quit channel. If a signal is passed through this channel, the client closes.
*/
func StartClient(clientName string, chatApi *ChatAPI, msgCh chan<- BurpTCMessage, cn io.ReadWriteCloser, roomName string) (*client, <-chan struct{}) {
	c := new(client)
	c.outputChannel = msgCh
	c.Name = clientName
	c.Reader = bufio.NewReader(cn)
	c.Writer = bufio.NewWriter(cn)
	c.wc = make(chan string)
	channelDone := make(chan struct{})

	go func() {
		scanner := bufio.NewScanner(c.Reader)
		buf := make([]byte, 0, 8192*8192)
		scanner.Buffer(buf, 8192*8192)
	Scanner:
		for scanner.Scan() {
			log.Printf("scan buffer: %s", scanner.Text())
			msg := NewBurpTCMessage()
			var decryptedMessage string
			if roomName == "server" {
				decryptedMessage = chatApi.serverRoom.crypter.Decrypt(scanner.Text())
			} else {
				decryptedMessage = chatApi.rooms[roomName].crypter.Decrypt(scanner.Text())
			}
			if err := json.Unmarshal([]byte(decryptedMessage), &msg); err != nil {
				log.Printf("Could not decode BurpTCMessage, error: %s \n", err)
			} else {
				switch msg.MessageType {
				case "SYNC_SCOPE_MESSAGE":
					if msg.MessageTarget != "me" {
						log.Printf("new scope from: %s", msg.SendingUser)
						chatApi.rooms[roomName].Scope = msg.Data
					} else {
						log.Printf("%s requesting scope", msg.SendingUser)
						msg.Data = chatApi.rooms[roomName].Scope
					}
					c.outputChannel <- *msg
				case "JOIN_ROOM_MESSAGE":
					log.Printf("%s joining room: %s", msg.SendingUser, msg.RoomName)
					chatApi.moveClientToRoom(c, roomName, msg.RoomName)
					roomName = msg.RoomName
				case "LEAVE_ROOM_MESSAGE":
					log.Printf("leaving room: %s", msg.RoomName)
					roomName = "server"
					chatApi.removeClientFromRoom(c, msg.RoomName)
				case "ADD_ROOM_MESSAGE":
					log.Printf("creating new room: %s", msg.RoomName)
					chatApi.moveClientToRoom(c, roomName, msg.RoomName)
					roomName = msg.RoomName
				case "QUIT_MESSAGE":
					log.Printf("client %s quitting", msg.SendingUser)
					break Scanner
				case "MUTE_MESSAGE":
					if msg.MessageTarget == "all" {
						keys := reflect.ValueOf(chatApi.rooms[roomName].clients).MapKeys()
						for i := 0; i < len(keys); i++ {
							key := keys[i].String()
							if key != msg.SendingUser {
								c.mutedClients = append(c.mutedClients, key)
							}
						}
						log.Printf("Clients to mute %s", keys)
					} else {
						log.Printf("Client to mute %s", msg.MessageTarget)
						c.mutedClients = append(c.mutedClients, msg.MessageTarget)
					}
					log.Printf("%s muted these clients %s", msg.SendingUser, c.mutedClients)
				case "UNMUTE_MESSAGE":
					if msg.MessageTarget == "all" {
						keys := reflect.ValueOf(chatApi.rooms[roomName].clients).MapKeys()
						for i := 0; i < len(keys); i++ {
							key := keys[i].String()
							if key != msg.SendingUser {
								c.mutedClients = remove(c.mutedClients, index(c.mutedClients, keys[i].String()))
							}
						}
					} else {
						c.mutedClients = remove(c.mutedClients, index(c.mutedClients, msg.MessageTarget))
					}
					log.Printf("%s unmuted %s", msg.SendingUser, msg.MessageTarget)
				case "REPEATER_MESSAGE":
					fallthrough
				case "INTRUDER_MESSAGE":
					fallthrough
				case "SYNC_ISSUE_MESSAGE":
					fallthrough
				case "BURP_MESSAGE":
					c.outputChannel <- *msg
				case "GET_ROOMS_MESSAGE":
					rooms := chatApi.GetRooms()
					keys := make([]string, 0, len(rooms))
					for k := range rooms {
						keys = append(keys, k)
					}
					msg.Data = strings.Join(keys, ",")
					c.outputChannel <- *msg
				default:
					log.Println("ERROR: unknown message type")
				}
			}
		}
		chatApi.removeClientFromRooms(c)
		close(channelDone)
		cn.Close()
		if err := scanner.Err(); err != nil {
			log.Printf("err: %s", err)
		}
	}()

	c.writeMonitor()
	return c, channelDone
}

func (c *client) changeChannel(newChannel chan<- BurpTCMessage) {
	c.outputChannel = newChannel
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

func (c *client) writeMonitor() {
	go func() {
		for s := range c.wc {
			c.WriteString(s)
			c.Flush()
		}
	}()
}
