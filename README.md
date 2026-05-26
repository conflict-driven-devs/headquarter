# Headquarter

Minimal control plane for provisioning and managing servers.

## Layout

- `apps` - application code, API, frontend embed, agent, models.
- `infra` - Docker, compose, bootstrap script, dev tooling.

## Start

```bash
docker compose up --build
```

Open `http://localhost` or `https://localhost` if certs are configured.

## Bootstrap a server

1. Create an instance.
2. Generate a bootstrap token for that instance.
3. Run the generated command on the server.

Example:

```bash
wget -qO- "https://hq.example.com/scripts/setup.sh" | BOOTSTRAP_TOKEN=... bash
```

## Notes

- Bootstrap tokens are one-time and hashed in DB.
- `AGENT_TOKEN` is encrypted in DB when `HQ_SECRET_KEY` is set.
- The app builds from `apps/cmd/jiramo` and serves the embedded frontend in production.