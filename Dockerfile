# Stage: build
FROM golang:1.25.5-alpine AS build

# Install build dependencies (CGO often needed for go-sqlite3)
RUN apk add --no-cache gcc musl-dev

WORKDIR /app

# Download dependencies first
COPY go.mod go.sum ./
RUN go mod download

# Copy source code and build
COPY . .
RUN CGO_ENABLED=1 GOOS=linux go build -trimpath -ldflags="-w -s" -o /app/neo .

# Stage: prod
FROM alpine:latest AS prod

# Install required tools (cwebp/gif2webp via libwebp-tools, ffmpeg, imagemagick)
RUN apk add --no-cache \
    ffmpeg \
    libwebp-tools \
    imagemagick \
    ca-certificates \
    tzdata

WORKDIR /app

# Copy the binary from the build stage
COPY --from=build /app/neo .

# Command to run the application
CMD ["./neo"]
