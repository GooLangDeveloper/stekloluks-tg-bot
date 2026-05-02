FROM golang:1.21-alpine

WORKDIR /app

COPY go.mod ./
COPY main.go ./

RUN go mod tidy
RUN go build -ldflags="-s -w" -o bot

CMD ["./bot"]
