ARG GO_VERSION=1.25.1

FROM golang:${GO_VERSION}-alpine AS builder

WORKDIR /app
COPY . .

RUN go build -o server cmd/server/main.go

FROM alpine:latest
WORKDIR /root/
COPY --from=builder /app/server .
COPY restart-loop.sh .
RUN chmod +x restart-loop.sh

EXPOSE 5001

# For auto-restart every 2 hours
CMD ["./restart-loop.sh"]

# For normal use without auto-restart
# CMD ["./server", "--addr", "0.0.0.0:5001"]
