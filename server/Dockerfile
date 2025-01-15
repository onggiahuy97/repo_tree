FROM golang:1.21-alpine as builder

WORKDIR /app

COPY go.mod go.sum ./
COPY .env ./

RUN go mod download

COPY main.go ./

RUN CGO_ENABLED=0 GOOS=linux go build -o repo-tree

FROM alpine:latest

COPY --from=builder /app/repo-tree /repo-tree
COPY --from=builder /app/.env /.env

EXPOSE 8080

CMD ["./repo-tree"]
