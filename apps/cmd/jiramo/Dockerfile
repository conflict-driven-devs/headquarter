FROM node:20-alpine AS frontend-builder
WORKDIR /src/web
COPY web/package*.json ./
RUN npm install
COPY web/ .
RUN npm run build

FROM golang:1.25-alpine AS backend-builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=frontend-builder /src/web/dist ./web/dist
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/jiramo ./cmd/jiramo

FROM alpine:latest
WORKDIR /root/
COPY --from=backend-builder /out/jiramo ./jiramo
COPY --from=backend-builder /src/infra/scripts ./infra/scripts
EXPOSE 8080
CMD ["./jiramo"]
