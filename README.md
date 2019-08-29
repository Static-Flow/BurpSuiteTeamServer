# BurpSuite-Team-Server

This repository holds the code for the server side of the Burpsuite Team Collaborator tool found here https://github.com/Static-Flow/BurpSuite-Team-Extension. It is loosely built upon the chat server at ***MaChat by minaandrawos (https://github.com/minaandrawos/machat)***.

# Features

  + Multiple room support
  
  + AES encryption between server and client with server generated AES key
  
  + Seperate room scopes
  
  + More to come!
  
# How to start the Server

```
go get github.com/Static-Flow/BurpSuiteTeamServer/cmd/BurpSuiteTeamServer
cd ~/go/src/github.com/Static-Flow/BurpSuiteTeamServer/cmd/BurpSuiteTeamServer
go build BurpSuiteTeamServer.go
./BurpSuiteTeamServer
```
Output:
```
This is the server key that clients need to login: <Server key>
Starting chat room server
Awaiting Clients...
  
