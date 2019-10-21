package chatapi

import (
	"fmt"
	"time"
)

type Comment struct {
	comment          string
	userWhoCommented string
	timeOfComment    time.Time
}

type BurpMetaData struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Protocol string `json:"protocol"`
}

type BurpRequestResponse struct {
	Request     []int        `json:"request"`
	Response    []int        `json:"response"`
	HttpService BurpMetaData `json:"httpService"`
	Comment     []Comment    `json:"comments"`
}

type BurpTCMessage struct {
	BurpRequestResponse BurpRequestResponse `json:"burpmsg"`
	MessageTarget       string              `json:"messageTarget"`
	MessageType         string              `json:"msgtype"`
	Data                string              `json:"data"`
}

func NewBurpTCMessage() *BurpTCMessage {
	return &BurpTCMessage{}
}

func (b BurpTCMessage) String() string {
	return fmt.Sprintf("%s - %s - %b - %s",
		b.BurpRequestResponse, b.MessageTarget, b.MessageType, b.Data)
}
