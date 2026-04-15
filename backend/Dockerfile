FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY backend/go.mod backend/go.sum ./
RUN go mod download

COPY backend/ ./
RUN CGO_ENABLED=0 GOOS=linux go build -o /go-chat-server ./main.go

FROM alpine:3.20

WORKDIR /app
RUN adduser -D -g '' appuser

COPY --from=builder /go-chat-server /app/go-chat-server

EXPOSE 8080

USER appuser

ENTRYPOINT ["/app/go-chat-server"]
