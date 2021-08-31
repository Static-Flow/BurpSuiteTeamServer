package internal

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

func generateMessage(burpTCMessage *BurpTCMessage, sender *Client, roomName string, target string) Message {
	return Message{
		msg:      burpTCMessage,
		sender:   sender,
		roomName: roomName,
		target:   target,
	}
}
