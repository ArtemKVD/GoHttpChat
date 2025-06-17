FROM golang:1.24-alpine AS builder

RUN apk add --no-cache git gcc musl-dev

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=1 GOOS=linux go build -ldflags="-w -s" -o /app/httpchatgo ./server

FROM alpine:3.18

RUN apk add --no-cache postgresql-client

COPY --from=builder /app/httpchatgo /app/httpchatgo
COPY --from=builder /app/views /app/views

COPY postgrescheck.sh /app/postgrescheck.sh
RUN chmod +x /app/postgrescheck.sh

WORKDIR /app

EXPOSE 8080 8443

CMD ["/app/httpchatgo"]

RUN mkdir /var/log/app
VOLUME /var/log/app