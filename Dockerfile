# Stage 1: Build Frontend
# Use a newer Node LTS for more up-to-date toolchain (22 is a current LTS)
FROM node:22 AS frontend_builder
WORKDIR /frontend
COPY frontend/package*.json ./
RUN npm install
COPY frontend/. .
RUN npm run build

# Stage 2: Build Backend
# Match the Go version declared in go.mod (1.25.4) to avoid subtle toolchain differences
FROM golang:1.25.4-alpine AS backend_builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# Produce a static linux binary to make the final image small and self-contained
ENV CGO_ENABLED=0 GOOS=linux GOARCH=amd64
RUN go build -ldflags "-s -w" -o main cmd/api/main.go

# Stage 3: Final Image
FROM alpine:3.20.1
WORKDIR /app
# Copy backend binary
COPY --from=backend_builder /app/main /app/main
# Copy frontend build artifacts
COPY --from=frontend_builder /frontend/dist /app/dist

# Install ca-certificates for external API calls (Spotify)
RUN apk --no-cache add ca-certificates

# Create a non-root user for better security. The binary is world-executable so this will run fine.
RUN addgroup -S app && adduser -S app -G app
USER app

EXPOSE 8080
CMD ["./main"]
