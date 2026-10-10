#!/usr/bin/env bash
#
# myapi 一键安装脚本（Docker Compose）
#
# 用法：
#   curl -fsSL https://raw.githubusercontent.com/axzcnzxis/myapi/main/install.sh | bash
#
# 脚本会：获取源码 -> 随机生成数据库/Redis 凭据与会话密钥写入 .env
#         -> 用 docker-compose.yml 构建并启动 myapi + PostgreSQL + Redis
#
# 可用环境变量（写在命令前，例如 `MYAPI_PORT=8080 bash install.sh`）：
#   MYAPI_DIR         源码/安装目录，默认 $PWD/myapi
#   MYAPI_PORT        控制台对外端口，默认 32180
#   MYAPI_REF         分支或标签，默认 main
#   MYAPI_REPO        源码仓库，默认 https://github.com/axzcnzxis/myapi.git
#   MYAPI_BUILD_JOBS  Go 编译并发度，默认按内存自动选择（2~8）
#   MYAPI_MIN_MEM_MB  构建所需最低内存（RAM + Swap），默认 4096
#   MYAPI_MIN_DISK_MB 构建所需最低可用磁盘，默认 8192
#   MYAPI_FORCE=1     跳过内存/磁盘检查，强制安装
#
# 本脚本不写入任何内置账号或密码：凭据在安装时随机生成，管理员账号在
# 首次访问时通过初始化向导创建。重复执行是安全的：已存在的 .env 不会被覆盖。
set -euo pipefail

PORT="${MYAPI_PORT:-32180}"
INSTALL_DIR="${MYAPI_DIR:-$PWD/myapi}"
REPO_URL="${MYAPI_REPO:-https://github.com/axzcnzxis/myapi.git}"
REF="${MYAPI_REF:-main}"
TARBALL_URL="${MYAPI_TARBALL_URL:-https://github.com/axzcnzxis/myapi/archive/refs/heads/${REF}.tar.gz}"

log() { printf '[myapi] %s\n' "$*"; }
warn() { printf '[myapi] 警告：%s\n' "$*" >&2; }
die() { printf '[myapi] 错误：%s\n' "$*" >&2; exit 1; }

rand_hex() {
  if command -v openssl >/dev/null 2>&1; then
    openssl rand -hex 32
  else
    head -c 32 /dev/urandom | od -An -tx1 | tr -d ' \n'
  fi
}

# 1. 检查依赖
command -v docker >/dev/null 2>&1 || die "未检测到 docker，请先安装 Docker：https://docs.docker.com/engine/install/"

if docker compose version >/dev/null 2>&1; then
  COMPOSE=(docker compose)
elif command -v docker-compose >/dev/null 2>&1; then
  COMPOSE=(docker-compose)
else
  die "未检测到 Docker Compose（docker compose 或 docker-compose）"
fi

docker info >/dev/null 2>&1 || die "无法连接 Docker 守护进程，请确认 Docker 已启动，且当前用户有权限访问"

# 2. 资源预检
# 构建镜像需要同时跑前端打包与 Go 编译，内存不足的机器会因内存峰值超限而
# 卡死甚至 OOM（表现为宿主机失去响应）。这里在动手前先做最低限度的把关。
MIN_MEM_MB="${MYAPI_MIN_MEM_MB:-4096}"
MIN_DISK_MB="${MYAPI_MIN_DISK_MB:-8192}"
FORCE="${MYAPI_FORCE:-0}"
MEM_SUM_MB=0

meminfo_mb() { awk -v k="$1" '$1 == k":" { print int($2/1024) }' /proc/meminfo 2>/dev/null; }

if [ -r /proc/meminfo ]; then
  MEM_TOTAL="$(meminfo_mb MemTotal)"; SWAP_TOTAL="$(meminfo_mb SwapTotal)"; MEM_AVAIL="$(meminfo_mb MemAvailable)"
  MEM_TOTAL="${MEM_TOTAL:-0}"; SWAP_TOTAL="${SWAP_TOTAL:-0}"; MEM_AVAIL="${MEM_AVAIL:-0}"
  MEM_SUM_MB=$((MEM_TOTAL + SWAP_TOTAL))
  log "资源检测：内存 ${MEM_TOTAL} MiB + Swap ${SWAP_TOTAL} MiB（当前可用 ${MEM_AVAIL} MiB），CPU $(nproc 2>/dev/null || echo '?') 核"
  if [ "$MEM_SUM_MB" -lt "$MIN_MEM_MB" ] && [ "$FORCE" != "1" ]; then
    cat >&2 <<EOF
[myapi] 错误：可用内存不足，构建很可能卡死或触发 OOM。
        构建至少需要 ${MIN_MEM_MB} MiB（内存 + Swap），当前只有 ${MEM_SUM_MB} MiB。
        请任选一种方式后重试：
          1) 增加 Swap（示例增加 4 GiB，需要 root）：
               sudo fallocate -l 4G /swapfile && sudo chmod 600 /swapfile
               sudo mkswap /swapfile && sudo swapon /swapfile
               echo '/swapfile none swap sw 0 0' | sudo tee -a /etc/fstab
          2) 换用内存更大的机器构建，或改用单容器（SQLite）方式；
          3) 已知晓风险仍要继续：MYAPI_FORCE=1 bash install.sh
