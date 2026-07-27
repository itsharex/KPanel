#!/usr/bin/env bash
# shellcheck disable=SC2034,SC2329
set -eu

[ "${KPANEL_APP_CONF_TEST_ROOTFS:-}" = 1 ] && [ -f /.dockerenv ] || {
	echo "refusing to run outside the disposable app-conf test container" >&2
	exit 1
}

PROJECT_DIR=${1:-/src}
RELEASE_VERSION=$(tr -d '\r\n' <"$PROJECT_DIR/VERSION")
export KPANEL_RELEASE_VERSION=$RELEASE_VERSION
TEST_DIR=$(mktemp -d /tmp/kpanel-app-conf-test.XXXXXX)
FAKE_BIN="$TEST_DIR/bin"
MOCK_STATE="$TEST_DIR/state"
mkdir -p "$FAKE_BIN" "$MOCK_STATE" /run/systemd/system /etc/systemd/system /home/docker

cleanup() {
	case "$TEST_DIR" in
		/tmp/kpanel-app-conf-test.*)
			rm -rf -- "$TEST_DIR"
			;;
	esac
	rm -rf -- /home/docker/kpanel
	rm -f /bin/systemctl
}
trap cleanup EXIT HUP INT TERM

cat >"$FAKE_BIN/docker" <<'EOF'
#!/bin/sh
set -eu
state=${KPANEL_MOCK_STATE:-}
require_state() {
	[ -n "$state" ] || {
		echo "KPANEL_MOCK_STATE is required for lifecycle mutations" >&2
		exit 2
	}
}
case "$1 ${2:-}" in
	"compose version"|"pull docker.io/kjlion/kejilion-panel:$KPANEL_RELEASE_VERSION")
		exit 0
		;;
	"ps -a")
		exit 0
		;;
	"network inspect")
		require_state
		if [ "${3:-}" = "--format" ]; then
			printf '%s\n' '172.30.0.0/16'
			exit 0
		fi
		[ -f "$state/network" ]
		exit
		;;
	"create --name")
		require_state
		: >"$state/release-container"
		printf '%s\n' mock-release-container
		exit 0
		;;
	"cp "*)
		destination=$3
		cat >"$destination" <<'AGENT'
#!/bin/sh
case "${1:-}" in
	version) printf '%s v1alpha1\n' "${KPANEL_RELEASE_VERSION:?}" ;;
	healthcheck)
		[ -f "${KEJILION_AGENT_TOKEN_FILE:?}" ]
		[ "$(stat -c '%a' "$KEJILION_AGENT_TOKEN_FILE")" = 640 ]
		[ "$(tr -d '\r\n' <"$KEJILION_AGENT_TOKEN_FILE" | wc -c)" = 64 ]
		;;
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
		require_state
		case "${4:-}" in
			create) : >"$state/network" ;;
			up)
				if [ "${KPANEL_MOCK_BOOTSTRAP_MISSING:-0}" != 1 ]; then
					mkdir -p /home/docker/kpanel/data/panel
					printf '%s\n' 'test-bootstrap-token' \
						>/home/docker/kpanel/data/panel/bootstrap.token
					chmod 600 /home/docker/kpanel/data/panel/bootstrap.token
				fi
				;;
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
printf '%s|%s\n' "$#" "$*" >>"${KPANEL_MOCK_SYSTEMCTL_LOG:?}"
if [ "$1" = "--version" ]; then
	printf '%s\n' 'systemd 255 (mock)'
	exit 0
fi
case "$1" in
	link)
		ln -sf "$2" /etc/systemd/system/kejilion-agent.service
		exit 0
		;;
	daemon-reload|enable|start|stop|disable) exit 0 ;;
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
[ ! -e /bin/systemctl ] || {
	echo "disposable app-conf test image unexpectedly provides /bin/systemctl" >&2
	exit 1
}
ln -s "$FAKE_BIN/systemctl" /bin/systemctl

