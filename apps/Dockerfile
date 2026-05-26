FROM node:20-alpine AS frontend-builder
WORKDIR /web
COPY apps/web/package*.json ./
RUN npm install
COPY apps/web/ .
RUN npm run build

FROM golang:1.25-alpine AS backend-builder
WORKDIR /app
COPY . .
WORKDIR /app/apps
COPY apps/go.mod apps/go.sum ./
RUN go mod download
COPY --from=frontend-builder /web/dist ./apps/web/dist
WORKDIR /app/apps
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/main ./cmd/jiramo

FROM alpine:latest
WORKDIR /root/
COPY --from=backend-builder /app/main .
COPY --from=backend-builder /app/infra/scripts ./infra/scripts

EXPOSE 8080
CMD ["./main"]
