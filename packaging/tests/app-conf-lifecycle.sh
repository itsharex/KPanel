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
	rm -f /root/kejilion.sh
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
	"compose version"|"pull docker.io/kjlion/kejilion-panel:latest")
		exit 0
		;;
	"ps -a")
		exit 0
		;;
	"network inspect")
		require_state
		if [ "${3:-}" = "--format" ]; then
			case "${5:-}" in
				kejilion-panel-internal) printf '%s\n' '172.30.0.0/16' ;;
				kejilion-panel-egress) printf '%s\n' '172.31.0.1' ;;
				*) exit 2 ;;
			esac
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
	"container create")
		require_state
		: >"$state/network-preflight"
		printf '%s\n' mock-network-preflight
		exit 0
		;;
	"container start")
		require_state
		[ -f "$state/network-preflight" ]
		if [ "${KPANEL_MOCK_PORT_PUBLISH_FAIL:-0}" = 1 ]; then
			echo "simulated missing DOCKER chain" >&2
			exit 1
		fi
		exit 0
		;;
	"container rm")
		require_state
		rm -f "$state/network-preflight"
		exit 0
		;;
	"cp "*)
		destination=$3
		case "$2" in
			*:/release/VERSION)
				printf '%s\n' \
					"${KPANEL_MOCK_RELEASE_FILE_VERSION:-${KPANEL_RELEASE_VERSION:?}}" \
					>"$destination"
				exit 0
				;;
			*:/release/kejilion.sh)
				cat >"$destination" <<'SCRIPT'
#!/usr/bin/env bash
canshu="default"
permission_granted="false"
ENABLE_STATS="true"
KJ_DNS_NONINTERACTIVE=1
KJ_APP_NONINTERACTIVE=1
KJ_WEB_NONINTERACTIVE=1
KJ_WEB_INTERACTIVE=1
KJ_TEST_NONINTERACTIVE=1
SCRIPT
				chmod 700 "$destination"
				exit 0
				;;
		esac
		cat >"$destination" <<'AGENT'
#!/bin/sh
case "${1:-}" in
	version)
		printf '%s v1alpha1\n' \
			"${KPANEL_MOCK_AGENT_VERSION:-${KPANEL_RELEASE_VERSION:?}}"
		;;
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
	"image inspect")
		case "$4" in
			*org.opencontainers.image.version*)
				printf '%s\n' \
					"${KPANEL_MOCK_IMAGE_VERSION:-${KPANEL_RELEASE_VERSION:?}}"
				;;
			*org.opencontainers.image.revision*)
				printf '%s\n' \
					"${KPANEL_MOCK_IMAGE_REVISION:-2222222222222222222222222222222222222222}"
				;;
			*io.kejilion.script.revision*)
				printf '%s\n' \
					"${KPANEL_MOCK_SCRIPT_REVISION:-4444444444444444444444444444444444444444}"
				;;
			*io.kejilion.script.sha256*)
				printf '%s\n' \
					"${KPANEL_MOCK_SCRIPT_SHA256:-1111111111111111111111111111111111111111111111111111111111111111}"
				;;
			*) exit 2 ;;
		esac
		exit 0
		;;
	"image tag")
		require_state
		printf '%s|%s\n' "$3" "$4" >"$state/image-tag"
		: >"$state/rollback-tagged"
		exit 0
		;;
	"rm "*)
		exit 0
		;;
	"inspect --format")
		case "$3" in
			*"{{.Image}}"*) printf '%s\n' \
				'sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa' ;;
			*PortBindings*) printf '%s\n' "${KPANEL_MOCK_CURRENT_PORT:-18080}" ;;
			*NetworkSettings*) printf '%s\n' 1 ;;
			*)
				if [ "${KPANEL_MOCK_HEALTH_FAIL:-0}" = 1 ] ||
					{ [ "${KPANEL_MOCK_UPDATE_HEALTH_FAIL:-0}" = 1 ] &&
						[ ! -f "$state/rollback-tagged" ]; }; then
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

cat >"$FAKE_BIN/sha256sum" <<'EOF'
#!/bin/sh
printf '%s  %s\n' \
	"${KPANEL_MOCK_SCRIPT_SHA256_ACTUAL:-1111111111111111111111111111111111111111111111111111111111111111}" \
	"$1"
