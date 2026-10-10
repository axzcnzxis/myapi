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
#   MYAPI_DIR    源码/安装目录，默认 $PWD/myapi
#   MYAPI_PORT   控制台对外端口，默认 32180
#   MYAPI_REF    分支或标签，默认 main
#   MYAPI_REPO   源码仓库，默认 https://github.com/axzcnzxis/myapi.git
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

# 2. 获取源码
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

# 3. 生成 .env（凭据随机，不覆盖已有配置）
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

# 4. 构建并启动
log "构建并启动容器（首次构建需拉取依赖，耗时较长）..."
"${COMPOSE[@]}" up -d --build

# 5. 输出访问信息
HOST_IP="$(hostname -I 2>/dev/null | awk '{print $1}')"
[ -n "$HOST_IP" ] || HOST_IP="<服务器地址>"

log "安装完成"
printf '\n'
printf '  控制台地址：http://%s:%s\n' "$HOST_IP" "$PORT"
printf '  查看日志：  cd %s && %s logs -f myapi\n' "$INSTALL_DIR" "${COMPOSE[*]}"
printf '  首次访问会进入初始化向导，用于创建管理员账号（无内置账号密码）。\n\n'
