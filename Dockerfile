FROM registry.cn-hangzhou.aliyuncs.com/gladsheim/golang:alpine
MAINTAINER Odin Lee <odin@godigitalchina.com>

ADD ../../src /go/src
ADD ../../pkg /go/pkg
