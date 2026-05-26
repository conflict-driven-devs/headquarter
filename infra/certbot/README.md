Certbot container configuration

- Certs and state are persisted under `infra/letsencrypt` and `infra/letsencrypt-var`.
- ACME webroot is `infra/nginx/certbot-www` and is mounted into both `proxy` and `certbot` containers.
- Provide `CERTBOT_DOMAINS` (comma-separated) and `CERTBOT_EMAIL` env vars to obtain certificates.
