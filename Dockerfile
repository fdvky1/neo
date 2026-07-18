# Stage: build
FROM golang:1.25.5-alpine AS build

# Install build dependencies (CGO for go-sqlite3 + govips)
RUN apk add --no-cache gcc musl-dev vips-dev

WORKDIR /app

# Download dependencies first
COPY go.mod go.sum ./
RUN go mod download

# Copy source code and build
COPY . .
RUN CGO_ENABLED=1 GOOS=linux go build -trimpath -ldflags="-w -s" -o /app/neo .

# Stage: prod
FROM alpine:latest AS prod

# Install runtime dependencies (vips for image processing, ffmpeg for video)
RUN apk add --no-cache \
    ffmpeg \
    vips \
    ca-certificates \
    tzdata

WORKDIR /app

# Copy the binary from the build stage
COPY --from=build /app/neo .

# Command to run the application
CMD ["./neo"]
