### Build binary from official Go image
FROM golang:stretch as build
COPY . /src
WORKDIR /src
RUN go get github.com/Static-Flow/BurpSuiteTeamServer/chatapi
RUN go install github.com/Static-Flow/BurpSuiteTeamServer/cmd/BurpSuiteTeamServer

EXPOSE 8989

CMD /go/bin/BurpSuiteTeamServer