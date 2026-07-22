#!/bin/sh
# Resolver la IP del host (vision-event-detector corre en network_mode:host)
HOST_IP=$(getent hosts host.docker.internal | awk '{print $1}')
if [ -z "$HOST_IP" ]; then
    HOST_IP="172.17.0.1"
fi
export DETECTOR_HOST_IP="$HOST_IP"
envsubst '${DETECTOR_HOST_IP}' < /etc/nginx/conf.d/default.conf.tmpl > /etc/nginx/conf.d/default.conf

nc -lk -p 9090 -e /opt/cgi-bin/env &
exec nginx -g 'daemon off;'
