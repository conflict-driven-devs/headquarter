#!/bin/sh
set -eu

domains="${CERTBOT_DOMAINS:-}"
email="${CERTBOT_EMAIL:-}"
client_hostname="${CLIENT_HOSTNAME:-}"
dashboard_hostname="${DASHBOARD_HOSTNAME:-}"
api_hostname="${API_HOSTNAME:-}"

if [ -z "$domains" ] || [ -z "$email" ]; then
  exit 0
fi

primary_domain=$(printf '%s' "$domains" | cut -d',' -f1)
if [ ! -d "/etc/letsencrypt/live/$primary_domain" ]; then
  certbot certonly \
    --webroot -w /var/www/certbot \
    -d "$domains" \
    --email "$email" \
    --agree-tos --no-eff-email --non-interactive

  cat > /etc/nginx/conf.d/ssl.conf <<EOF
server {
  listen 443 ssl;
  server_name $client_hostname $dashboard_hostname;
  resolver 127.0.0.11 valid=10s ipv6=off;
  ssl_certificate /etc/letsencrypt/live/$primary_domain/fullchain.pem;
  ssl_certificate_key /etc/letsencrypt/live/$primary_domain/privkey.pem;
  location /api/ {
    set $backend_upstream http://backend:8080;
    proxy_pass $backend_upstream;
    proxy_set_header Host \$host;
    proxy_set_header X-Real-IP \$remote_addr;
    proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto \$scheme;
  }
  location / {
    set $dashboard_upstream http://dashboard:80;
    proxy_pass $dashboard_upstream;
    proxy_set_header Host \$host;
    proxy_set_header X-Real-IP \$remote_addr;
    proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto \$scheme;
  }
}

server {
  listen 443 ssl;
  server_name $api_hostname;
  resolver 127.0.0.11 valid=10s ipv6=off;
  ssl_certificate /etc/letsencrypt/live/$primary_domain/fullchain.pem;
  ssl_certificate_key /etc/letsencrypt/live/$primary_domain/privkey.pem;
  location / {
    set $backend_upstream http://backend:8080;
    proxy_pass $backend_upstream;
    proxy_set_header Host \$host;
    proxy_set_header X-Real-IP \$remote_addr;
    proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto \$scheme;
  }
}

server {
  listen 443 ssl;
  server_name $api_hostname;
  ssl_certificate /etc/letsencrypt/live/$primary_domain/fullchain.pem;
  ssl_certificate_key /etc/letsencrypt/live/$primary_domain/privkey.pem;
  location / {
    proxy_pass http://app:8080;
    proxy_set_header Host \$host;
    proxy_set_header X-Real-IP \$remote_addr;
    proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto \$scheme;
  }
}
EOF
fi
