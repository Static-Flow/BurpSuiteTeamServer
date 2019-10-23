# BurpSuite-Team-Server

This repository holds the code for the server side of the Burpsuite Team Collaborator tool found here https://github.com/AonCyberLabs/BurpSuite-Team-Extension.

# Features

  + Multiple room support
  
  + Support for rooms with passwords
  
  + Mutual TLS encryption between server and client with server generated certificate and key
  
  + Seperate room scopes
  
  + More to come!
  
# How to start the Server

```
go get github.com/AonCyberLabs/BurpSuiteTeamServer/cmd/BurpSuiteTeamServer
cd ~/go/src/github.com/AonCyberLabs/BurpSuiteTeamServer/
go get ./...
go install ./...
~/go/bin/BurpSuiteTeamServer -h
```
Output:
```
Usage of BurpSuiteTeamServer:
  -host string
    	host for TLS cert. Defaults to localhost (default "localhost")
  -port string
    	http service address (default "9999")
  -serverPassword string
    	password for the server
