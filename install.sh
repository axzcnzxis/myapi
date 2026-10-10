#!/usr/bin/env bash
#
# myapi 一键安装脚本（Docker Compose）
#
# 用法：
#   curl -fsSL https://raw.githubusercontent.com/axzcnzxis/myapi/main/install.sh | bash
#
# 脚本会：获取源码 -> 随机生成数据库/Redis 凭据与会话密钥写入 .env
#         -> 拉取预构建镜像并启动 myapi + PostgreSQL + Redis（不在本机构建）
#
# 可用环境变量（写在命令前，例如 `MYAPI_PORT=8080 bash install.sh`）：
#   MYAPI_DIR     源码/安装目录，默认 $PWD/myapi
#   MYAPI_PORT    控制台对外端口，默认 32180；仅在首次生成 .env 时写入，
#                 之后以 .env 里的 MYAPI_PORT 为准（脚本据此打印访问地址）
#   MYAPI_IMAGE   使用的 myapi 镜像，默认 ghcr.io/axzcnzxis/myapi:latest；
#                 同样仅在首次生成 .env 时写入，之后以 .env 里的值为准
#   MYAPI_REF     分支或标签，默认 main
#   MYAPI_REPO    源码仓库，默认 https://github.com/axzcnzxis/myapi.git
#   MYAPI_TARBALL_URL      未安装 git 时的源码压缩包地址（默认按分支下载）
#   MYAPI_TARBALL_URL_TAG  未安装 git 时的标签压缩包地址（默认按标签下载）
#
# 本脚本不写入任何内置账号或密码：凭据在安装时随机生成，管理员账号在
# 首次访问时通过初始化向导创建。重复执行是安全的：已存在的 .env 不会被覆盖。
set -euo pipefail

PORT="${MYAPI_PORT:-32180}"
IMAGE="${MYAPI_IMAGE:-ghcr.io/axzcnzxis/myapi:latest}"
INSTALL_DIR="${MYAPI_DIR:-$PWD/myapi}"
REPO_URL="${MYAPI_REPO:-https://github.com/axzcnzxis/myapi.git}"
REF="${MYAPI_REF:-main}"
TARBALL_URL="${MYAPI_TARBALL_URL:-https://github.com/axzcnzxis/myapi/archive/refs/heads/${REF}.tar.gz}"
TARBALL_URL_TAG="${MYAPI_TARBALL_URL_TAG:-https://github.com/axzcnzxis/myapi/archive/refs/tags/${REF}.tar.gz}"

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
  # 目录存在但没有 docker-compose.yml，通常是上次下载/克隆中断的残留：
  # 此时 git clone 会因目标目录非空而失败，重复执行也就不再安全。
  if [ -d "$INSTALL_DIR" ] && [ -n "$(ls -A "$INSTALL_DIR" 2>/dev/null)" ]; then
    die "$INSTALL_DIR 已存在且非空，但里面没有 docker-compose.yml（可能是上次安装中断的残留）。请先删除该目录，或用 MYAPI_DIR 指定其它目录后重试"
  fi
  mkdir -p "$INSTALL_DIR"
  if command -v git >/dev/null 2>&1; then
    git clone --depth 1 --branch "$REF" "$REPO_URL" "$INSTALL_DIR"
  else
    warn "未检测到 git，改用源码压缩包下载"
    tmp="$(mktemp -d)"
    trap 'rm -rf "$tmp"' EXIT
    if ! curl -fsSL "$TARBALL_URL" | tar -xz -C "$tmp"; then
      log "按分支 $REF 下载失败，尝试同名标签"
      rm -rf "${tmp:?}/"*
      if ! curl -fsSL "$TARBALL_URL_TAG" | tar -xz -C "$tmp"; then
        die "源码下载失败：分支与标签 $REF 均无法获取"
      fi
    fi
    src=""
    for d in "$tmp"/*/; do
      if [ -d "$d" ]; then src="${d%/}"; break; fi
    done
    [ -n "$src" ] || die "源码包内容异常：$REF"
    cp -a "$src/." "$INSTALL_DIR/"
    rm -rf "$tmp"
    trap - EXIT
  fi
fi

cd "$INSTALL_DIR"
[ -f docker-compose.yml ] || die "未找到 docker-compose.yml，源码获取不完整"

# 3. 生成 .env（凭据随机，不覆盖已有配置）
if [ -f .env ]; then
  log "已存在 .env，保留现有配置（如需改端口或镜像请编辑 $INSTALL_DIR/.env）"
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
MYAPI_IMAGE=$IMAGE
NODE_NAME=myapi-node-1
TZ=${TZ:-Asia/Shanghai}
EOF
  )
fi

# 生效端口以 .env 为准：docker compose 只读 .env，脚本不会把 MYAPI_PORT 导出给 compose。
# 因此复用已有 .env（或 .env 中的值与命令行不一致）时，必须以 .env 的值打印访问地址。
EFFECTIVE_PORT="$(sed -n 's/^[[:space:]]*MYAPI_PORT[[:space:]]*=[[:space:]]*//p' .env | tail -n1 | tr -d "\"' \t\r")"
if [ -n "$EFFECTIVE_PORT" ]; then
  if [ -n "${MYAPI_PORT:-}" ] && [ "$MYAPI_PORT" != "$EFFECTIVE_PORT" ]; then
    warn "命令行 MYAPI_PORT=$MYAPI_PORT 与 .env 中的 MYAPI_PORT=$EFFECTIVE_PORT 不一致，实际以 .env 为准"
  fi
  PORT="$EFFECTIVE_PORT"
