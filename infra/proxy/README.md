Nginx + Certbot consolidated configuration for Headquarter reverse proxy.

- `default.conf.template`: HTTP config template that routes `CLIENT_HOSTNAME` and `DASHBOARD_HOSTNAME` to `dashboard`, while forwarding `/api` to `backend`; `API_HOSTNAME` goes to `backend`.
- `ssl.conf`: TLS config written by Certbot when certificates are obtained.
- `certbot-www/`: webroot used by Certbot to respond to ACME challenges.
- `letsencrypt/`: persisted certs and certbot state (single folder).

The values come from `infra/env/${DEPLOY_ENV:-dev}.env`.
`dev.env` is the local default; `prod.env` is the production template.

When running with Docker Compose, set the env vars near `docker-compose.yml` or export them in your shell:

```
CERTBOT_DOMAINS=example.com,www.example.com
CERTBOT_EMAIL=you@example.com
```

The `certbot` container will create `ssl.conf` inside this folder when obtaining certificates.
