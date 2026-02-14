# Stage 1
FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o beam main.go

# Stage 2
FROM alpine:latest

WORKDIR /app

RUN apk --no-cache add ca-certificates

COPY --from=builder /app/beam .
COPY --from=builder /app/public ./public

RUN mkdir -p data uploads

EXPOSE 3000

VOLUME ["/app/data", "/app/uploads"]

CMD ["./beam"]