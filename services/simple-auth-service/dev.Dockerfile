FROM golang:1.26-trixie

WORKDIR /app

RUN go install github.com/air-verse/air@latest

CMD ["air", "-c", "./services/simple-auth-service/.air.toml"]
