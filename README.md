# BurpSuite-Team-Server

This repository holds the code for the server side of the Burpsuite Team Collaborator tool found here https://github.com/GDSSecurity/BurpSuite-Team-Extension. It is loosely built upon the chat server at ***MaChat by minaandrawos (https://github.com/minaandrawos/machat)***.

# Features

  + Multiple room support
  
  + AES encryption between server and client with server generated AES key
  
  + Seperate room scopes
  
  + More to come!
  
# How to start the Server

```
git clone https://github.com/GDSSecurity/BurpSuiteTeamServer.git
cd BurpSuiteTeamServer/cmd/BurpSuiteTeamServer
go build
./BurpSuiteTeamServer
```
Output:
```
This is the server key that clients need to login: <Server key>
Starting chat room server
Awaiting Clients...
  
