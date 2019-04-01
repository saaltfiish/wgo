FROM golang:latest
MAINTAINER Odin Lee <odin@godigitalchina.com>

ENV GO111MODULE=on

WORKDIR /wgo

COPY go.mod .
COPY go.sum .

RUN go mod download

ADD . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build
