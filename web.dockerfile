ARG GO_VERSION=1.25.1

FROM golang:${GO_VERSION}-alpine AS builder

WORKDIR /app
COPY . .

RUN go mod download
RUN go build -o web cmd/web/main.go

FROM alpine:latest

WORKDIR /root/

COPY --from=builder /app/web .
COPY ./ui ./ui

EXPOSE 3000

CMD [ "./web", "--addr", "0.0.0.0:3000", "--cache-addr", "server:5001" ]
