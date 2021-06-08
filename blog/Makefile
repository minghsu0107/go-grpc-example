# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean

all: proc server client

proc:
	protoc proto/blog.proto --go_out=plugins=grpc:.
server:
	$(GOBUILD) -o server -v blog_server/server.go
client:
	$(GOBUILD) -o client -v blog_client/client.go

clean:
	rm -f server client