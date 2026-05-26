#!/bin/sh
set -eu

domains="${CERTBOT_DOMAINS:-}"
email="${CERTBOT_EMAIL:-}"

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
  server_name $domains;
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
