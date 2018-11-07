FROM golang:1.11-alpine

RUN apk add --no-cache git
RUN apk add bash
RUN apk add protobuf

RUN go get -u github.com/ksubedi/gomove
RUN go get github.com/kujtimiihoxha/kit
RUN go get golang.org/x/tools/cmd/goimports
RUN go get -u google.golang.org/grpc
RUN go get -u github.com/golang/protobuf/protoc-gen-go

WORKDIR /src
ADD adiciona_transport.sh .
ADD templates/graphql graphql/.
ADD templates/init_service.go .
