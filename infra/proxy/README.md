Nginx + Certbot consolidated configuration for Headquarter reverse proxy.

- `default.conf`: HTTP config (serves ACME challenge and proxies to `app:8080`).
- `certbot-www/`: webroot used by Certbot to respond to ACME challenges.
 - `letsencrypt/`: persisted certs and certbot state (single folder).

When running with Docker Compose, set the env vars near `docker-compose.yml` or export them in your shell:

```
CERTBOT_DOMAINS=example.com,www.example.com
CERTBOT_EMAIL=you@example.com
```

The `certbot` container will create `ssl.conf` inside this folder when obtaining certificates and Nginx will be reloaded.
