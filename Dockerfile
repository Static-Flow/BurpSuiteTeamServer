### Build binary from official Go image
FROM golang:stretch as build
COPY . /src
WORKDIR /src
RUN go get github.com/Static-Flow/BurpSuiteTeamServer/chatapi
RUN go install github.com/Static-Flow/BurpSuiteTeamServer/cmd/BurpSuiteTeamServer

### Put the binary onto Heroku image
FROM alpine
WORKDIR /app
COPY --from=build /go/bin/BurpSuiteTeamServer /app/
CMD ./BurpSuiteTeamServer