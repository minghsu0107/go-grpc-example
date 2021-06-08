FROM golang:1.15.2 AS builder

# Install grpc
RUN go get -u google.golang.org/grpc && \
    go get -u github.com/golang/protobuf/protoc-gen-go

# Install protoc and zip system library
RUN apt-get update && apt-get install -y zip && \
    mkdir /opt/protoc && cd /opt/protoc && wget https://github.com/protocolbuffers/protobuf/releases/download/v3.7.0/protoc-3.7.0-linux-x86_64.zip && \
    unzip protoc-3.7.0-linux-x86_64.zip

ENV PATH=$PATH:$GOPATH/bin:/opt/protoc/bin

RUN mkdir -p /app
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

ARG SVC_NAME

COPY . .
RUN make protoc $SVC_NAME-linux

FROM alpine:latest

# RUN apk update && apk add ca-certificates && rm -rf /var/cache/apk/*

RUN mkdir -p /app
WORKDIR /app
COPY --from=builder /app/$SVC_NAME ./
RUN apk add --no-cache bash

ENV SVC_NAME=$SVC_NAME
ENTRYPOINT /app/$SVC_NAME

