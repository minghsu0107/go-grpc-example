# Go gRPC Tutorial
A repository that demonstrates Golang gRPC usages with hands-on examples. We will cover the following content:
- Greeting service 
  - Unary call, server streaming, client streaming, and bidirectional streaming 
  - Deadlines and SSL encryption
- Calculator service
  - Error handling
- Blog service
  - CRUD API with MongoDB
### Installation
You should have `protoc` binary installed:
```bash
brew install protobuf
protoc --version
```
Also, you should install Go packages for code generation:
```bash
go get -u github.com/golang/protobuf/protoc-gen-go
```
Finally, the gRPC package:
```bash
go get -u google.golang.org/grpc
```
Add the following line in `.zshrc`:
```bash=
export GO_PATH=~/go
export PATH=$PATH:/$GO_PATH/bin
```
### Evan
[github](https://github.com/ktr0731/evans)

Evan is a command line gRPC client. It is useful for development environment.

```bash
brew tap ktr0731/evans
brew install evans
```

Or use `go get`:
```bash
go get github.com/ktr0731/evans
```

Connect to server using gPRC reflection:
```bash
evans --host 127.0.0.1 -p 50051 -r
```