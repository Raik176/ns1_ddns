FROM golang:1.17-alpine as builder

WORKDIR /app
COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN go build -o ddns .

FROM alpine:latest

WORKDIR /root/
COPY --from=builder /app/ddns .

CMD ["./ddns"]