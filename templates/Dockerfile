FROM golang:1.11-alpine as stage1

WORKDIR /app

RUN apk add --no-cache git gcc libc-dev

COPY go.mod .
COPY go.sum .
RUN go mod download
# graphql patch1

COPY . .

# graphql patch2

RUN cd cmd && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -a -installsuffix cgo -ldflags="-w -s" -o /app/app

FROM scratch

LABEL siop.descricao="Serviço de exemplo"

COPY --from=stage1 /app/app /
# copy schema.graphql

CMD [ "/app" ]