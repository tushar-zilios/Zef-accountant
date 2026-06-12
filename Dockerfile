FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o accountant ./src/cmd/server

FROM alpine:3.21

WORKDIR /app

COPY --from=builder /app/accountant .

EXPOSE 8082

CMD ["./accountant"]
