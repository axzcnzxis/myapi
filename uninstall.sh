#!/usr/bin/env bash
#
# myapi 一键卸载脚本（Docker Compose）
#
# 用法：
#   # 默认：停止并移除容器、网络与本地镜像，保留数据
#   curl -fsSL https://raw.githubusercontent.com/axzcnzxis/myapi/main/uninstall.sh | bash
#
#   # 彻底清理：连同数据卷与 data、logs 目录一并删除（不可恢复）
#   curl -fsSL https://raw.githubusercontent.com/axzcnzxis/myapi/main/uninstall.sh | bash -s -- --purge
#
# 可用环境变量（写在命令前）：
#   MYAPI_DIR    安装目录，默认 $PWD（若不存在 docker-compose.yml 则尝试 $PWD/myapi）
#   MYAPI_PURGE  设为 1 等价于 --purge
#
# 安全边界：只操作本项目的 Compose 工程（按 docker-compose.yml 中的
# `container_name: myapi` 校验），不执行 docker system prune，也不影响其它容器。
set -euo pipefail

log() { printf '[myapi] %s\n' "$*"; }
warn() { printf '[myapi] 警告：%s\n' "$*" >&2; }
die() { printf '[myapi] 错误：%s\n' "$*" >&2; exit 1; }

usage() {
  cat <<'EOF'
用法：uninstall.sh [--purge]
  （默认）停止并移除 myapi 容器、网络与本地镜像，保留数据库卷与数据目录
  --purge  额外删除 pg_data 卷与 data、logs 目录（数据不可恢复）
EOF
}

PURGE="${MYAPI_PURGE:-0}"
for arg in "$@"; do
  case "$arg" in
    --purge) PURGE=1 ;;
    -h|--help) usage; exit 0 ;;
    *) die "未知参数：$arg（可用：--purge、--help）" ;;
  esac
done

# 1. 定位安装目录
if [ -n "${MYAPI_DIR:-}" ]; then
  INSTALL_DIR="$MYAPI_DIR"
elif [ -f "$PWD/docker-compose.yml" ]; then
  INSTALL_DIR="$PWD"
elif [ -d "$PWD/myapi" ]; then
  INSTALL_DIR="$PWD/myapi"
else
  die "未找到安装目录，请用 MYAPI_DIR=/path/to/myapi 指定"
fi

cd "$INSTALL_DIR"
[ -f docker-compose.yml ] || die "$INSTALL_DIR 下没有 docker-compose.yml，请确认安装目录是否正确"

# 2. 校验这是 myapi 的 Compose 工程，避免误删其它项目
if ! grep -Eq '^[[:space:]]*container_name:[[:space:]]*myapi[[:space:]]*$' docker-compose.yml; then
  die "$INSTALL_DIR/docker-compose.yml 未包含 container_name: myapi，为安全起见已中止（请用 MYAPI_DIR 指向正确的安装目录）"
fi

# 3. 检查 docker / compose
command -v docker >/dev/null 2>&1 || die "未检测到 docker"
if docker compose version >/dev/null 2>&1; then
  COMPOSE=(docker compose)
elif command -v docker-compose >/dev/null 2>&1; then
  COMPOSE=(docker-compose)
else
  die "未检测到 Docker Compose（docker compose 或 docker-compose）"
fi
docker info >/dev/null 2>&1 || die "无法连接 Docker 守护进程，请确认 Docker 已启动，且当前用户有权限访问"

# 4. 停止并移除容器与网络
if [ "$PURGE" = "1" ]; then
  log "停止并移除 myapi 容器、网络与数据卷：$INSTALL_DIR"
  "${COMPOSE[@]}" down -v --remove-orphans
else
  log "停止并移除 myapi 容器与网络（保留数据）：$INSTALL_DIR"
  "${COMPOSE[@]}" down --remove-orphans
fi

# 5. 移除本地镜像（仅 myapi:latest，不做全局清理）
if docker image inspect myapi:latest >/dev/null 2>&1; then
  log "移除本地镜像 myapi:latest"
  docker image rm myapi:latest >/dev/null || warn "镜像 myapi:latest 移除失败（可能仍被其它容器引用）"
else
  log "未发现本地镜像 myapi:latest，跳过"
fi

# 6. --purge 时删除数据与日志目录
if [ "$PURGE" = "1" ]; then
  log "删除数据目录 data/ 与日志目录 logs/"
  rm -rf ./data ./logs
fi

log "卸载完成"
printf '\n'
if [ "$PURGE" = "1" ]; then
  printf '  已删除：容器、网络、镜像、pg_data 卷、./data、./logs\n'
else
  printf '  已保留：pg_data 卷、./data、./logs（需要一并删除请加 --purge）\n'
fi
printf '  源码目录未被删除：%s（确认无误后可自行删除）\n\n' "$INSTALL_DIR"
