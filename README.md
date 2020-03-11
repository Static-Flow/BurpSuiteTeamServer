# BurpSuite-Team-Server

This repository holds the code for the server side of the Burpsuite Team Collaborator tool found here https://github.com/Static-Flow/BurpSuite-Team-Extension.

# Features

  + Multiple room support
  
  + Support for rooms with passwords
  
  + Mutual TLS encryption between server and client with server generated certificate and key
  
  + Seperate room scopes
  
  + More to come!
  
# How to start the Server

```
go get github.com/Static-Flow/BurpSuiteTeamServer/cmd/BurpSuiteTeamServer
cd ~/go/src/github.com/Static-Flow/BurpSuiteTeamServer/
go get ./...
go install ./...
~/go/bin/BurpSuiteTeamServer -h
```
Output:
```
Usage of BurpSuiteTeamServer:
  -enableShortener
        Enables the built-in URL shortener
  -host string
        host for TLS cert. Defaults to localhost (default "localhost")
  -port string
        http service address (default "9999")
  -serverPassword string
        password for the server
```
