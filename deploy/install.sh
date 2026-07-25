#!/bin/sh
set -eu

umask 027
LC_ALL=C
export LC_ALL

SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
PROJECT_DIR=$(CDPATH= cd -- "$SCRIPT_DIR/.." && pwd)

AGENT_BINARY=
AGENT_SHA256=
IMAGE=
PUBLIC_URL=
PORT=18443
NETWORK_SUBNET=172.29.255.240/28
DRY_RUN=false

usage() {
	cat <<'EOF'
Usage:
  sudo ./deploy/install.sh \
    --agent-binary ./dist/linux-amd64/kejilion-agent \
    --agent-sha256 <64-character-sha256> \
    --image docker.io/OWNER/kejilion-panel@sha256:<64-character-digest> \
    --public-url https://panel.example.com \
    [--port 18443] \
    [--network-subnet 172.29.255.240/28] \
    [--dry-run]

The installer only creates KPanel files and services. It does not edit
kejilion.sh, /home/web, Nginx configuration, firewall rules, or existing sites.
EOF
}

fail() {
	printf 'install: %s\n' "$*" >&2
	exit 1
}

validate_private_subnet() {
	subnet=$1
	case "$subnet" in
		*/*)
			address=${subnet%/*}
			prefix=${subnet#*/}
			;;
		*)
			return 1
			;;
	esac
	[ "$prefix" = 28 ] || return 1
	printf '%s' "$address" |
		grep -Eq '^[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}$' ||
		return 1
	old_ifs=$IFS
	IFS=.
	set -- $address
	IFS=$old_ifs
	[ "$#" -eq 4 ] || return 1
	for octet in "$@"; do
		case "$octet" in
			0|[1-9]|[1-9][0-9]|[1-9][0-9][0-9]) ;;
			*) return 1 ;;
		esac
		[ "$octet" -le 255 ] || return 1
	done
	first=$1
	second=$2
	fourth=$4
	case "$first:$second" in
		10:*) ;;
		172:*)
			[ "$second" -ge 16 ] && [ "$second" -le 31 ] || return 1
			;;
		192:168) ;;
		*) return 1 ;;
	esac
	[ $((fourth % 16)) -eq 0 ] || return 1
}

network_routes() {
	matched_routes=$(ip -4 route show match "$NETWORK_SUBNET" 2>/dev/null) || return 1
	rooted_routes=$(ip -4 route show root "$NETWORK_SUBNET" 2>/dev/null) || return 1
	printf '%s\n%s\n' "$matched_routes" "$rooted_routes" |
		while IFS= read -r route; do
			case "$route" in
				''|default\ *) ;;
				*) printf '%s\n' "$route" ;;
			esac
		done
}

docker_local() {
	docker --host unix:///var/run/docker.sock "$@"
}

inspect_panel_group() {
	PANEL_GROUP_ENTRY=$(getent group "$PANEL_GROUP" 2>/dev/null) || return 1
	PANEL_GROUP_MATCHES=$(printf '%s\n' "$PANEL_GROUP_ENTRY" |
		awk -F: '$1 == "kejilion-panel" {count++} END {print count + 0}')
	[ "$PANEL_GROUP_MATCHES" -eq 1 ] ||
		fail "cannot uniquely resolve $PANEL_GROUP group"
	PANEL_GID=$(printf '%s\n' "$PANEL_GROUP_ENTRY" |
		awk -F: '$1 == "kejilion-panel" {print $3; exit}')
	PANEL_MEMBERS=$(printf '%s\n' "$PANEL_GROUP_ENTRY" |
		awk -F: '$1 == "kejilion-panel" {print $4; exit}')
	[ -n "$PANEL_GID" ] || fail "cannot resolve $PANEL_GROUP gid"
	[ -z "$PANEL_MEMBERS" ] ||
		fail "$PANEL_GROUP group has supplemental members and cannot be used as the Agent boundary"
	PASSWD_ENTRIES=$(getent passwd) ||
		fail "cannot enumerate host users for the Agent group boundary"
	[ -n "$PASSWD_ENTRIES" ] ||
		fail "host user enumeration returned no entries"
	PRIMARY_GROUP_USERS=$(printf '%s\n' "$PASSWD_ENTRIES" |
		awk -F: -v gid="$PANEL_GID" '$4 == gid {print $1}')
	[ -z "$PRIMARY_GROUP_USERS" ] ||
		fail "$PANEL_GROUP gid is a primary group for host users: $PRIMARY_GROUP_USERS"
}

