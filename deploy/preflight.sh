#!/bin/sh
set -eu

PORT=18443
PUBLIC_URL=
FAILURES=0
WARNINGS=0

usage() {
	cat <<'EOF'
Usage:
  ./deploy/preflight.sh --public-url https://panel.example.com [--port 18443]

This command is read-only. It does not connect to the Docker socket, start a
service, create a directory, or change kejilion.sh and /home/web.
EOF
}

ok() {
	printf '[OK]   %s\n' "$*"
}

warn() {
	WARNINGS=$((WARNINGS + 1))
	printf '[WARN] %s\n' "$*" >&2
}

fail() {
	FAILURES=$((FAILURES + 1))
	printf '[FAIL] %s\n' "$*" >&2
}

while [ "$#" -gt 0 ]; do
	case "$1" in
		--public-url)
			[ "$#" -ge 2 ] || {
				fail "--public-url requires a value"
				break
			}
			PUBLIC_URL=$2
			shift 2
			;;
		--port)
			[ "$#" -ge 2 ] || {
				fail "--port requires a value"
				break
			}
			PORT=$2
			shift 2
			;;
		-h|--help)
			usage
			exit 0
			;;
		*)
			fail "unknown argument: $1"
			shift
			;;
	esac
done

if [ "$(uname -s 2>/dev/null || true)" = Linux ]; then
	ok "Linux host detected"
else
	fail "production deployment requires Linux"
fi

if [ "$(id -u)" -eq 0 ]; then
	ok "running as root"
else
	warn "run the installer with sudo after this preflight"
fi

case "$PORT" in
	''|*[!0-9]*)
		fail "--port must be numeric"
		;;
	*)
		if [ "$PORT" -ge 1024 ] && [ "$PORT" -le 65535 ]; then
			ok "local panel port is valid: $PORT"
		else
			fail "--port must be between 1024 and 65535"
		fi
		;;
esac

if printf '%s' "$PUBLIC_URL" |
	grep -Eq '^https://([A-Za-z0-9]([A-Za-z0-9.-]*[A-Za-z0-9])?)(:[0-9]{1,5})?$'; then
	ok "public URL is an HTTPS origin: $PUBLIC_URL"
else
	fail "--public-url must be an HTTPS origin without a path, query, or fragment"
fi

for command_name in docker systemctl getent groupadd install openssl sha256sum; do
	if command -v "$command_name" >/dev/null 2>&1; then
		ok "command available: $command_name"
	else
		fail "required command not found: $command_name"
	fi
done

if command -v docker >/dev/null 2>&1; then
	if docker compose version >/dev/null 2>&1; then
		ok "Docker Compose v2 is available"
	else
		fail "Docker Compose v2 is required"
	fi
fi

# Deliberately do not call docker info or open docker.sock here: a read-only
# preflight must not socket-activate a stopped Docker daemon.
if [ -S /var/run/docker.sock ]; then
	ok "Docker Unix socket exists"
else
	warn "Docker Unix socket is absent; Docker views will degrade until Docker is running"
fi

for web_path in /home/web /home/web/conf.d /home/web/html /home/web/certs; do
	if [ -d "$web_path" ] && [ -r "$web_path" ]; then
		ok "Kejilion path is readable: $web_path"
	else
		warn "Kejilion path is unavailable: $web_path"
	fi
done

if command -v ss >/dev/null 2>&1; then
	if ss -H -ltn 2>/dev/null | awk '{print $4}' |
		grep -Eq "(^|:)$PORT$"; then
		fail "TCP port $PORT is already listening"
	else
		ok "TCP port $PORT is not currently listening"
	fi
else
	warn "ss is unavailable; local port occupancy was not checked"
fi

for managed_path in \
	/etc/kejilion-panel \
	/opt/kejilion-panel \
	/var/lib/kejilion-panel \
	/usr/local/libexec/kejilion-agent \
	/etc/systemd/system/kejilion-agent.service; do
	if [ -e "$managed_path" ]; then
		warn "existing Panel resource found (installer will treat this as an upgrade): $managed_path"
	fi
done

if [ "$FAILURES" -ne 0 ]; then
	printf '\nPreflight failed: %s failure(s), %s warning(s).\n' "$FAILURES" "$WARNINGS" >&2
	exit 1
fi

printf '\nPreflight passed with %s warning(s). No host state was changed.\n' "$WARNINGS"
