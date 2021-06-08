# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean

all: proc server client

proc:
	protoc proto/greet.proto --go_out=plugins=grpc:.
server:
	$(GOBUILD) -o server -v greet_server/server.go
client:
	$(GOBUILD) -o client -v greet_client/client.go

clean:
	rm -f server client