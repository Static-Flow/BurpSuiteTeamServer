package internal

type Room struct {
	scope    string
	name     string
	password string
	clients  map[string]*Client
}

func NewRoom(roomName string, password string) *Room {
	return &Room{
		"",
		roomName,
		password,
		make(map[string]*Client),
	}
}
