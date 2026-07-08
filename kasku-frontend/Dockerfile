FROM node:22-alpine AS builder

WORKDIR /app

COPY package.json package-lock.json ./
RUN npm ci --ignore-scripts

COPY . .

# PUBLIC_* vars are baked at build time via $env/static/public
ARG PUBLIC_API_BASE_URL=https://api.kasku.id/v1
ENV PUBLIC_API_BASE_URL=${PUBLIC_API_BASE_URL}
ARG PUBLIC_GOOGLE_CLIENT_ID=
ENV PUBLIC_GOOGLE_CLIENT_ID=${PUBLIC_GOOGLE_CLIENT_ID}

RUN npm run build

# ── Runtime ──────────────────────────────────────────────────
FROM node:22-alpine

WORKDIR /app

RUN addgroup -S kasku && adduser -S -G kasku kasku

COPY --from=builder --chown=kasku:kasku /app/build ./build
COPY --from=builder --chown=kasku:kasku /app/package.json ./

USER kasku

ENV NODE_ENV=production
ENV PORT=3000
ENV HOST=0.0.0.0

EXPOSE 3000

CMD ["node", "build"]
