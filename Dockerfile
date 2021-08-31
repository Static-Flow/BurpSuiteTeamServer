FROM golang:1.16-alpine
WORKDIR /teamserver
COPY go.mod ./
COPY go.sum ./
RUN go mod download
COPY . .
RUN go build -o /burpteamserver ./cmd/BurpSuiteTeamServer
EXPOSE 443
CMD [ "/burpteamserver","-port=443", "-host=localhost", "-serverPassword=test123" ]