inspect_systemd_unit_absence() {
	UNIT_LOAD_STATE=$(systemctl show \
		--property=LoadState --value kejilion-agent.service 2>/dev/null) ||
		fail "cannot query systemd for an existing kejilion-agent.service"
	UNIT_FRAGMENT_PATH=$(systemctl show \
		--property=FragmentPath --value kejilion-agent.service 2>/dev/null) ||
		fail "cannot query the kejilion-agent.service fragment path"
	UNIT_DROP_INS=$(systemctl show \
		--property=DropInPaths --value kejilion-agent.service 2>/dev/null) ||
		fail "cannot query kejilion-agent.service drop-ins"
	[ "$UNIT_LOAD_STATE" = "not-found" ] &&
		[ -z "$UNIT_FRAGMENT_PATH" ] &&
		[ -z "$UNIT_DROP_INS" ] ||
		fail "an existing or loaded kejilion-agent.service was found"
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
		--network-subnet)
			[ "$#" -ge 2 ] || fail "--network-subnet requires a value"
			NETWORK_SUBNET=$2
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
validate_private_subnet "$NETWORK_SUBNET" ||
	fail "--network-subnet must be an aligned RFC1918 IPv4 /28"

if [ -n "${DOCKER_HOST:-}" ] || [ -n "${DOCKER_CONTEXT:-}" ]; then
	fail "DOCKER_HOST and DOCKER_CONTEXT must be unset; installation is restricted to the local Docker socket"
fi
unset DOCKER_HOST DOCKER_CONTEXT

for command_name in \
	awk cat curl dirname docker getent grep groupadd id install ip mkdir \
	mktemp openssl rm rmdir sed sha256sum sleep ss stat systemctl systemd-analyze tr uname; do
	command -v "$command_name" >/dev/null 2>&1 || fail "required command not found: $command_name"
done
[ "$(uname -s)" = "Linux" ] || fail "production deployment requires Linux"
docker_local compose version >/dev/null 2>&1 || fail "Docker Compose v2 is required"
printf '%s  %s\n' "$AGENT_SHA256" "$AGENT_BINARY" | sha256sum --check --status ||
	fail "agent binary SHA-256 mismatch"
[ -f "$PROJECT_DIR/VERSION" ] || fail "release VERSION file is missing"
EXPECTED_VERSION=$(tr -d '\r\n' <"$PROJECT_DIR/VERSION")
printf '%s' "$EXPECTED_VERSION" |
	grep -Eq '^[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?$' ||
	fail "release VERSION is invalid"
AGENT_RELEASE=$("$AGENT_BINARY" version 2>/dev/null) ||
	fail "Agent binary cannot run on this host"
[ "$AGENT_RELEASE" = "$EXPECTED_VERSION v1alpha1" ] ||
	fail "Agent release $AGENT_RELEASE does not match $EXPECTED_VERSION v1alpha1"

NETWORK_ROUTES=$(network_routes) ||
	fail "cannot inspect host routes for Docker subnet safety"
[ -z "$NETWORK_ROUTES" ] ||
	fail "network subnet overlaps an existing host route: $NETWORK_ROUTES"

