#!/usr/bin/env bash
set -euo pipefail

# Headquarter agent bootstrap
# Usage: sudo bash bootstrap-server.sh

print() { echo "[hq-bootstrap] $*"; }

if [ "$(id -u)" -ne 0 ]; then
  echo "Run as root: sudo $0" >&2
  exit 1
fi

SERVER_ADDR=${SERVER_ADDR:-}
if [ -z "$SERVER_ADDR" ]; then
  read -r -p "FQDN or IP of this server (public address): " SERVER_ADDR
fi

AGENT_PORT=${AGENT_PORT:-8085}
if [ -z "$AGENT_PORT" ]; then
  read -r -p "Port agent should listen on (default 8085): " AGENT_PORT
  AGENT_PORT=${AGENT_PORT:-8085}
fi

HQ_URL=${HQ_URL:-}
if [ -z "$HQ_URL" ]; then
  read -r -p "Full URL of Headquarter (e.g. https://hq.example.com): " HQ_URL
fi

DO_REGISTER=${DO_REGISTER:-}
if [ -z "$DO_REGISTER" ]; then
  read -r -p "Do you want to register this server instance in Headquarter now? (y/N): " DO_REGISTER
fi

AGENT_TOKEN=${AGENT_TOKEN:-}
if [ -z "$AGENT_TOKEN" ]; then
  read -r -p "Agent token to use for authentication (will be stored on agent only): " AGENT_TOKEN
fi

AGENT_BINARY_URL=${AGENT_BINARY_URL:-}
if [ -z "$AGENT_BINARY_URL" ]; then
  read -r -p "URL where the agent binary can be downloaded (http(s)://.../agent): " AGENT_BINARY_URL
fi

ADMIN_JWT=${ADMIN_JWT:-}
if [ -z "$ADMIN_JWT" ] && [[ "$DO_REGISTER" =~ ^[Yy] ]]; then
  read -r -p "Optional: admin JWT for Headquarter (leave empty to skip registration): " ADMIN_JWT
fi

print "Updating packages and installing prerequisites..."
if command -v apt-get >/dev/null 2>&1; then
  apt-get update
  apt-get install -y ca-certificates curl gnupg lsb-release git jq
  # install docker if missing
  if ! command -v docker >/dev/null 2>&1; then
    print "Installing Docker (apt)..."
    mkdir -p /etc/apt/keyrings
    curl -fsSL https://download.docker.com/linux/ubuntu/gpg | gpg --dearmor -o /etc/apt/keyrings/docker.gpg
    echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable" |
      tee /etc/apt/sources.list.d/docker.list > /dev/null
    apt-get update
    apt-get install -y docker-ce docker-ce-cli containerd.io docker-compose-plugin
  fi
elif command -v yum >/dev/null 2>&1; then
  yum install -y curl git jq
  if ! command -v docker >/dev/null 2>&1; then
    yum install -y docker
    systemctl enable --now docker
  fi
else
  print "Unsupported package manager; please install docker, git and curl manually."
fi

print "Creating service user 'hq-agent' and directory /opt/hq-agent..."
useradd -m -s /bin/bash -r -U hq-agent || true
mkdir -p /opt/hq-agent
chown hq-agent:hq-agent /opt/hq-agent
chmod 750 /opt/hq-agent

print "Downloading agent binary to /opt/hq-agent/agent ..."
curl -fsSL "$AGENT_BINARY_URL" -o /opt/hq-agent/agent
chmod 750 /opt/hq-agent/agent
chown hq-agent:hq-agent /opt/hq-agent/agent

print "Creating systemd unit for hq-agent..."
cat >/etc/systemd/system/hq-agent.service <<EOF
[Unit]
Description=Headquarter Agent
After=network.target docker.service

[Service]
User=hq-agent
Group=hq-agent
Environment=AGENT_PORT=${AGENT_PORT}
ExecStart=/opt/hq-agent/agent -listen ":${AGENT_PORT}" -token "${AGENT_TOKEN}"
Restart=on-failure
RestartSec=5s

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable --now hq-agent.service

print "Agent service started. Checking status..."
systemctl status --no-pager hq-agent.service || true

if [[ "$DO_REGISTER" =~ ^[Yy] ]]; then
  if [ -z "${ADMIN_JWT:-}" ]; then
    print "No admin JWT provided; cannot register instance automatically."
  else
    print "Registering instance in Headquarter..."
    INST_NAME="instance-$(hostname -s)-$(date +%s)"
    PAYLOAD=$(jq -n --arg name "$INST_NAME" --arg server "http://${SERVER_ADDR}:${AGENT_PORT}" \
      --arg base_repo_url "" --arg base_repo_ref "main" --arg compose_path "." \
      --argjson env '{}' '{name: $name, server: $server, base_repo_url: $base_repo_url, base_repo_ref: $base_repo_ref, compose_path: $compose_path, environment: {AGENT_TOKEN: "'""$AGENT_TOKEN"'""}}')

    RESP=$(curl -s -w "\n%{http_code}" -X POST "$HQ_URL/api/instances" -H "Content-Type: application/json" -H "Authorization: Bearer $ADMIN_JWT" -d "$PAYLOAD")
    HTTP=$(echo "$RESP" | tail -n1)
    BODY=$(echo "$RESP" | sed '$ d')
    if [ "$HTTP" -ge 200 ] && [ "$HTTP" -lt 300 ]; then
      print "Instance registered successfully: $BODY"
    else
      print "Failed to register instance, status=$HTTP, body=$BODY"
    fi
  fi
fi

print "Bootstrap complete. Agent is running and listening on port ${AGENT_PORT}."
print "If you registered the instance, check Headquarter UI or API for the new instance record."

exit 0
