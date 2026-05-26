Nginx + Certbot configuration for Headquarter reverse proxy.

- `default.conf`: HTTP config (serves ACME challenge and proxies to `app:8080`).
- `certbot-www/`: webroot used by Certbot to respond to ACME challenges.

When running with Docker Compose, mount `CERTBOT_DOMAINS` and `CERTBOT_EMAIL` as env vars (or place them in a `.env` file next to `docker-compose.yml`).

Example `.env`:

CERTBOT_DOMAINS=example.com,www.example.com
CERTBOT_EMAIL=you@example.com
