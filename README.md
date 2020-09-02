你好！
很冒昧用这样的方式来和你沟通，如有打扰请忽略我的提交哈。我是光年实验室（gnlab.com）的HR，在招Golang开发工程师，我们是一个技术型团队，技术氛围非常好。全职和兼职都可以，不过最好是全职，工作地点杭州。
我们公司是做流量增长的，Golang负责开发SAAS平台的应用，我们做的很多应用是全新的，工作非常有挑战也很有意思，是国内很多大厂的顾问。
如果有兴趣的话加我微信：13515810775  ，也可以访问 https://gnlab.com/，联系客服转发给HR。
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
