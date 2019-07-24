FROM golang:stretch as build
COPY . /src
WORKDIR /src
RUN go get -d -v ./...
RUN go install -v ./...

EXPOSE 8989
CMD ["BurpSuiteTeamServer"]