for managed_path in \
	/etc/kejilion-panel \
	/opt/kejilion-panel \
	/var/lib/kejilion-panel \
	/run/kejilion-panel \
	/usr/local/libexec/kejilion-agent \
	/etc/systemd/system/kejilion-agent.service \
	/etc/systemd/system/kejilion-agent.service.d \
	/run/systemd/system/kejilion-agent.service \
	/run/systemd/system/kejilion-agent.service.d \
	/usr/lib/systemd/system/kejilion-agent.service \
	/usr/lib/systemd/system/kejilion-agent.service.d \
	/lib/systemd/system/kejilion-agent.service \
	/lib/systemd/system/kejilion-agent.service.d; do
	{
		[ ! -e "$managed_path" ] && [ ! -L "$managed_path" ]
	} ||
		fail "existing Panel resource found; v0.1.0 installer only supports a fresh install: $managed_path"
done
inspect_systemd_unit_absence

WEB_ROOT_KIND=$(stat -L -c %F /home/web 2>/dev/null) ||
	fail "Kejilion Web root is unavailable: /home/web"
[ "$WEB_ROOT_KIND" = "directory" ] ||
	fail "Kejilion Web root is not a directory: /home/web"
DOCKER_SOCKET_KIND=$(stat -L -c %F /var/run/docker.sock 2>/dev/null) ||
	fail "local Docker Unix socket is unavailable: /var/run/docker.sock"
[ "$DOCKER_SOCKET_KIND" = "socket" ] ||
	fail "local Docker endpoint is not a Unix socket: /var/run/docker.sock"
systemctl is-active --quiet docker.service ||
	fail "Docker service is not already active; assess existing containers before starting it manually"

if ss -H -ltn 2>/dev/null | awk '{print $4}' | grep -Eq "(^|:)$PORT$"; then
	fail "TCP port $PORT is already listening"
fi

PANEL_GROUP=kejilion-panel
PANEL_GROUP_PRESENT=false
if inspect_panel_group; then
	PANEL_GROUP_PRESENT=true
fi

if [ "$DRY_RUN" = true ]; then
	printf 'Preflight passed.\n'
	printf 'Agent: %s\nImage: %s\nPublic URL: %s\nLocal port: %s\nNetwork subnet: %s\n' \
		"$AGENT_BINARY" "$IMAGE" "$PUBLIC_URL" "$PORT" "$NETWORK_SUBNET"
	printf 'Docker daemon was not queried and no host state was changed.\n'
	exit 0
fi

LOCK_DIR=/run/lock/kejilion-panel-install
mkdir "$LOCK_DIR" 2>/dev/null || fail "another installation is running"
TEMP_TOKEN=
TEMP_ENV=
INSTALL_SUCCEEDED=false
AGENT_ENABLE_ATTEMPTED=false
AGENT_START_ATTEMPTED=false
PANEL_START_ATTEMPTED=false
cleanup_failure() {
	CLEANUP_FAILURES=$((CLEANUP_FAILURES + 1))
	printf 'CRITICAL cleanup: %s\n' "$*" >&2
}

