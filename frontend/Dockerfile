# =============================================================================
# Stage 1: Install dependencies and build
# =============================================================================
FROM node:22-alpine AS builder

WORKDIR /build

# Install deps (layer cached until lockfile changes)
COPY package.json package-lock.json ./
RUN npm ci --loglevel=warn

# Copy source and build
COPY . .
RUN npm run build

# =============================================================================
# Stage 2: Production server
# =============================================================================
FROM node:22-alpine AS runner

RUN addgroup -S appgroup && adduser -S appuser -G appgroup

# wget is required by the HEALTHCHECK below.
RUN apk add --no-cache wget

WORKDIR /app

# Only what's needed at runtime
COPY --from=builder /build/.next/standalone ./
COPY --from=builder /build/.next/static ./.next/static
COPY --from=builder /build/public ./public

USER appuser

EXPOSE 3000

ENV NODE_ENV=production \
    HOSTNAME=0.0.0.0 \
    PORT=3000

HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:3000/ || exit 1

ENTRYPOINT ["node", "server.js"]
