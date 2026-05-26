FROM golang:1.25-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/agent ./cmd/agent

FROM alpine:latest
WORKDIR /root/
COPY --from=builder /out/agent ./agent
EXPOSE 8081
CMD ["./agent"]