cleanup() {
	cleanup_status=$?
	trap - EXIT HUP INT TERM
	CLEANUP_FAILURES=0
	if [ "$INSTALL_SUCCEEDED" != true ]; then
		if [ "$PANEL_START_ATTEMPTED" = true ]; then
			if systemctl is-active --quiet docker.service; then
				if PANEL_PROJECT=$(docker_local container inspect \
					--format '{{index .Config.Labels "com.docker.compose.project"}}' \
					kejilion-panel 2>/dev/null); then
					if PANEL_SERVICE=$(docker_local container inspect \
						--format '{{index .Config.Labels "com.docker.compose.service"}}' \
						kejilion-panel 2>/dev/null); then
						if [ "$PANEL_PROJECT" = "kejilion-panel" ] &&
							[ "$PANEL_SERVICE" = "panel" ]; then
							docker_local container stop --time 10 \
								kejilion-panel >/dev/null 2>&1 || true
							if PANEL_RUNNING=$(docker_local container inspect \
								--format '{{.State.Running}}' \
								kejilion-panel 2>/dev/null); then
								[ "$PANEL_RUNNING" = "false" ] ||
									cleanup_failure "kejilion-panel container is still running"
							else
								cleanup_failure "cannot verify kejilion-panel container state"
							fi
						else
							cleanup_failure \
								"container ownership labels changed; it was not stopped"
						fi
					else
						cleanup_failure "cannot read the Panel service ownership label"
					fi
				else
					cleanup_failure "cannot determine whether the attempted Panel container exists"
				fi
			else
				cleanup_failure \
					"Docker service is inactive; Panel container state cannot be verified"
			fi
		fi
		if [ "$AGENT_START_ATTEMPTED" = true ]; then
			systemctl stop kejilion-agent.service >/dev/null 2>&1 || true
			if AGENT_ACTIVE_STATE=$(systemctl show \
				--property=ActiveState --value kejilion-agent.service 2>/dev/null); then
				case "$AGENT_ACTIVE_STATE" in
					inactive|failed) ;;
					*) cleanup_failure \
						"Agent ActiveState is $AGENT_ACTIVE_STATE after stop" ;;
				esac
			else
				cleanup_failure "cannot verify Agent ActiveState after stop"
			fi
		fi
		if [ "$AGENT_ENABLE_ATTEMPTED" = true ]; then
			systemctl disable kejilion-agent.service >/dev/null 2>&1 || true
			if AGENT_UNIT_FILE_STATE=$(systemctl show \
				--property=UnitFileState --value kejilion-agent.service 2>/dev/null); then
				[ "$AGENT_UNIT_FILE_STATE" = "disabled" ] ||
					cleanup_failure \
						"Agent UnitFileState is $AGENT_UNIT_FILE_STATE after disable"
			else
				cleanup_failure "cannot verify Agent UnitFileState after disable"
			fi
		fi
		if [ "$PANEL_START_ATTEMPTED" = true ] ||
			[ "$AGENT_START_ATTEMPTED" = true ] ||
			[ "$AGENT_ENABLE_ATTEMPTED" = true ]; then
			if [ "$CLEANUP_FAILURES" -eq 0 ]; then
				printf '%s\n' \
					'Installation failed; no newly started Panel process remains active or enabled.' \
					'Panel files and logs were retained for diagnosis. Follow docs/deployment.md before retrying.' >&2
			else
				printf '%s\n' \
					'CRITICAL: automatic failure cleanup could not be fully verified.' \
					'Do not retry or start Docker/Agent until docs/deployment.md recovery checks are complete.' >&2
			fi
		fi
	fi
	[ -z "$TEMP_TOKEN" ] || rm -f -- "$TEMP_TOKEN"
	[ -z "$TEMP_ENV" ] || rm -f -- "$TEMP_ENV"
	rmdir "$LOCK_DIR" 2>/dev/null || true
	exit "$cleanup_status"
}
trap cleanup EXIT
trap 'exit 129' HUP
trap 'exit 130' INT
trap 'exit 143' TERM

systemctl is-active --quiet docker.service ||
	fail "Docker service is not already active; assess existing containers before starting it manually"
docker_local info >/dev/null 2>&1 ||
	fail "the active local Docker daemon is unavailable through /var/run/docker.sock"
if docker_local container inspect kejilion-panel >/dev/null 2>&1; then
	fail "Docker container name kejilion-panel is already in use"
fi
if docker_local network inspect kejilion-panel-internal >/dev/null 2>&1; then
	fail "Docker network name kejilion-panel-internal is already in use"
fi
PROJECT_CONTAINERS=$(docker_local container ls --all --quiet \
	--filter label=com.docker.compose.project=kejilion-panel) ||
	fail "cannot enumerate existing Panel Compose containers"
[ -z "$PROJECT_CONTAINERS" ] ||
	fail "existing containers use the kejilion-panel Compose project: $PROJECT_CONTAINERS"
