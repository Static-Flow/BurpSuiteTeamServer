### Build binary from official Go image
FROM golang:stretch as build
COPY . /src
WORKDIR /src
RUN go get ./...
RUN go build -o /BurpSuiteTeamServer ./...

### Put the binary onto Heroku image
FROM alpine
WORKDIR /app
COPY --from=build /src/BurpSuiteTeamServer /app/
CMD ./BurpSuiteTeamServer