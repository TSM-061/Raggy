FROM golang:1.26-alpine

WORKDIR /app

RUN go install github.com/air-verse/air@latest

CMD ["air", "-c", "./services/dashboard/.air.toml"]
