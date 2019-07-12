package chatapi

import (
	"bufio"
	"encoding/json"
	"io"
	"log"
	//"strings"
)

//client type represents a chat client
type client struct {
	*bufio.Reader
	*bufio.Writer
	wc           chan string
	mutedClients []string
}

/*
	StartClient starts a chat client. This function uses the channel generator pattern
	first argument is the BurpSuiteTeamServer channel of chat room which the client belongs to. The client will be sending messages to this channel
	second argument is a readwritercloser representing a connection at which the client is communicating
	third argument is a quit channel. If a signal is passed through this channel, the client closes.
*/
func StartClient(serverPassword string, msgCh chan<- BurpTCMessage, cn io.ReadWriteCloser, roomName string) (*client, <-chan struct{}) {
	c := new(client)
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
			msg := NewBurpTCMessage()
			if err := json.Unmarshal([]byte(scanner.Text()), &msg); err != nil {
				log.Printf("Could not decode BurpTCMessage, error: %s \n", err)
			} else if msg.AuthenticationString == serverPassword {
				switch msg.MessageType {
				case "QUIT_MESSAGE":
					log.Printf("client %s quitting", msg.SendingUser)
					break Scanner
				case "MUTE_MESSAGE":
					c.mutedClients = append(c.mutedClients, msg.MessageTarget)
					log.Printf("%s muted %s", msg.SendingUser, msg.MessageTarget)
				case "UNMUTE_MESSAGE":
					c.mutedClients = remove(c.mutedClients, index(c.mutedClients, msg.MessageTarget))
					log.Printf("%s unmuted %s", msg.SendingUser, msg.MessageTarget)
				case "REPEATER_MESSAGE":
					fallthrough
				case "INTRUDER_MESSAGE":
					fallthrough
				case "BURP_MESSAGE":
					msgCh <- *msg
				default:
					log.Println("ERROR: unknown message type")
				}
			}
		}
		close(channelDone)
		cn.Close()
		if err := scanner.Err(); err != nil {
			log.Printf("err: %s", err)
		}
	}()

	c.writeMonitor()
	return c, channelDone
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
