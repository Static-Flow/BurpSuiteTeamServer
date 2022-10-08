package tests

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"github.com/Static-Flow/BurpSuiteTeamServer/internal"
	"github.com/fasthttp/websocket"
	"io/ioutil"
	"math/rand"
	"net/http"
	"testing"
	"time"
)

func TestStartServerConnection(t *testing.T) {

	caCert, _ := ioutil.ReadFile("./burpServer.pem")
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)
	crt, _ := tls.LoadX509KeyPair("./burpServer.pem", "./burpServer.key")
	wsDialer := websocket.Dialer{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		Subprotocols:    []string{"p1", "p2"},
		TLSClientConfig: &tls.Config{
			Certificates: []tls.Certificate{crt},
			RootCAs:      caCertPool,
		},
	}
	var passwd, host, shortPort, srvPort string
	enableShort := false
	passwd = ""
	host = "localhost"
	shortPort = "8080"
	srvPort = "9999"
	//var hub *internal.ServerHub
	go func() {
		_ = internal.StartServer(&passwd, &host, &enableShort, &shortPort, &srvPort)
	}()
	ws, _, err := wsDialer.Dial(fmt.Sprintf("wss://%s:%s", host, srvPort), http.Header{"Username": {randSeq(10)}})
	if err != nil {
		t.Error(err)
	} else {
		ws.Close()
	}

}

var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func randSeq(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

type TestingClient struct {
	Server     string
	Port       string
	WsDialer   websocket.Dialer
	Connection *websocket.Conn
}

func SendRecv(ws *websocket.Conn) error {
	for i := 0; i < 100; i++ {
		if err := ws.SetWriteDeadline(time.Now().Add(time.Second * 10)); err != nil {
			return err
		}
		if err := ws.WriteMessage(websocket.PingMessage, nil); err != nil {
			return err
		}
		//if err := t.Connection.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
		//	return err
		//}
		//_, _, err := t.Connection.ReadMessage()
		//if err != nil {
		//	return err
		//}
	}
	return nil
}

func (t *TestingClient) Connect() error {
	var err error
	if t.Connection, _, err = t.WsDialer.Dial(fmt.Sprintf("wss://%s:%s", t.Server, t.Port), http.Header{"Username": {randSeq(10)}}); err != nil {
		return err
	}
	return nil
}
