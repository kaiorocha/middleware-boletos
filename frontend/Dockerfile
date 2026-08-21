FROM node:18-alpine AS builder
WORKDIR /app
COPY package.json package-lock.json* ./
RUN npm ci --quiet
COPY . .
RUN mkdir -p public
ENV NODE_ENV=production
RUN npm run build

FROM node:18-alpine
# non-root user
RUN addgroup -S app && adduser -S -G app app
WORKDIR /app
COPY --from=builder /app/.next ./.next
COPY --from=builder /app/public ./public
COPY --from=builder /app/package.json ./package.json
COPY --from=builder /app/node_modules ./node_modules
RUN chown -R app:app /app
USER app
EXPOSE 3000
CMD ["npm", "start"]
