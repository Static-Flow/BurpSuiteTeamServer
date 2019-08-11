FROM golang:stretch as build
COPY . /src
WORKDIR /src
RUN go get -d -v ./...
# RUN go install -v ./...
RUN CGO_ENABLED=0 GOOS=linux go build -a -ldflags '-extldflags "-static"' -o /src/BurpSuiteTeamServer cmd/BurpSuiteTeamServer/BurpSuiteTeamServer.go

FROM scratch
COPY --from=build /src/BurpSuiteTeamServer /BurpSuiteTeamServer
EXPOSE 8989
ENTRYPOINT ["/BurpSuiteTeamServer"]