fi

# 镜像同理：compose 只读 .env 与 shell 环境，脚本不会导出 MYAPI_IMAGE，
# 因此复用已有 .env 时以 .env 中的镜像地址为准。
EFFECTIVE_IMAGE="$(sed -n 's/^[[:space:]]*MYAPI_IMAGE[[:space:]]*=[[:space:]]*//p' .env | tail -n1 | tr -d "\"' \t\r")"
if [ -n "$EFFECTIVE_IMAGE" ]; then
  if [ -n "${MYAPI_IMAGE:-}" ] && [ "$MYAPI_IMAGE" != "$EFFECTIVE_IMAGE" ]; then
    warn "命令行 MYAPI_IMAGE=$MYAPI_IMAGE 与 .env 中的 MYAPI_IMAGE=$EFFECTIVE_IMAGE 不一致，实际以 .env 为准"
  fi
  IMAGE="$EFFECTIVE_IMAGE"
fi

# 4. 拉取预构建镜像并启动
# 不在本机构建：构建阶段要下载前端/后端依赖并编译，小内存机器（如 2G）容易被拖死。
log "拉取镜像：$IMAGE"
if ! "${COMPOSE[@]}" pull; then
  warn "镜像拉取失败：请检查网络与镜像地址（当前为 $IMAGE，可用 MYAPI_IMAGE=<镜像地址> 指定其它来源）"
  if grep -Eq '^[[:space:]]*build:' docker-compose.yml; then
    warn "当前目录的 docker-compose.yml 是含 build: 的旧版配置；本脚本不会据此在本机构建，请先更新源码目录（cd $INSTALL_DIR && git pull）后重跑"
  fi
  die "安装中止：镜像未就绪，未启动任何容器"
fi

log "启动容器..."
"${COMPOSE[@]}" up -d --no-build

# 5. 输出访问信息
HOST_IP="$(hostname -I 2>/dev/null | awk '{print $1}')"
[ -n "$HOST_IP" ] || HOST_IP="<服务器地址>"

log "安装完成"
printf '\n'
printf '  控制台地址：http://%s:%s\n' "$HOST_IP" "$PORT"
printf '  查看日志：  cd %s && %s logs -f myapi\n' "$INSTALL_DIR" "${COMPOSE[*]}"
printf '  首次访问会进入初始化向导，用于创建管理员账号（无内置账号密码）。\n\n'
