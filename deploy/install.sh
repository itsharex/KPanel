#!/bin/sh
set -eu

umask 027

SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
PROJECT_DIR=$(CDPATH= cd -- "$SCRIPT_DIR/.." && pwd)

AGENT_BINARY=
AGENT_SHA256=
IMAGE=
PUBLIC_URL=
PORT=18443
DRY_RUN=false

usage() {
	cat <<'EOF'
Usage:
  sudo ./deploy/install.sh \
    --agent-binary ./dist/linux-amd64/kejilion-agent \
    --agent-sha256 <64-character-sha256> \
    --image docker.io/OWNER/kejilion-panel@sha256:<64-character-digest> \
    --public-url https://panel.example.com \
    [--port 18443] [--dry-run]

The installer only creates Kejilion Panel files and services. It does not edit
kejilion.sh, /home/web, Nginx configuration, firewall rules, or existing sites.
EOF
}

fail() {
	printf 'install: %s\n' "$*" >&2
	exit 1
}

while [ "$#" -gt 0 ]; do
	case "$1" in
		--agent-binary)
			[ "$#" -ge 2 ] || fail "--agent-binary requires a value"
			AGENT_BINARY=$2
			shift 2
			;;
		--image)
			[ "$#" -ge 2 ] || fail "--image requires a value"
			IMAGE=$2
			shift 2
			;;
		--agent-sha256)
			[ "$#" -ge 2 ] || fail "--agent-sha256 requires a value"
			AGENT_SHA256=$2
			shift 2
			;;
		--public-url)
			[ "$#" -ge 2 ] || fail "--public-url requires a value"
			PUBLIC_URL=$2
			shift 2
			;;
		--port)
			[ "$#" -ge 2 ] || fail "--port requires a value"
			PORT=$2
			shift 2
			;;
		--dry-run)
			DRY_RUN=true
			shift
			;;
		-h|--help)
			usage
			exit 0
			;;
		*)
			fail "unknown argument: $1"
			;;
	esac
done

[ "$(id -u)" -eq 0 ] || fail "run as root"
[ -n "$AGENT_BINARY" ] || fail "--agent-binary is required"
[ -f "$AGENT_BINARY" ] || fail "agent binary not found: $AGENT_BINARY"
[ -x "$AGENT_BINARY" ] || fail "agent binary is not executable: $AGENT_BINARY"
[ -n "$AGENT_SHA256" ] || fail "--agent-sha256 is required"
printf '%s' "$AGENT_SHA256" | grep -Eq '^[A-Fa-f0-9]{64}$' || fail "invalid agent SHA-256"
[ -n "$IMAGE" ] || fail "--image is required"
printf '%s' "$IMAGE" | grep -Eq '^[A-Za-z0-9._/@:+-]+$' || fail "invalid image reference"
printf '%s' "$IMAGE" | grep -Eq '@sha256:[a-f0-9]{64}$' || fail "image must be pinned by sha256 digest"
printf '%s' "$PUBLIC_URL" | grep -Eq '^https://([A-Za-z0-9]([A-Za-z0-9.-]*[A-Za-z0-9])?)(:[0-9]{1,5})?$' ||
	fail "--public-url must be an https origin without path, userinfo, query, or fragment"
PUBLIC_PORT=$(printf '%s' "$PUBLIC_URL" | sed -n 's#^https://[^:]*:\([0-9][0-9]*\)$#\1#p')
if [ -n "$PUBLIC_PORT" ]; then
	[ "$PUBLIC_PORT" -ge 1 ] && [ "$PUBLIC_PORT" -le 65535 ] || fail "public URL port is invalid"
fi
case "$PORT" in
	''|*[!0-9]*)
		fail "--port must be numeric"
		;;
esac
[ "$PORT" -ge 1024 ] && [ "$PORT" -le 65535 ] || fail "--port must be between 1024 and 65535"

for command_name in docker systemctl getent groupadd install openssl mktemp sha256sum; do
	command -v "$command_name" >/dev/null 2>&1 || fail "required command not found: $command_name"
done
docker compose version >/dev/null 2>&1 || fail "Docker Compose v2 is required"
docker info >/dev/null 2>&1 || fail "Docker daemon is unavailable"
printf '%s  %s\n' "$AGENT_SHA256" "$AGENT_BINARY" | sha256sum --check --status ||
	fail "agent binary SHA-256 mismatch"

if [ "$DRY_RUN" = true ]; then
	printf 'Preflight passed.\n'
	printf 'Agent: %s\nImage: %s\nPublic URL: %s\nLocal port: %s\n' \
		"$AGENT_BINARY" "$IMAGE" "$PUBLIC_URL" "$PORT"
	exit 0
fi

LOCK_DIR=/run/lock/kejilion-panel-install
mkdir "$LOCK_DIR" 2>/dev/null || fail "another installation is running"
TEMP_TOKEN=
TEMP_ENV=
cleanup() {
	[ -z "$TEMP_TOKEN" ] || rm -f -- "$TEMP_TOKEN"
	[ -z "$TEMP_ENV" ] || rm -f -- "$TEMP_ENV"
	rmdir "$LOCK_DIR" 2>/dev/null || true
}
trap cleanup EXIT HUP INT TERM

PANEL_GROUP=kejilion-panel
if ! getent group "$PANEL_GROUP" >/dev/null 2>&1; then
	groupadd --system "$PANEL_GROUP"
