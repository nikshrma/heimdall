FROM golang:1.26.2-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
  go build -ldflags="-s -w" \
  -o /heimdall \
  ./cmd/heimdall

FROM scratch

COPY --from=builder /heimdall /heimdall

EXPOSE 8080

ENTRYPOINT ["/heimdall"]
