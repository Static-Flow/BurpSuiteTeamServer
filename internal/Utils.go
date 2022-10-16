package internal

import (
	"crypto/rand"
	"math"
	"math/big"
)

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

func generateRandomUserNumber() (int, error) {
	maxLimit := int64(int(math.Pow10(5)) - 1)
	lowLimit := int(math.Pow10(5 - 1))

	randomNumber, err := rand.Int(rand.Reader, big.NewInt(maxLimit))
	if err != nil {
		return 0, err
	}
	randomNumberInt := int(randomNumber.Int64())

	// Handling integers between 0, 10^(n-1) .. for n=4, handling cases between (0, 999)
	if randomNumberInt <= lowLimit {
		randomNumberInt += lowLimit
	}

	// Never likely to occur, kust for safe side.
	if randomNumberInt > int(maxLimit) {
		randomNumberInt = int(maxLimit)
	}
	return randomNumberInt, nil
}

func generateMessage(burpTCMessage *BurpTCMessage, sender *Client, roomName string) *Message {
	return &Message{
		msg:      burpTCMessage,
		sender:   sender,
		roomName: roomName,
	}
}