PROJECT_NETWORKS=$(docker_local network ls --quiet \
	--filter label=com.docker.compose.project=kejilion-panel) ||
	fail "cannot enumerate existing Panel Compose networks"
[ -z "$PROJECT_NETWORKS" ] ||
	fail "existing networks use the kejilion-panel Compose project: $PROJECT_NETWORKS"
PROJECT_VOLUMES=$(docker_local volume ls --quiet \
	--filter label=com.docker.compose.project=kejilion-panel) ||
	fail "cannot enumerate existing Panel Compose volumes"
[ -z "$PROJECT_VOLUMES" ] ||
	fail "existing volumes use the kejilion-panel Compose project: $PROJECT_VOLUMES"
docker_local image pull "$IMAGE" >/dev/null ||
	fail "cannot pull the pinned Panel image"
IMAGE_USER=$(docker_local image inspect --format '{{.Config.User}}' "$IMAGE")
[ "$IMAGE_USER" = "65532:65532" ] ||
	fail "Panel image must run as 65532:65532, got: $IMAGE_USER"
IMAGE_HEALTHCHECK=$(docker_local image inspect --format '{{join .Config.Healthcheck.Test " "}}' "$IMAGE")
[ "$IMAGE_HEALTHCHECK" = "CMD /paneld healthcheck" ] ||
	fail "Panel image has an unexpected healthcheck: $IMAGE_HEALTHCHECK"
IMAGE_VERSION=$(docker_local image inspect --format '{{index .Config.Labels "org.opencontainers.image.version"}}' "$IMAGE")
[ "$IMAGE_VERSION" = "$EXPECTED_VERSION" ] ||
	fail "Panel image version $IMAGE_VERSION does not match release $EXPECTED_VERSION"

if [ "$PANEL_GROUP_PRESENT" = true ]; then
	inspect_panel_group ||
		fail "$PANEL_GROUP group disappeared during installation"
else
	groupadd --system "$PANEL_GROUP"
	inspect_panel_group ||
		fail "cannot inspect the newly created $PANEL_GROUP group"
fi
[ -n "$PANEL_GID" ] || fail "cannot resolve $PANEL_GROUP gid"

ETC_DIR=/etc/kejilion-panel
OPT_DIR=/opt/kejilion-panel
DATA_DIR=/var/lib/kejilion-panel/panel
AGENT_TARGET=/usr/local/libexec/kejilion-agent
SERVICE_TARGET=/etc/systemd/system/kejilion-agent.service
COMPOSE_TARGET=$OPT_DIR/compose.yml
ENV_TARGET=$OPT_DIR/.env
TOKEN_TARGET=$ETC_DIR/agent.token

install -d -o root -g "$PANEL_GROUP" -m 0750 "$ETC_DIR"
install -d -o root -g root -m 0755 "$OPT_DIR"
install -d -o 65532 -g 65532 -m 0700 "$DATA_DIR"
install -d -o root -g root -m 0755 "$(dirname "$AGENT_TARGET")"

TEMP_TOKEN=$(mktemp "$ETC_DIR/.agent.token.XXXXXX")
openssl rand -hex 32 >"$TEMP_TOKEN"
install -o root -g "$PANEL_GROUP" -m 0640 "$TEMP_TOKEN" "$TOKEN_TARGET"
rm -f -- "$TEMP_TOKEN"
TEMP_TOKEN=

install -o root -g root -m 0755 "$AGENT_BINARY" "$AGENT_TARGET"
install -o root -g root -m 0644 \
	"$PROJECT_DIR/deploy/systemd/kejilion-agent.service" "$SERVICE_TARGET"
install -o root -g root -m 0644 \
	"$PROJECT_DIR/deploy/compose/compose.yml" "$COMPOSE_TARGET"
systemd-analyze verify "$SERVICE_TARGET" ||
	fail "Agent systemd unit is not supported by this host"

