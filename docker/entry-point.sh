#!/bin/bash

# 定义变量
DMP_HOME="/root"
STEAM_DIR="$DMP_HOME/steamcmd"

cd $DMP_HOME || exit

# 定义 SIGTERM 信号处理函数
cleanup() {
    echo "Received SIGTERM, cleaning up..."
    # 发送停止信号给 dmp 进程
    if [[ -n "$DMP_PID" ]]; then
        kill "$DMP_PID"
        echo "Stopped dmp process with PID $DMP_PID"
    fi
    exit 0
}

install_steamcmd() {
	if [ ! -x "$STEAM_DIR/steamcmd.sh" ] || [ ! -x "$STEAM_DIR/linux32/steamcmd" ]; then
		local staging
		staging="$(mktemp -d)" || return 1
		if ! wget -q https://steamcdn-a.akamaihd.net/client/installer/steamcmd_linux.tar.gz -O "$staging/steamcmd.tar.gz"; then
			rm -rf "$staging"
			return 1
		fi
		# 先在临时目录解压并检查，避免失败下载留下看似可用的安装。
		if ! tar -zxf "$staging/steamcmd.tar.gz" -C "$staging" ||
			[ ! -x "$staging/steamcmd.sh" ] || [ ! -x "$staging/linux32/steamcmd" ]; then
			rm -rf "$staging"
			return 1
		fi
		if ! rm -f "$staging/steamcmd.tar.gz" || ! mkdir -p "$STEAM_DIR" || ! cp -a "$staging/." "$STEAM_DIR/"; then
			rm -rf "$staging"
			return 1
		fi
		rm -rf "$staging"
	fi
}

link_so() {
	local curl64 curl32
	case "$(uname -m)" in
		x86_64)
			curl64="/usr/lib/x86_64-linux-gnu/libcurl-gnutls.so.4"
			curl32="/usr/lib/i386-linux-gnu/libcurl-gnutls.so.4"
			;;
		aarch64|arm64)
			curl64="/usr/lib/aarch64-linux-gnu/libcurl-gnutls.so.4"
			curl32="/usr/lib/arm-linux-gnueabihf/libcurl-gnutls.so.4"
			;;
		*)
			return 0
			;;
	esac
	if [ -d "dst/bin64/lib64" ] && [ ! -e "dst/bin64/lib64/libcurl-gnutls.so.4" ] && [ ! -L "dst/bin64/lib64/libcurl-gnutls.so.4" ]; then
		ln -s "$curl64" dst/bin64/lib64/libcurl-gnutls.so.4 || return 1
	fi
	if [ -d "dst/bin/lib32" ] && [ ! -e "dst/bin/lib32/libcurl-gnutls.so.4" ] && [ ! -L "dst/bin/lib32/libcurl-gnutls.so.4" ]; then
		ln -s "$curl32" dst/bin/lib32/libcurl-gnutls.so.4 || return 1
	fi
}

log_runtime() {
	case "$(uname -m)" in
		x86_64)
			echo "Runtime architecture: amd64"
			echo "DMP architecture: amd64"
			echo "SteamCMD compatibility: ${DMP_STEAMCMD_COMPAT:-auto} (auto: native)"
			;;
		aarch64|arm64)
			echo "Runtime architecture: arm64"
			echo "DMP architecture: arm64"
			echo "Box64: $("${DMP_BOX64_PATH:-box64}" --version 2>&1 | head -n 1)"
			echo "Box86: $("${DMP_BOX86_PATH:-box86}" --version 2>&1 | head -n 1)"
			echo "SteamCMD compatibility: ${DMP_STEAMCMD_COMPAT:-auto} (auto: box86)"
			;;
		*)
			echo "Runtime architecture: $(uname -m)"
			;;
	esac
}

# 捕获 SIGTERM 信号
trap cleanup SIGTERM

log_runtime
install_steamcmd || exit 1
link_so || exit 1

# 构建启动命令
DMP_CMD="./dmp -bind $DMP_PORT -dbpath ./data -level ${LEVEL:-info}"

# 如果启用 TLS，追加证书和私钥参数
if [ "$TLS" = "true" ]; then
    TLS_CERT="${TLS_CERT:-/etc/ssl/dmp/fullchain.pem}"
    TLS_KEY="${TLS_KEY:-/etc/ssl/dmp/privkey.pem}"
    DMP_CMD="$DMP_CMD -cert $TLS_CERT -key $TLS_KEY"
    echo "TLS enabled, cert: $TLS_CERT, key: $TLS_KEY"
fi

# 启动 dmp 并获取其 PID
$DMP_CMD 2>&1 &
DMP_PID=$!  # 获取 dmp 进程的 PID

# 让脚本保持运行状态，直到收到信号
while true; do
    sleep 1
done
