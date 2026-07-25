#!/bin/sh
set -eu

SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
PROJECT_DIR=$(CDPATH= cd -- "$SCRIPT_DIR/../.." && pwd)
FAKE_BIN=$SCRIPT_DIR/fake-bin
TEST_DIR=$(mktemp -d /tmp/kejilion-panel-install-test.XXXXXX)

cleanup() {
	case "$TEST_DIR" in
		/tmp/kejilion-panel-install-test.*)
			rm -rf -- "$TEST_DIR"
			;;
		*)
			echo "refusing to remove unexpected test directory: $TEST_DIR" >&2
			;;
	esac
}
trap cleanup EXIT HUP INT TERM

DOCKER_LOG=$TEST_DIR/docker.log
: >"$DOCKER_LOG"
AGENT_BINARY=$FAKE_BIN/agent
AGENT_SHA=$(sha256sum "$AGENT_BINARY" | awk '{print $1}')
IMAGE=docker.io/example/kejilion-panel@sha256:0000000000000000000000000000000000000000000000000000000000000000

run_installer() {
	PATH="$FAKE_BIN:$PATH" \
	KP_DOCKER_LOG="$DOCKER_LOG" \
	sh "$PROJECT_DIR/deploy/install.sh" \
		--agent-binary "$AGENT_BINARY" \
		--agent-sha256 "$AGENT_SHA" \
		--image "$IMAGE" \
		--public-url https://panel.example.com \
		--port 18443 \
		--network-subnet 172.29.255.240/28 \
		--dry-run
}

run_installer >"$TEST_DIR/success.out"
grep -F 'Docker daemon was not queried and no host state was changed.' "$TEST_DIR/success.out" >/dev/null
test "$(wc -l <"$DOCKER_LOG" | tr -d ' ')" = 1
grep -Fx -- '--host unix:///var/run/docker.sock compose version' "$DOCKER_LOG" >/dev/null

BEFORE_LINES=$(wc -l <"$DOCKER_LOG" | tr -d ' ')
if DOCKER_HOST=tcp://198.51.100.1:2375 run_installer >"$TEST_DIR/remote.out" 2>&1; then
	echo "installer accepted DOCKER_HOST during dry-run" >&2
	exit 1
fi
test "$(wc -l <"$DOCKER_LOG" | tr -d ' ')" = "$BEFORE_LINES"
grep -F 'DOCKER_HOST and DOCKER_CONTEXT must be unset' "$TEST_DIR/remote.out" >/dev/null

if KP_AGENT_VERSION=0.0.1 run_installer >"$TEST_DIR/version.out" 2>&1; then
	echo "installer accepted a mismatched Agent version" >&2
	exit 1
fi
grep -F 'does not match 0.1.0 v1alpha1' "$TEST_DIR/version.out" >/dev/null

if PATH="$FAKE_BIN:$PATH" KP_DOCKER_LOG="$DOCKER_LOG" \
	sh "$PROJECT_DIR/deploy/install.sh" \
		--agent-binary "$AGENT_BINARY" \
		--agent-sha256 "$AGENT_SHA" \
		--image "$IMAGE" \
		--public-url https://panel.example.com \
		--network-subnet 172.29.255.241/28 \
		--dry-run >"$TEST_DIR/subnet.out" 2>&1; then
	echo "installer accepted a misaligned subnet" >&2
	exit 1
fi
grep -F 'aligned RFC1918 IPv4 /28' "$TEST_DIR/subnet.out" >/dev/null

if PATH="$FAKE_BIN:$PATH" KP_DOCKER_LOG="$DOCKER_LOG" KP_ROUTE_CONFLICT=1 \
	sh "$PROJECT_DIR/deploy/install.sh" \
		--agent-binary "$AGENT_BINARY" \
		--agent-sha256 "$AGENT_SHA" \
		--image "$IMAGE" \
		--public-url https://panel.example.com \
		--network-subnet 172.29.255.240/28 \
		--dry-run >"$TEST_DIR/route.out" 2>&1; then
	echo "installer accepted an overlapping route" >&2
	exit 1
fi
grep -F 'overlaps an existing host route' "$TEST_DIR/route.out" >/dev/null

if KP_GROUP_MEMBER=1 run_installer >"$TEST_DIR/install-group.out" 2>&1; then
	echo "installer accepted an Agent group with supplemental members" >&2
	exit 1
fi
grep -F 'group has supplemental members' "$TEST_DIR/install-group.out" >/dev/null

