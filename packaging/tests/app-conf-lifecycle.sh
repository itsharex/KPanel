#!/usr/bin/env bash
# shellcheck disable=SC2034,SC2329
set -eu

[ "${KPANEL_APP_CONF_TEST_ROOTFS:-}" = 1 ] && [ -f /.dockerenv ] || {
	echo "refusing to run outside the disposable app-conf test container" >&2
	exit 1
}

PROJECT_DIR=${1:-/src}
TEST_DIR=$(mktemp -d /tmp/kpanel-app-conf-test.XXXXXX)
FAKE_BIN="$TEST_DIR/bin"
MOCK_STATE="$TEST_DIR/state"
mkdir -p "$FAKE_BIN" "$MOCK_STATE" /run/systemd/system /home/docker

cleanup() {
	case "$TEST_DIR" in
		/tmp/kpanel-app-conf-test.*)
			rm -rf -- "$TEST_DIR"
			;;
	esac
	rm -rf -- /home/docker/kpanel
}
trap cleanup EXIT HUP INT TERM

cat >"$FAKE_BIN/docker" <<'EOF'
#!/bin/sh
set -eu
state=${KPANEL_MOCK_STATE:?}
case "$1 ${2:-}" in
	"compose version"|"pull docker.io/kjlion/kejilion-panel:0.14.0")
		exit 0
		;;
	"ps -a")
		exit 0
		;;
	"network inspect")
		if [ "${3:-}" = "--format" ]; then
			printf '%s\n' '172.30.0.0/16'
			exit 0
		fi
		[ -f "$state/network" ]
		exit
		;;
	"create --name")
		: >"$state/release-container"
		printf '%s\n' mock-release-container
		exit 0
		;;
	"cp "*)
		destination=$3
		cat >"$destination" <<'AGENT'
#!/bin/sh
case "${1:-}" in
	version) printf '%s\n' '0.14.0 v1alpha1' ;;
	healthcheck) exit 0 ;;
	*) exit 0 ;;
esac
AGENT
		chmod 755 "$destination"
		exit 0
		;;
	"rm "*)
		exit 0
		;;
	"inspect --format")
		case "$3" in
			*NetworkSettings*) printf '%s\n' 1 ;;
			*)
				if [ "${KPANEL_MOCK_HEALTH_FAIL:-0}" = 1 ]; then
					printf '%s\n' unhealthy
				else
					printf '%s\n' healthy
				fi
				;;
		esac
		exit 0
		;;
	"compose --env-file")
		case "${4:-}" in
			create) : >"$state/network" ;;
			up) : ;;
			down) rm -f "$state/network" ;;
			*) exit 2 ;;
		esac
		exit 0
		;;
esac
echo "unexpected docker invocation: $*" >&2
exit 2
EOF

cat >"$FAKE_BIN/systemctl" <<'EOF'
#!/bin/sh
case "$1" in
	link|daemon-reload|enable|start|stop|disable) exit 0 ;;
esac
echo "unexpected systemctl invocation: $*" >&2
exit 2
EOF

cat >"$FAKE_BIN/getent" <<'EOF'
#!/bin/sh
[ "$1" = group ] && [ "$2" = kejilion-panel ] || exit 1
printf '%s\n' 'kejilion-panel:x:987:'
EOF

cat >"$FAKE_BIN/groupadd" <<'EOF'
#!/bin/sh
exit 0
EOF

cat >"$FAKE_BIN/groupdel" <<'EOF'
#!/bin/sh
exit 0
EOF

cat >"$FAKE_BIN/sleep" <<'EOF'
#!/bin/sh
exit 0
EOF
chmod 755 "$FAKE_BIN"/*

run_lifecycle() {
	local ipv4_address="198.51.100.25"

	docker_app_plus() {
		:
	}
	check_docker_app_ip() {
		:
	}

	# shellcheck source=/dev/null
	. "$PROJECT_DIR/packaging/kejilion-app/kpanel.conf"
	docker_port="18080"
	docker_app_install

	grep -F 'image: docker.io/kjlion/kejilion-panel:0.14.0' \
		/home/docker/kpanel/docker-compose.yml >/dev/null
	grep -F -- '- "18080:8080"' /home/docker/kpanel/docker-compose.yml >/dev/null
	grep -Fx 'KPANEL_PUBLIC_URL=http://198.51.100.25:18080' \
		/home/docker/kpanel/.env >/dev/null
	grep -Fx 'KPANEL_TRUSTED_PROXY_CIDRS=127.0.0.0/8,::1/128,172.30.0.0/16' \
		/home/docker/kpanel/.env >/dev/null
	test "$(grep -c '^    networks:$' /home/docker/kpanel/docker-compose.yml)" = 1
	grep -F 'ExecStart=/home/docker/kpanel/bin/kejilion-agent' \
		/home/docker/kpanel/kejilion-agent.service >/dev/null

	docker_app_update
	test "$(/home/docker/kpanel/bin/kejilion-agent version)" = '0.14.0 v1alpha1'

	docker_app_uninstall
	[ ! -e /home/docker/kpanel ]
}

run_failed_install() {
	local ipv4_address="198.51.100.25"

	docker_app_plus() {
		:
	}
	check_docker_app_ip() {
		:
	}

	# shellcheck source=/dev/null
	. "$PROJECT_DIR/packaging/kejilion-app/kpanel.conf"
	docker_port="18080"
	if KPANEL_MOCK_HEALTH_FAIL=1 docker_app_install; then
		echo "failed install unexpectedly succeeded" >&2
		return 1
	fi
	[ ! -e /home/docker/kpanel ]
	[ ! -e "$MOCK_STATE/network" ]
}

PATH="$FAKE_BIN:$PATH" KPANEL_MOCK_STATE="$MOCK_STATE" run_lifecycle
PATH="$FAKE_BIN:$PATH" KPANEL_MOCK_STATE="$MOCK_STATE" run_failed_install
printf '%s\n' "app_conf_lifecycle=pass"
