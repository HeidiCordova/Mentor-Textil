# Jetson Orin - ui-local
# Multi-stage: build con Node, serve con nginx ARM64

FROM node:18-alpine AS builder

WORKDIR /app

COPY package*.json ./
RUN npm ci --production=false

COPY . .
RUN npm run build

FROM nginx:1.25-alpine

RUN rm /etc/nginx/conf.d/default.conf

COPY --from=builder /app/dist /usr/share/nginx/html
COPY nginx.conf /etc/nginx/conf.d/default.conf.tmpl
COPY cgi-bin/env /opt/cgi-bin/env
COPY entrypoint.sh /opt/entrypoint.sh
RUN chmod +x /opt/cgi-bin/env /opt/entrypoint.sh && mkdir -p /env-data

HEALTHCHECK --interval=15s --timeout=3s --retries=5 \
    CMD wget -q --spider http://127.0.0.1:8080/ || exit 1

EXPOSE 8080

CMD ["/opt/entrypoint.sh"]