if KP_GROUP_EXISTS=1 KP_PRIMARY_USER=1 \
	run_installer >"$TEST_DIR/install-primary-group.out" 2>&1; then
	echo "installer accepted an Agent group used as a primary group" >&2
	exit 1
fi
grep -F 'gid is a primary group for host users' "$TEST_DIR/install-primary-group.out" >/dev/null

if KP_GROUP_EXISTS=1 KP_PASSWD_FAIL=1 \
	run_installer >"$TEST_DIR/install-passwd.out" 2>&1; then
	echo "installer accepted incomplete host user enumeration" >&2
	exit 1
fi
grep -F 'cannot enumerate host users' "$TEST_DIR/install-passwd.out" >/dev/null

if KP_UNIT_LOAD_STATE=loaded \
	run_installer >"$TEST_DIR/install-unit.out" 2>&1; then
	echo "installer accepted an already loaded Agent unit" >&2
	exit 1
fi
grep -F 'existing or loaded kejilion-agent.service' "$TEST_DIR/install-unit.out" >/dev/null

if KP_WEB_ROOT_FAIL=1 \
	run_installer >"$TEST_DIR/install-web-root-missing.out" 2>&1; then
	echo "installer accepted a missing Kejilion Web root" >&2
	exit 1
fi
grep -F 'Kejilion Web root is unavailable' "$TEST_DIR/install-web-root-missing.out" >/dev/null

if KP_WEB_ROOT_KIND='regular file' \
	run_installer >"$TEST_DIR/install-web-root-file.out" 2>&1; then
	echo "installer accepted a non-directory Kejilion Web root" >&2
	exit 1
fi
grep -F 'Kejilion Web root is not a directory' "$TEST_DIR/install-web-root-file.out" >/dev/null

if KP_DOCKER_SOCKET_FAIL=1 \
	run_installer >"$TEST_DIR/install-docker-socket-missing.out" 2>&1; then
	echo "installer accepted a missing local Docker socket" >&2
	exit 1
fi
grep -F 'local Docker Unix socket is unavailable' "$TEST_DIR/install-docker-socket-missing.out" >/dev/null

if KP_DOCKER_SOCKET_KIND='regular file' \
	run_installer >"$TEST_DIR/install-docker-socket-file.out" 2>&1; then
	echo "installer accepted a non-socket local Docker endpoint" >&2
	exit 1
fi
grep -F 'local Docker endpoint is not a Unix socket' "$TEST_DIR/install-docker-socket-file.out" >/dev/null

grep -F 'trap cleanup EXIT' "$PROJECT_DIR/deploy/install.sh" >/dev/null
grep -F "trap 'exit 129' HUP" "$PROJECT_DIR/deploy/install.sh" >/dev/null
grep -F "trap 'exit 130' INT" "$PROJECT_DIR/deploy/install.sh" >/dev/null
grep -F "trap 'exit 143' TERM" "$PROJECT_DIR/deploy/install.sh" >/dev/null
grep -F '/run/kejilion-panel \' "$PROJECT_DIR/deploy/install.sh" >/dev/null
grep -F '/run/kejilion-panel \' "$PROJECT_DIR/deploy/preflight.sh" >/dev/null
grep -F 'CRITICAL: automatic failure cleanup could not be fully verified.' \
	"$PROJECT_DIR/deploy/install.sh" >/dev/null

: >"$DOCKER_LOG"
PATH="$FAKE_BIN:$PATH" KP_DOCKER_LOG="$DOCKER_LOG" \
	sh "$PROJECT_DIR/deploy/preflight.sh" \
		--public-url https://panel.example.com \
		--port 18443 \
		--network-subnet 172.29.255.240/28 \
		>"$TEST_DIR/preflight.out" 2>&1 || true
test "$(wc -l <"$DOCKER_LOG" | tr -d ' ')" = 1
grep -Fx -- '--host unix:///var/run/docker.sock compose version' "$DOCKER_LOG" >/dev/null

: >"$DOCKER_LOG"
PATH="$FAKE_BIN:$PATH" KP_DOCKER_LOG="$DOCKER_LOG" KP_GROUP_MEMBER=1 \
	sh "$PROJECT_DIR/deploy/preflight.sh" \
		--public-url https://panel.example.com \
		--port 18443 \
		--network-subnet 172.29.255.240/28 \
		>"$TEST_DIR/group.out" 2>&1 || true
grep -F 'kejilion-panel group has supplemental members' "$TEST_DIR/group.out" >/dev/null

echo "install_safety=pass"
