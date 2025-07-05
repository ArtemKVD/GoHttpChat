FROM golang:1.24-alpine AS builder

RUN apk add --no-cache git gcc musl-dev

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=1 GOOS=linux go build -ldflags="-w -s" -o /app/httpchatgo ./server

FROM alpine:3.19

RUN apk add --no-cache curl
RUN apk add --no-cache postgresql-client

WORKDIR /app

COPY --from=builder /app/httpchatgo /app/httpchatgo
COPY --from=builder /app/views /app/views

EXPOSE 8080 8444
EXPOSE 2112

CMD ["/app/httpchatgo"]