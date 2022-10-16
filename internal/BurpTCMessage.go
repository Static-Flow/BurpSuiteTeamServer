package internal

import (
	"fmt"
	"strings"
	"time"
)

type JavaJsonTime struct {
	T time.Time
}

func (j *JavaJsonTime) UnmarshalJSON(b []byte) error {
	s := strings.Trim(string(b), "\"")
	t, err := time.Parse("Jan _2 15:04:05", s)
	if err != nil {
		return err
	}
	*j = JavaJsonTime{t}
	return nil
}

func (j JavaJsonTime) String() string {
	return j.T.Format("Jan _2 15:04:05")
}

func (j JavaJsonTime) MarshalJSON() ([]byte, error) {
	return []byte(`"` + j.T.Format("Jan _2 15:04:05") + `"`), nil
}

type Comment struct {
	Comment          string       `json:"comment"`
	UserWhoCommented string       `json:"userWhoCommented"`
	TimeOfComment    JavaJsonTime `json:"timeOfComment"`
}

type BurpMetaData struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Protocol string `json:"protocol"`
}

type BurpRequestResponse struct {
	Request     []int         `json:"request"`
	Response    []int         `json:"response"`
	HttpService *BurpMetaData `json:"httpService"`
	Comments    []Comment     `json:"comments"`
}

type BurpTCMessage struct {
	BurpRequestResponse *BurpRequestResponse `json:"burpmsg"`
	MessageType         string               `json:"msgtype"`
	Data                string               `json:"data"`
}

func NewBurpTCMessage() *BurpTCMessage {
	return &BurpTCMessage{}
}

func (b BurpTCMessage) String() string {
	return fmt.Sprintf("%+v - %s - %s",
		b.BurpRequestResponse, b.MessageType, b.Data)
}

func (b BurpRequestResponse) addComment(comment Comment) {
	b.Comments = append(b.Comments, comment)
}

func (b BurpRequestResponse) removeComments() {
	b.Comments = nil
}

func (b BurpRequestResponse) setComments(comments []Comment) {
	b.Comments = append([]Comment(nil), comments...)
}

func (b BurpRequestResponse) String() string {
	return fmt.Sprintf("%+q - %+q - %+v - %+v", b.Request, b.Response, b.HttpService, b.Comments)
}

func (b BurpMetaData) String() string {
	return fmt.Sprintf("%s - %d - %s", b.Host, b.Port, b.Protocol)
}

func (c *Comment) String() string {
	return fmt.Sprintf("%s - %s - %+v", c.Comment, c.UserWhoCommented, c.TimeOfComment)
}
