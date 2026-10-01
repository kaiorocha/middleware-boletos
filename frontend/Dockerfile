FROM node:22-alpine AS builder
WORKDIR /app
ARG NEXT_PUBLIC_API_URL=""
ARG NEXT_PUBLIC_APP_ENV="production"
ENV NEXT_PUBLIC_API_URL=$NEXT_PUBLIC_API_URL
ENV NEXT_PUBLIC_APP_ENV=$NEXT_PUBLIC_APP_ENV
COPY frontend/package.json frontend/package-lock.json* ./
RUN npm ci --quiet
COPY frontend/ .
COPY docs/openapi/middleware-boletos-etapa-5.openapi.json public/openapi.json
RUN mkdir -p public
ENV NODE_ENV=production
RUN npm run build

FROM node:22-alpine
# non-root user
RUN addgroup -S app && adduser -S -G app app
WORKDIR /app
ENV NODE_ENV=production
ENV HOSTNAME=0.0.0.0
COPY --from=builder /app/.next ./.next
COPY --from=builder /app/public ./public
COPY --from=builder /app/package.json ./package.json
COPY --from=builder /app/node_modules ./node_modules
RUN chown -R app:app /app
USER app
EXPOSE 3000
HEALTHCHECK --interval=30s --timeout=5s --start-period=30s --retries=3 CMD wget -qO- http://localhost:3000/web-health || exit 1
CMD ["npm", "start"]
