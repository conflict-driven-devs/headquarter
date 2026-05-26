#!/usr/bin/env bash
set -euo pipefail

# Self-contained bootstrap script.
# Intended usage:
#   wget -qO- https://hq.example.com/scripts/setup.sh | BOOTSTRAP_TOKEN=... bash
# or interactive execution with prompts.

log() { echo "[hq-setup] $*"; }
die() { echo "[hq-setup] ERROR: $*" >&2; exit 1; }

if [ "$(id -u)" -eq 0 ]; then
  SUDO=""
else
  if ! command -v sudo >/dev/null 2>&1; then
    die "sudo is required when not running as root"
  fi
  SUDO="sudo"
fi

BOOTSTRAP_TOKEN=${BOOTSTRAP_TOKEN:-}
if [ -z "$BOOTSTRAP_TOKEN" ]; then
  read -r -p "Bootstrap token (from Headquarter): " BOOTSTRAP_TOKEN
fi

HQ_URL=${HQ_URL:-}
if [ -z "$HQ_URL" ]; then
  read -r -p "Headquarter URL (e.g. https://hq.example.com): " HQ_URL
fi

SERVER_ADDR=${SERVER_ADDR:-}
if [ -z "$SERVER_ADDR" ]; then
  SERVER_ADDR=$(hostname -I 2>/dev/null | awk '{print $1}' || true)
fi
if [ -z "${SERVER_ADDR:-}" ]; then
  SERVER_ADDR=$(hostname -f 2>/dev/null || hostname)
fi

AGENT_PORT=${AGENT_PORT:-8085}
AGENT_LISTEN=${AGENT_LISTEN:-":${AGENT_PORT}"}
AGENT_DIR=${AGENT_DIR:-/opt/hq-agent}
AGENT_USER=${AGENT_USER:-hq-agent}

log "Installing prerequisites..."
if command -v apt-get >/dev/null 2>&1; then
  $SUDO apt-get update
  $SUDO apt-get install -y ca-certificates curl git jq openssl
  if ! command -v docker >/dev/null 2>&1; then
    log "Docker not found; installing docker.io"
    $SUDO apt-get install -y docker.io
    $SUDO systemctl enable --now docker
  fi
elif command -v dnf >/dev/null 2>&1; then
  $SUDO dnf install -y ca-certificates curl git jq openssl docker
  $SUDO systemctl enable --now docker
elif command -v yum >/dev/null 2>&1; then
  $SUDO yum install -y ca-certificates curl git jq openssl docker
  $SUDO systemctl enable --now docker
else
  die "Unsupported package manager"
fi

log "Requesting bootstrap configuration from Headquarter..."
HOSTNAME=$(hostname -s)
PAYLOAD=$(jq -n --arg hostname "$HOSTNAME" --arg server_addr "$SERVER_ADDR" '{hostname: $hostname, server_addr: $server_addr}')
RESP=$(curl -fsSL -X POST "$HQ_URL/api/bootstrap" \
  -H "Authorization: Bearer $BOOTSTRAP_TOKEN" \
  -H "Content-Type: application/json" \
  -d "$PAYLOAD")

AGENT_TOKEN=$(printf '%s' "$RESP" | jq -r '.agent_token // empty')
AGENT_BINARY_URL=$(printf '%s' "$RESP" | jq -r '.agent_binary_url // empty')
INSTANCE_NAME=$(printf '%s' "$RESP" | jq -r '.instance_name // empty')
BASE_REPO_URL=$(printf '%s' "$RESP" | jq -r '.base_repo_url // empty')
BASE_REPO_REF=$(printf '%s' "$RESP" | jq -r '.base_repo_ref // "main"')
COMPOSE_PATH=$(printf '%s' "$RESP" | jq -r '.compose_path // "."')

[ -n "$AGENT_TOKEN" ] || die "Headquarter response did not contain agent token"
[ -n "$AGENT_BINARY_URL" ] || die "Headquarter response did not contain agent binary URL"

log "Creating service user and directories..."
if ! id -u "$AGENT_USER" >/dev/null 2>&1; then
  $SUDO useradd --system --create-home --shell /usr/sbin/nologin "$AGENT_USER"
fi

$SUDO mkdir -p "$AGENT_DIR" /etc/hq-agent
$SUDO chown -R "$AGENT_USER:$AGENT_USER" "$AGENT_DIR"
$SUDO chmod 750 "$AGENT_DIR"

log "Downloading agent binary..."
AGENT_BIN="$AGENT_DIR/agent"
curl -fsSL "$AGENT_BINARY_URL" -o "$AGENT_BIN"
chmod 750 "$AGENT_BIN"
$SUDO chown "$AGENT_USER:$AGENT_USER" "$AGENT_BIN"

log "Writing bootstrap-derived config..."
TMP_ENV=$(mktemp)
{
  printf 'INSTANCE_NAME=%q\n' "$INSTANCE_NAME"
  printf 'BASE_REPO_URL=%q\n' "$BASE_REPO_URL"
  printf 'BASE_REPO_REF=%q\n' "$BASE_REPO_REF"
  printf 'COMPOSE_PATH=%q\n' "$COMPOSE_PATH"
  printf 'SERVER_ADDR=%q\n' "$SERVER_ADDR"
  printf '%s' "$RESP" | jq -r '.environment // {} | to_entries[] | @tsv' 2>/dev/null | while IFS=$'\t' read -r k v; do
    printf '%s=%q\n' "$k" "$v"
  done
  printf 'AGENT_TOKEN=%q\n' "$AGENT_TOKEN"
} > "$TMP_ENV"
$SUDO install -m 600 -o root -g root "$TMP_ENV" /etc/hq-agent/config.env
rm -f "$TMP_ENV"

log "Creating systemd unit..."
$SUDO tee /etc/systemd/system/hq-agent.service >/dev/null <<EOF
[Unit]
Description=Headquarter Agent
After=network-online.target docker.service
Wants=network-online.target docker.service

[Service]
Type=simple
User=${AGENT_USER}
Group=${AGENT_USER}
EnvironmentFile=/etc/hq-agent/config.env
ExecStart=${AGENT_BIN} -listen "${AGENT_LISTEN}" -token "${AGENT_TOKEN}"
Restart=always
RestartSec=5
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=${AGENT_DIR} /etc/hq-agent

[Install]
WantedBy=multi-user.target
EOF

log "Enabling and starting service..."
$SUDO systemctl daemon-reload
$SUDO systemctl enable --now hq-agent.service

log "Bootstrap complete."
log "Instance: ${INSTANCE_NAME:-unknown}"
log "Agent listen: ${AGENT_LISTEN}"
log "Check: systemctl status hq-agent.service"
