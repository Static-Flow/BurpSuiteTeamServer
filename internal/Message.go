package internal

type Message struct {
	msg      *BurpTCMessage
	sender   *Client
	roomName string
}
