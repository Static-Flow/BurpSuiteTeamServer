### Build binary from official Go image
FROM golang:stretch as build
COPY . /src
WORKDIR /src
RUN go get -d -v ./...
RUN go install -v ./...
#RUN go get github.com/Static-Flow/BurpSuiteTeamServer/chatapi
#RUN go get github.com/Static-Flow/BurpSuiteTeamServer/authentication
#RUN go get github.com/gorilla/mux
#RUN go install github.com/Static-Flow/BurpSuiteTeamServer/cmd/BurpSuiteTeamServer

EXPOSE 8989

#CMD /go/bin/BurpSuiteTeamServer
CMD ["BurpSuiteTeamServer"]