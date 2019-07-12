package chatapi

import (
	"fmt"
)

type BurpMetaData struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Protocol string `json:"protocol"`
}

type BurpRequestResponse struct {
	Request     []int        `json:"request"`
	Response    []int        `json:"response"`
	HttpService BurpMetaData `json:"httpService"`
}

type BurpTCMessage struct {
	BurpRequestResponse  BurpRequestResponse `json:"burpmsg"`
	AuthenticationString string              `json:"auth"`
	SendingUser          string              `json:"sender"`
	RoomName             string              `json:"room"`
	MessageTarget        string              `json:"receiver"`
	MessageType          string              `json:"msgtype"`
	Data                 string              `json:"data"`
}

func NewBurpTCMessage() *BurpTCMessage {
	return &BurpTCMessage{}
}

func (b BurpTCMessage) String() string {
	return fmt.Sprintf("%s - %s %s - %s - %s - %b - %s",
		b.BurpRequestResponse, b.AuthenticationString, b.SendingUser, b.RoomName, b.MessageTarget, b.MessageType, b.Data)
}
