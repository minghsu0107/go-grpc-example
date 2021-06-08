# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean

all: protoc server client

protoc:
	protoc proto/calculator.proto --go_out=plugins=grpc:.
server:
	$(GOBUILD) -o server -v calculator_server/server.go
client:
	$(GOBUILD) -o client -v calculator_client/client.go

server-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) -o server -v calculator_server/server.go
client-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) -o client -v calculator_client/client.go
clean:
	rm -f server client