EOF
chmod 755 "$FAKE_BIN"/*
[ ! -e /bin/systemctl ] || {
	echo "disposable app-conf test image unexpectedly provides /bin/systemctl" >&2
	exit 1
}
ln -s "$FAKE_BIN/systemctl" /bin/systemctl

run_lifecycle() {
	local ipv4_address="198.51.100.25"

	cat >/root/kejilion.sh <<'EOF'
#!/usr/bin/env bash
canshu="CN"
permission_granted="true"
ENABLE_STATS="false"
EOF
	chmod 700 /root/kejilion.sh

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
	grep -F "image: docker.io/kjlion/kejilion-panel:latest" \
		/home/docker/kpanel/docker-compose.yml >/dev/null
	grep -F -- '- "18080:8080"' /home/docker/kpanel/docker-compose.yml >/dev/null
	grep -Fx 'KPANEL_PUBLIC_URL=http://198.51.100.25:18080' \
		/home/docker/kpanel/.env >/dev/null
	grep -Fx 'KPANEL_SECURE_COOKIE=false' \
		/home/docker/kpanel/.env >/dev/null
	grep -Fx 'KPANEL_ALLOW_IP_HOSTS=true' \
		/home/docker/kpanel/.env >/dev/null
	grep -F 'KEJILION_PANEL_SECURE_COOKIE: ${KPANEL_SECURE_COOKIE:-false}' \
		/home/docker/kpanel/docker-compose.yml >/dev/null
	grep -F 'KEJILION_PANEL_ALLOW_IP_HOSTS: ${KPANEL_ALLOW_IP_HOSTS:-true}' \
		/home/docker/kpanel/docker-compose.yml >/dev/null
	grep -Fx 'KPANEL_TRUSTED_PROXY_CIDRS=127.0.0.0/8,::1/128,172.30.0.0/16,172.31.0.1/32' \
		/home/docker/kpanel/.env >/dev/null
	grep -F 'KEJILION_PANEL_CLUSTER_PRIVATE_CIDRS: ${KPANEL_CLUSTER_PRIVATE_CIDRS:-}' \
		/home/docker/kpanel/docker-compose.yml >/dev/null
	test "$(grep -c '^    networks:$' /home/docker/kpanel/docker-compose.yml)" = 1
	grep -Fx '      - kpanel-internal' /home/docker/kpanel/docker-compose.yml >/dev/null
	grep -Fx '      - kpanel-egress' /home/docker/kpanel/docker-compose.yml >/dev/null
	grep -Fx '    internal: true' /home/docker/kpanel/docker-compose.yml >/dev/null
	grep -Fx '    name: kejilion-panel-internal' /home/docker/kpanel/docker-compose.yml >/dev/null
	grep -Fx '    name: kejilion-panel-egress' /home/docker/kpanel/docker-compose.yml >/dev/null
	grep -F 'host.docker.internal:host-gateway' /home/docker/kpanel/docker-compose.yml >/dev/null
	grep -F 'ExecStart=/home/docker/kpanel/bin/kejilion-agent' \
		/home/docker/kpanel/kejilion-agent.service >/dev/null
	grep -Fx 'CapabilityBoundingSet=CAP_SYS_ADMIN CAP_SYS_MODULE CAP_NET_ADMIN CAP_SYS_RESOURCE CAP_DAC_OVERRIDE CAP_CHOWN CAP_LINUX_IMMUTABLE CAP_SYS_PTRACE' \
		/home/docker/kpanel/kejilion-agent.service >/dev/null
	grep -Fx 'AmbientCapabilities=CAP_SYS_ADMIN CAP_SYS_MODULE CAP_NET_ADMIN CAP_SYS_RESOURCE CAP_DAC_OVERRIDE CAP_CHOWN CAP_LINUX_IMMUTABLE CAP_SYS_PTRACE' \
		/home/docker/kpanel/kejilion-agent.service >/dev/null
	grep -Fx 'RestrictAddressFamilies=AF_UNIX AF_INET AF_INET6 AF_NETLINK' \
		/home/docker/kpanel/kejilion-agent.service >/dev/null
	grep -Fx 'ProtectHome=false' \
		/home/docker/kpanel/kejilion-agent.service >/dev/null
	grep -Fx 'ProtectSystem=false' \
		/home/docker/kpanel/kejilion-agent.service >/dev/null
	grep -F 'kpanel_report_failed_install' \
		"$PROJECT_DIR/packaging/kejilion-app/kpanel.conf" >/dev/null
	grep -F 'docker compose --env-file .env up -d --force-recreate --remove-orphans' \
		"$PROJECT_DIR/packaging/kejilion-app/kpanel.conf" >/dev/null
	if grep -F '请运行：systemctl status kejilion-agent' \
		"$PROJECT_DIR/packaging/kejilion-app/kpanel.conf" >/dev/null; then
		echo "KPanel installer still points users at a unit removed by cleanup" >&2
		exit 1
	fi
	if grep -q '^ReadWritePaths=' /home/docker/kpanel/kejilion-agent.service; then
		echo "KPanel app unit relies on ineffective root write exceptions" >&2
		exit 1
	fi
	grep -Fx 'ReadOnlyPaths=/home/docker/kpanel/data/panel' \
		/home/docker/kpanel/kejilion-agent.service >/dev/null
	test -f /home/docker/kpanel/secrets/agent.token
	test "$(stat -c '%a' /home/docker/kpanel/secrets/agent.token)" = 640
	test "$(stat -c '%u:%g' /home/docker/kpanel/secrets/agent.token)" = 0:987
	test "$(tr -d '\r\n' </home/docker/kpanel/secrets/agent.token | wc -c)" = 64
	test -f /home/docker/kpanel/.managed-by-kejilion-app
	test -x /home/docker/kpanel/bin/kejilion.sh
	test "$(stat -c '%a' /home/docker/kpanel/bin/kejilion.sh)" = 700
	grep -Fx 'permission_granted="true"' /home/docker/kpanel/bin/kejilion.sh >/dev/null
	grep -Fx 'ENABLE_STATS="false"' /home/docker/kpanel/bin/kejilion.sh >/dev/null
	grep -Fx 'canshu="CN"' /home/docker/kpanel/bin/kejilion.sh >/dev/null

	local systemctl_lines_before=""
	systemctl_lines_before="$(wc -l <"$KPANEL_MOCK_SYSTEMCTL_LOG")"
	if KPANEL_MOCK_PORT_PUBLISH_FAIL=1 docker_app_update \
		>"$TEST_DIR/network-preflight-output.txt" 2>&1; then
		echo "KPanel update ignored a failed Docker port-publish preflight" >&2
		return 1
	fi
	grep -F 'Docker 端口映射预检失败' \
		"$TEST_DIR/network-preflight-output.txt" >/dev/null
	test "$(wc -l <"$KPANEL_MOCK_SYSTEMCTL_LOG")" = "$systemctl_lines_before"
	test ! -e "$MOCK_STATE/network-preflight"
	test ! -e "$MOCK_STATE/rollback-tagged"
	test "$(/home/docker/kpanel/bin/kejilion-agent version)" = "$RELEASE_VERSION v1alpha1"

	rm -f "$MOCK_STATE/rollback-tagged" "$MOCK_STATE/image-tag"
	if KPANEL_MOCK_AGENT_VERSION=9.9.9 docker_app_update; then
		echo "KPanel update accepted a mismatched Agent" >&2
		return 1
	fi
	grep -Fx \
		'sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa|docker.io/kjlion/kejilion-panel:latest' \
		"$MOCK_STATE/image-tag" >/dev/null
	test "$(/home/docker/kpanel/bin/kejilion-agent version)" = "$RELEASE_VERSION v1alpha1"

	sed -i 's#^KPANEL_TRUSTED_PROXY_CIDRS=.*#KPANEL_TRUSTED_PROXY_CIDRS=127.0.0.0/8,::1/128,172.20.0.0/16#' \
		/home/docker/kpanel/.env
	sed -i '/^KPANEL_SECURE_COOKIE=/d' /home/docker/kpanel/.env
	docker_port="8080"
	docker_app_update
	test "$(/home/docker/kpanel/bin/kejilion-agent version)" = "$RELEASE_VERSION v1alpha1"
	grep -Fx 'permission_granted="true"' /home/docker/kpanel/bin/kejilion.sh >/dev/null
	grep -F -- '- "18080:8080"' /home/docker/kpanel/docker-compose.yml >/dev/null
	grep -Fx 'KPANEL_SECURE_COOKIE=false' \
		/home/docker/kpanel/.env >/dev/null
	grep -Fx 'KPANEL_TRUSTED_PROXY_CIDRS=127.0.0.0/8,::1/128,172.30.0.0/16,172.31.0.1/32' \
		/home/docker/kpanel/.env >/dev/null
	test ! -e /home/docker/kpanel/.env.rollback

	sed -i 's#^KPANEL_PUBLIC_URL=.*#KPANEL_PUBLIC_URL=https://panel.example.com#' \
		/home/docker/kpanel/.env
	sed -i '/^KPANEL_SECURE_COOKIE=/d' /home/docker/kpanel/.env
	docker_app_update
	grep -Fx 'KPANEL_SECURE_COOKIE=true' /home/docker/kpanel/.env >/dev/null
	grep -Fx 'KPANEL_PUBLIC_URL=https://panel.example.com' /home/docker/kpanel/.env >/dev/null
	cp -p /home/docker/kpanel/.env "$TEST_DIR/https-env"

	rm -f "$MOCK_STATE/rollback-tagged" "$MOCK_STATE/image-tag"
	if KPANEL_MOCK_UPDATE_HEALTH_FAIL=1 docker_app_update; then
		echo "failed KPanel update unexpectedly succeeded" >&2
		return 1
	fi
	grep -Fx \
		'sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa|docker.io/kjlion/kejilion-panel:latest' \
		"$MOCK_STATE/image-tag" >/dev/null
	grep -F -- '- "18080:8080"' /home/docker/kpanel/docker-compose.yml >/dev/null
	test "$(/home/docker/kpanel/bin/kejilion-agent version)" = "$RELEASE_VERSION v1alpha1"
	cmp -s "$TEST_DIR/https-env" /home/docker/kpanel/.env
	grep -Fx 'KPANEL_SECURE_COOKIE=true' /home/docker/kpanel/.env >/dev/null
	test ! -e /home/docker/kpanel/.env.rollback

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

run_release_contract_guards() {
	local candidate_agent="$TEST_DIR/release-contract-agent"
	local candidate_script="$TEST_DIR/release-contract-script"

	# shellcheck source=/dev/null
	. "$PROJECT_DIR/packaging/kejilion-app/kpanel.conf"

	if grep -Eq '"[0-9]+\.[0-9]+\.[0-9]+ v1alpha1"' \
		"$PROJECT_DIR/packaging/kejilion-app/kpanel.conf"; then
		echo "KPanel app config still hard-codes an Agent release version" >&2
		return 1
	fi
	if grep -Eq 'local script_sha256="[0-9a-f]{64}"' \
		"$PROJECT_DIR/packaging/kejilion-app/kpanel.conf"; then
		echo "KPanel app config still hard-codes a script release digest" >&2
		return 1
	fi

	if (
		export KPANEL_MOCK_IMAGE_VERSION=9.9.9
		kpanel_extract_release mock-image "$candidate_agent" "$candidate_script"
	); then
		echo "release VERSION mismatch was accepted" >&2
		return 1
	fi
	if (
		export KPANEL_MOCK_AGENT_VERSION=9.9.9
		kpanel_extract_release mock-image "$candidate_agent" "$candidate_script"
	); then
		echo "Agent version mismatch was accepted" >&2
		return 1
	fi
	if (
		export KPANEL_MOCK_SCRIPT_SHA256=3333333333333333333333333333333333333333333333333333333333333333
		kpanel_extract_release mock-image "$candidate_agent" "$candidate_script"
	); then
		echo "script digest mismatch was accepted" >&2
		return 1
	fi
	if (
		export KPANEL_MOCK_IMAGE_REVISION=invalid
		kpanel_extract_release mock-image "$candidate_agent" "$candidate_script"
	); then
		echo "invalid image revision was accepted" >&2
		return 1
	fi
	if (
		export KPANEL_MOCK_SCRIPT_REVISION=invalid
		kpanel_extract_release mock-image "$candidate_agent" "$candidate_script"
	); then
		echo "invalid script revision was accepted" >&2
		return 1
	fi
	[ ! -e "$candidate_agent" ]
	[ ! -e "$candidate_script" ]
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
run_release_contract_guards
printf '%s\n' "app_conf_lifecycle=pass"
