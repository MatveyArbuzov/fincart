FROM golang:1.26 AS builder

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build \
    -o /app/fincart-api \
    ./cmd/api

RUN CGO_ENABLED=0 GOOS=linux go build \
    -o /app/fincart-migrate \
    ./cmd/migrate


FROM alpine:3.22 AS api

WORKDIR /app

COPY --from=builder /app/fincart-api .

EXPOSE 8080

CMD ["./fincart-api"]


FROM postgres:17 AS migrate

WORKDIR /app

COPY --from=builder /app/fincart-migrate .

COPY migrations ./migrations

CMD ["./fincart-migrate"]