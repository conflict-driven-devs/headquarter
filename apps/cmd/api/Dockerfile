FROM node:20-alpine AS frontend-builder
WORKDIR /src/web
COPY apps/web/package*.json ./
RUN npm install
COPY apps/web/ .
RUN npm run build

FROM golang:1.25-alpine AS backend-builder
WORKDIR /src/apps
COPY apps/go.mod apps/go.sum ./
RUN go mod download
COPY apps/cmd ./cmd
COPY apps/internal ./internal
COPY apps/web ./web
COPY infra/scripts ./infra/scripts
COPY --from=frontend-builder /src/web/dist ./web/dist
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/backend ./cmd/api

FROM alpine:latest
WORKDIR /root/
COPY --from=backend-builder /out/backend ./backend
COPY --from=backend-builder /src/apps/infra/scripts ./infra/scripts
EXPOSE 8080
CMD ["./backend"]