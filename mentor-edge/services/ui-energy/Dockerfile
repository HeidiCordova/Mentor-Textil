# ui-energy — RPi Energy Monitor UI
# Multi-stage: build con Node, serve con nginx ARM64

FROM node:18-alpine AS builder

WORKDIR /app

COPY package*.json ./
RUN npm ci --production=false

COPY . .
RUN npm run build

FROM nginx:1.25-alpine

COPY --from=builder /app/dist /usr/share/nginx/html
COPY nginx.conf /etc/nginx/conf.d/default.conf

HEALTHCHECK --interval=15s --timeout=3s --retries=5 \
    CMD wget -q --spider http://127.0.0.1:8087/ || exit 1

EXPOSE 8087

CMD ["nginx", "-g", "daemon off;"]