fi
PANEL_GID=$(getent group "$PANEL_GROUP" | awk -F: '{print $3}')
[ -n "$PANEL_GID" ] || fail "cannot resolve $PANEL_GROUP gid"

ETC_DIR=/etc/kejilion-panel
OPT_DIR=/opt/kejilion-panel
DATA_DIR=/var/lib/kejilion-panel/panel
AGENT_TARGET=/usr/local/libexec/kejilion-agent
SERVICE_TARGET=/etc/systemd/system/kejilion-agent.service
COMPOSE_TARGET=$OPT_DIR/compose.yml
ENV_TARGET=$OPT_DIR/.env
TOKEN_TARGET=$ETC_DIR/agent.token
BACKUP_DIR=/var/backups/kejilion-panel/$(date -u +%Y%m%dT%H%M%SZ)

install -d -o root -g "$PANEL_GROUP" -m 0750 "$ETC_DIR"
install -d -o root -g root -m 0755 "$OPT_DIR"
install -d -o 65532 -g 65532 -m 0700 "$DATA_DIR"
install -d -o root -g root -m 0755 "$(dirname "$AGENT_TARGET")"
install -d -o root -g root -m 0700 "$BACKUP_DIR"

for current_path in "$AGENT_TARGET" "$SERVICE_TARGET" "$COMPOSE_TARGET" "$ENV_TARGET"; do
	if [ -e "$current_path" ]; then
		cp -a -- "$current_path" "$BACKUP_DIR/"
	fi
done

if [ ! -s "$TOKEN_TARGET" ]; then
	TEMP_TOKEN=$(mktemp "$ETC_DIR/.agent.token.XXXXXX")
	openssl rand -hex 32 >"$TEMP_TOKEN"
	install -o root -g "$PANEL_GROUP" -m 0640 "$TEMP_TOKEN" "$TOKEN_TARGET"
	rm -f -- "$TEMP_TOKEN"
	TEMP_TOKEN=
else
	chown root:"$PANEL_GROUP" "$TOKEN_TARGET"
	chmod 0640 "$TOKEN_TARGET"
fi

install -o root -g root -m 0755 "$AGENT_BINARY" "$AGENT_TARGET"
install -o root -g root -m 0644 \
	"$PROJECT_DIR/deploy/systemd/kejilion-agent.service" "$SERVICE_TARGET"
install -o root -g root -m 0644 \
	"$PROJECT_DIR/deploy/compose/compose.yml" "$COMPOSE_TARGET"

TEMP_ENV=$(mktemp "$OPT_DIR/.env.XXXXXX")
{
	printf 'KEJILION_PANEL_IMAGE=%s\n' "$IMAGE"
	printf 'KEJILION_PANEL_GID=%s\n' "$PANEL_GID"
	printf 'KEJILION_PANEL_DATA_DIR_HOST=%s\n' "$DATA_DIR"
	printf 'KEJILION_PANEL_PORT=%s\n' "$PORT"
	printf 'KEJILION_PANEL_PUBLIC_URL=%s\n' "$PUBLIC_URL"
	printf 'KEJILION_PANEL_SECURE_COOKIE=true\n'
	printf 'KEJILION_PANEL_NETWORK_SUBNET=172.29.255.240/28\n'
	printf 'KEJILION_PANEL_TRUSTED_PROXY_CIDRS=127.0.0.0/8,::1/128,172.29.255.240/28\n'
} >"$TEMP_ENV"
install -o root -g root -m 0600 "$TEMP_ENV" "$ENV_TARGET"
rm -f -- "$TEMP_ENV"
TEMP_ENV=

docker compose --env-file "$ENV_TARGET" -f "$COMPOSE_TARGET" config --quiet
systemctl daemon-reload
systemctl enable --now kejilion-agent.service

socket_ready=false
attempt=0
while [ "$attempt" -lt 20 ]; do
	if [ -S /run/kejilion-panel/agent.sock ]; then
		socket_ready=true
		break
	fi
	attempt=$((attempt + 1))
	sleep 1
done
[ "$socket_ready" = true ] || fail "Agent socket was not created; inspect: journalctl -u kejilion-agent"

docker compose --env-file "$ENV_TARGET" -f "$COMPOSE_TARGET" pull
docker compose --env-file "$ENV_TARGET" -f "$COMPOSE_TARGET" up -d

healthy=false
attempt=0
while [ "$attempt" -lt 30 ]; do
	if docker compose --env-file "$ENV_TARGET" -f "$COMPOSE_TARGET" \
		exec -T panel /paneld healthcheck >/dev/null 2>&1; then
		healthy=true
		break
	fi
	attempt=$((attempt + 1))
	sleep 2
done
[ "$healthy" = true ] || fail "panel health check failed; previous files are in $BACKUP_DIR"

printf '\nKejilion Panel is running on 127.0.0.1:%s.\n' "$PORT"
printf 'Public URL: %s\n' "$PUBLIC_URL"
printf 'Backup: %s\n' "$BACKUP_DIR"
if [ -s "$DATA_DIR/bootstrap.token" ]; then
	printf 'Read the one-time setup token as root from: %s\n' "$DATA_DIR/bootstrap.token"
fi
printf 'No kejilion.sh, /home/web, Nginx, firewall, or existing site file was changed.\n'