EOF
    exit 1
  fi
  if [ "$MEM_TOTAL" -lt "$MIN_MEM_MB" ]; then
    warn "物理内存仅 ${MEM_TOTAL} MiB，构建会大量使用 Swap，速度明显变慢；建议先加内存或 Swap。"
  fi
fi

DISK_PATH="$INSTALL_DIR"
while [ ! -d "$DISK_PATH" ] && [ "$DISK_PATH" != "/" ] && [ "$DISK_PATH" != "." ]; do
  DISK_PATH="$(dirname "$DISK_PATH")"
done
FREE_MB="$(df -Pk "$DISK_PATH" 2>/dev/null | awk 'NR==2 { print int($4/1024) }')"
if [ -n "${FREE_MB:-}" ] && [ "$FREE_MB" -lt "$MIN_DISK_MB" ] && [ "$FORCE" != "1" ]; then
  die "可用磁盘不足：$DISK_PATH 仅剩 ${FREE_MB} MiB，构建镜像与缓存建议至少 ${MIN_DISK_MB} MiB（可用 MYAPI_FORCE=1 跳过检查）"
fi

# 3. 获取源码
if [ -f "$INSTALL_DIR/docker-compose.yml" ]; then
  log "复用已有源码目录：$INSTALL_DIR"
else
  log "获取源码到：$INSTALL_DIR"
  mkdir -p "$INSTALL_DIR"
  if command -v git >/dev/null 2>&1; then
    git clone --depth 1 --branch "$REF" "$REPO_URL" "$INSTALL_DIR"
  else
    warn "未检测到 git，改用源码压缩包下载"
    tmp="$(mktemp -d)"
    trap 'rm -rf "$tmp"' EXIT
    if ! curl -fsSL "$TARBALL_URL" | tar -xz -C "$tmp"; then
      die "源码下载失败：$TARBALL_URL"
    fi
    src=""
    for d in "$tmp"/*/; do
      if [ -d "$d" ]; then src="${d%/}"; break; fi
    done
    [ -n "$src" ] || die "源码包内容异常：$TARBALL_URL"
    cp -a "$src/." "$INSTALL_DIR/"
    rm -rf "$tmp"
    trap - EXIT
  fi
fi

cd "$INSTALL_DIR"
[ -f docker-compose.yml ] || die "未找到 docker-compose.yml，源码获取不完整"

# 4. 生成 .env（凭据随机，不覆盖已有配置）
if [ -f .env ]; then
  log "已存在 .env，保留现有配置（如需改端口请编辑 $INSTALL_DIR/.env）"
else
  log "生成随机凭据并写入 .env"
  ( umask 077; cat > .env <<EOF
# 由 install.sh 生成，请勿提交到版本库
POSTGRES_USER=myapi
POSTGRES_DB=myapi
POSTGRES_PASSWORD=$(rand_hex)
REDIS_PASSWORD=$(rand_hex)
SESSION_SECRET=$(rand_hex)
MYAPI_PORT=$PORT
NODE_NAME=myapi-node-1
TZ=${TZ:-Asia/Shanghai}
EOF
  )
fi

# 5. 构建并启动
# 限制 Compose 层的构建/拉取并行度，避免与镜像构建争抢内存。
export COMPOSE_PARALLEL_LIMIT="${COMPOSE_PARALLEL_LIMIT:-1}"

# Go 编译并发度：内存充裕时按 CPU 核数放开（上限 8），否则保守取 2。
if [ -z "${MYAPI_BUILD_JOBS:-}" ]; then
  if [ "$MEM_SUM_MB" -ge 8192 ]; then
    JOBS="$(nproc 2>/dev/null || echo 2)"
    [ "$JOBS" -gt 8 ] && JOBS=8
  else
    JOBS=2
  fi
  export MYAPI_BUILD_JOBS="$JOBS"
fi

log "构建并发度：GO_BUILD_JOBS=${MYAPI_BUILD_JOBS}，Compose 并行上限=${COMPOSE_PARALLEL_LIMIT}"
log "构建并启动容器（首次构建需拉取依赖，耗时较长）..."
"${COMPOSE[@]}" up -d --build

# 6. 输出访问信息
HOST_IP="$(hostname -I 2>/dev/null | awk '{print $1}')"
[ -n "$HOST_IP" ] || HOST_IP="<服务器地址>"

log "安装完成"
printf '\n'
printf '  控制台地址：http://%s:%s\n' "$HOST_IP" "$PORT"
printf '  查看日志：  cd %s && %s logs -f myapi\n' "$INSTALL_DIR" "${COMPOSE[*]}"
printf '  首次访问会进入初始化向导，用于创建管理员账号（无内置账号密码）。\n\n'
