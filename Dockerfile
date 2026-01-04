# Build stage
FROM node:20-alpine AS builder

WORKDIR /app

# Copy package files
COPY package.json package-lock.json ./

# Install all dependencies (including devDependencies for build)
RUN apk add --no-cache git
RUN npm ci

# Copy source code and tsconfig
COPY src ./src
COPY tsconfig.json ./

# Build TypeScript
RUN npm run build

# Remove devDependencies after build
RUN npm prune --production

# Production stage
FROM node:20-alpine

WORKDIR /app

# Copy package files
COPY package.json package-lock.json ./

# Copy node_modules from builder (already pruned to production only)
COPY --from=builder /app/node_modules ./node_modules

# Copy built files from builder stage
COPY --from=builder /app/dist ./dist

# Set environment to production
ENV NODE_ENV=production

# Expose port if needed (uncomment and modify if your app uses a port)
# EXPOSE 3000

# Run the application
CMD ["node", "dist/index.js"]
