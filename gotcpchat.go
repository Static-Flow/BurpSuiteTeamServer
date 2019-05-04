package machat

import (
	"github.com/Static-Flow/BurpSuiteTeamServer/chatapi"
	"log"
	"net"
)

//RunTCPWithExistingAPI will start chat tcp server on the provided connection string using an existing chat api session
func RunTCPWithExistingAPI(connection string, chat *chatapi.ChatAPI) error {
	l, err := net.Listen("tcp", connection)
	if err != nil {
		log.Println("Error connecting to chat client", err)
		return err
	}
	defer l.Close()
	for {
		conn, err := l.Accept()
		if err != nil {
			break
		}
		go func(c net.Conn) {
			chat.AddClient(c)
		}(conn)
	}

	return err
}
