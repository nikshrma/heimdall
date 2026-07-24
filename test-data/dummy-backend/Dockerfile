FROM golang:1.26.2-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
  go build -ldflags="-s -w" \
  -o /dummy-backend \
  ./test-data/dummy-backend

FROM scratch

COPY --from=builder /dummy-backend /dummy-backend

EXPOSE 8080

ENTRYPOINT ["/dummy-backend"]