TEMP_ENV=$(mktemp "$OPT_DIR/.env.XXXXXX")
{
	printf 'KEJILION_PANEL_IMAGE=%s\n' "$IMAGE"
	printf 'KEJILION_PANEL_GID=%s\n' "$PANEL_GID"
	printf 'KEJILION_PANEL_DATA_DIR_HOST=%s\n' "$DATA_DIR"
	printf 'KEJILION_PANEL_PORT=%s\n' "$PORT"
	printf 'KEJILION_PANEL_PUBLIC_URL=%s\n' "$PUBLIC_URL"
	printf 'KEJILION_PANEL_SECURE_COOKIE=true\n'
	printf 'KEJILION_PANEL_NETWORK_SUBNET=%s\n' "$NETWORK_SUBNET"
	printf 'KEJILION_PANEL_TRUSTED_PROXY_CIDRS=127.0.0.0/8,::1/128,%s\n' "$NETWORK_SUBNET"
} >"$TEMP_ENV"
install -o root -g root -m 0600 "$TEMP_ENV" "$ENV_TARGET"
rm -f -- "$TEMP_ENV"
TEMP_ENV=

docker_local compose --project-name kejilion-panel \
	--env-file "$ENV_TARGET" -f "$COMPOSE_TARGET" config --quiet
systemctl daemon-reload
AGENT_ENABLE_ATTEMPTED=true
systemctl enable kejilion-agent.service
AGENT_START_ATTEMPTED=true
systemctl restart kejilion-agent.service
systemctl is-active --quiet kejilion-agent.service ||
	fail "Agent service did not become active"

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
"$AGENT_TARGET" healthcheck ||
	fail "Agent readiness, version, or protocol healthcheck failed"

PANEL_START_ATTEMPTED=true
docker_local compose --project-name kejilion-panel \
	--env-file "$ENV_TARGET" -f "$COMPOSE_TARGET" up -d --pull never

healthy=false
attempt=0
while [ "$attempt" -lt 30 ]; do
	if docker_local compose --project-name kejilion-panel \
		--env-file "$ENV_TARGET" -f "$COMPOSE_TARGET" \
		exec -T panel /paneld healthcheck >/dev/null 2>&1 &&
		docker_local compose --project-name kejilion-panel \
			--env-file "$ENV_TARGET" -f "$COMPOSE_TARGET" \
			exec -T panel /paneld agent-healthcheck >/dev/null 2>&1 &&
		curl --noproxy '*' --fail --silent --show-error --max-time 4 \
			"http://127.0.0.1:$PORT/api/v1/health" >/dev/null 2>&1; then
		healthy=true
		break
	fi
	attempt=$((attempt + 1))
	sleep 2
done
if [ "$healthy" != true ]; then
	docker_local compose --project-name kejilion-panel \
		--env-file "$ENV_TARGET" -f "$COMPOSE_TARGET" \
		exec -T panel /paneld healthcheck || true
	docker_local compose --project-name kejilion-panel \
		--env-file "$ENV_TARGET" -f "$COMPOSE_TARGET" \
		exec -T panel /paneld agent-healthcheck || true
	curl --noproxy '*' --fail --silent --show-error --max-time 4 \
		"http://127.0.0.1:$PORT/api/v1/health" >/dev/null || true
	fail "panel or Panel-to-Agent health check failed; inspect the retained resources before retrying"
fi
INSTALL_SUCCEEDED=true

printf '\nKPanel is running on 127.0.0.1:%s.\n' "$PORT"
printf 'Public URL: %s\n' "$PUBLIC_URL"
if [ -s "$DATA_DIR/bootstrap.token" ]; then
	printf 'Read the one-time setup token as root from: %s\n' "$DATA_DIR/bootstrap.token"
fi
printf 'No kejilion.sh, /home/web, Nginx, firewall, or existing site file was changed.\n'