run_lifecycle() {
	local ipv4_address="198.51.100.25"

	systemctl() {
		local COMMAND="$1"
		local SERVICE_NAME="${2:-}"
		/bin/systemctl "$COMMAND" "$SERVICE_NAME"
	}
	docker_app_plus() {
		:
	}
	check_docker_app_ip() {
		:
	}

	# shellcheck source=/dev/null
	. "$PROJECT_DIR/packaging/kejilion-app/kpanel.conf"
	docker_port="18080"
	docker_app_install >"$TEST_DIR/install-output.txt"
	grep -Fx '首次初始化 Token：test-bootstrap-token' \
		"$TEST_DIR/install-output.txt" >/dev/null
	grep -Fx '请复制此 Token 完成管理员账户初始化；初始化成功后 Token 自动失效。' \
		"$TEST_DIR/install-output.txt" >/dev/null
	if grep -F '首次初始化 Token 文件：' "$TEST_DIR/install-output.txt" >/dev/null; then
		echo "install output still asks the user to read the token file" >&2
		return 1
	fi

	grep -F "image: docker.io/kjlion/kejilion-panel:$RELEASE_VERSION" \
		/home/docker/kpanel/docker-compose.yml >/dev/null
	grep -F -- '- "18080:8080"' /home/docker/kpanel/docker-compose.yml >/dev/null
	grep -Fx 'KPANEL_PUBLIC_URL=http://198.51.100.25:18080' \
		/home/docker/kpanel/.env >/dev/null
	grep -Fx 'KPANEL_TRUSTED_PROXY_CIDRS=127.0.0.0/8,::1/128,172.30.0.0/16' \
		/home/docker/kpanel/.env >/dev/null
	test "$(grep -c '^    networks:$' /home/docker/kpanel/docker-compose.yml)" = 1
	grep -F 'ExecStart=/home/docker/kpanel/bin/kejilion-agent' \
		/home/docker/kpanel/kejilion-agent.service >/dev/null
	grep -Fx 'CapabilityBoundingSet=CAP_SYS_ADMIN CAP_SYS_MODULE CAP_NET_ADMIN CAP_SYS_RESOURCE CAP_DAC_OVERRIDE' \
		/home/docker/kpanel/kejilion-agent.service >/dev/null
	grep -Fx 'AmbientCapabilities=CAP_SYS_ADMIN CAP_SYS_MODULE CAP_NET_ADMIN CAP_SYS_RESOURCE CAP_DAC_OVERRIDE' \
		/home/docker/kpanel/kejilion-agent.service >/dev/null
	grep -F -- '-/home/web/certs -/home/web/letsencrypt' \
		/home/docker/kpanel/kejilion-agent.service >/dev/null
	test -f /home/docker/kpanel/secrets/agent.token
	test "$(stat -c '%a' /home/docker/kpanel/secrets/agent.token)" = 640
	test "$(stat -c '%u:%g' /home/docker/kpanel/secrets/agent.token)" = 0:987
	test "$(tr -d '\r\n' </home/docker/kpanel/secrets/agent.token | wc -c)" = 64
	test -f /home/docker/kpanel/.managed-by-kejilion-app

	docker_app_update
	test "$(/home/docker/kpanel/bin/kejilion-agent version)" = "$RELEASE_VERSION v1alpha1"

	docker_app_uninstall
	[ ! -e /home/docker/kpanel ]
}

run_failed_install() {
	local ipv4_address="198.51.100.25"

	systemctl() {
		local COMMAND="$1"
		local SERVICE_NAME="${2:-}"
		/bin/systemctl "$COMMAND" "$SERVICE_NAME"
	}
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

run_missing_bootstrap_token() {
	local ipv4_address="198.51.100.25"

	systemctl() {
		local COMMAND="$1"
		local SERVICE_NAME="${2:-}"
		/bin/systemctl "$COMMAND" "$SERVICE_NAME"
	}
	docker_app_plus() {
		:
	}
	check_docker_app_ip() {
		:
	}

	# shellcheck source=/dev/null
	. "$PROJECT_DIR/packaging/kejilion-app/kpanel.conf"
	docker_port="18080"
	if KPANEL_MOCK_BOOTSTRAP_MISSING=1 docker_app_install \
		>"$TEST_DIR/missing-bootstrap-output.txt"; then
		echo "install without a bootstrap token unexpectedly succeeded" >&2
		return 1
	fi
	grep -Fx 'KPanel 初始化凭证读取失败，安装已停止。' \
		"$TEST_DIR/missing-bootstrap-output.txt" >/dev/null
	[ ! -e /home/docker/kpanel ]
	[ ! -e "$MOCK_STATE/network" ]
}

run_unmanaged_guard() {
	local manual_unit="/tmp/manual-kejilion-agent.service"

	systemctl() {
		local COMMAND="$1"
		local SERVICE_NAME="${2:-}"
		/bin/systemctl "$COMMAND" "$SERVICE_NAME"
	}
	docker_app_plus() {
		:
	}
	# shellcheck source=/dev/null
	. "$PROJECT_DIR/packaging/kejilion-app/kpanel.conf"
	mkdir -p /home/docker/kpanel
	: >/home/docker/kpanel/docker-compose.yml
	: >"$manual_unit"
	ln -sf "$manual_unit" /etc/systemd/system/kejilion-agent.service

	if docker_app_update || docker_app_uninstall; then
		echo "unmanaged KPanel instance was accepted" >&2
		return 1
	fi
	test -f /home/docker/kpanel/docker-compose.yml
	test "$(readlink -f /etc/systemd/system/kejilion-agent.service)" = "$manual_unit"
	rm -f /etc/systemd/system/kejilion-agent.service "$manual_unit"
	rm -rf /home/docker/kpanel
}

export PATH="$FAKE_BIN:$PATH"
export KPANEL_MOCK_STATE="$MOCK_STATE"
export KPANEL_MOCK_SYSTEMCTL_LOG="$TEST_DIR/systemctl.log"
run_lifecycle
grep -Fx '1|daemon-reload' "$KPANEL_MOCK_SYSTEMCTL_LOG" >/dev/null
grep -Fx '3|enable --now kejilion-agent.service' "$KPANEL_MOCK_SYSTEMCTL_LOG" >/dev/null
grep -Fx '3|disable --now kejilion-agent.service' "$KPANEL_MOCK_SYSTEMCTL_LOG" >/dev/null
if grep -F 'daemon-reload ' "$KPANEL_MOCK_SYSTEMCTL_LOG" >/dev/null; then
	echo "daemon-reload received the empty service argument from kejilion.sh's systemctl wrapper" >&2
	exit 1
fi
run_failed_install
run_missing_bootstrap_token
run_unmanaged_guard
printf '%s\n' "app_conf_lifecycle=pass"
