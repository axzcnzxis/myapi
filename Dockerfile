# 前端构建阶段。
#
# 用 Node 而不是 bun 执行 rsbuild：在部分宿主机上 bun 无法加载 @rspack/core 的
# 原生绑定，表现为 bun 以约 100% CPU 空转、不产生任何输出、也写不出 dist，
# 构建会永久挂起（bun 自身的 node 兼容层与 bun 运行时都会触发）。
# 因此 bun 只负责按 bun.lock 安装依赖，实际打包交由 Node 运行。
FROM node:22-bookworm-slim@sha256:c3de60bf2f9dd0ac6370e6117950ff62d6e339527e7472301c9c78a017978392 AS builder
# 直接复用 oven/bun 镜像中的 bun 二进制，无需额外安装步骤与网络依赖。
COPY --from=oven/bun:1.4.0@sha256:5ff609364c049b54eb0ff560ec96319729a972078ef2c755d758f0c6ef89c2d6 /usr/local/bin/bun /usr/local/bin/bun

WORKDIR /build/web
COPY web/package.json web/bun.lock ./
RUN bun install --frozen-lockfile
COPY ./web ./
COPY ./VERSION /build/VERSION
RUN DISABLE_ESLINT_PLUGIN='true' VITE_REACT_APP_VERSION=$(cat /build/VERSION) node node_modules/.bin/rsbuild build

FROM golang:1.26.1-alpine@sha256:2389ebfa5b7f43eeafbd6be0c3700cc46690ef842ad962f6c5bd6be49ed82039 AS builder2
ENV GO111MODULE=on CGO_ENABLED=0 GOWORK=off

ARG TARGETOS
ARG TARGETARCH
ENV GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH:-amd64}
ENV GOEXPERIMENT=greenteagc

# Go 编译并发度上限。BuildKit 默认按 CPU 核数并行编译，在小内存机器上
# （如 4C4G、无 Swap）内存峰值会超过物理内存，导致宿主机卡死 / OOM。
# 可用 `--build-arg GO_BUILD_JOBS=N` 覆盖（Compose 中对应 MYAPI_BUILD_JOBS）。
ARG GO_BUILD_JOBS=2
ENV GOFLAGS="-p=${GO_BUILD_JOBS}" GOMAXPROCS=${GO_BUILD_JOBS}

WORKDIR /build

ADD go.mod go.sum ./
# relaykit is a local submodule referenced via replace; its go.mod must be
# present for go mod download to resolve the main module graph.
ADD relaykit/go.mod ./relaykit/go.mod
RUN go mod download

# 串行化栅栏：显式依赖前端构建阶段（builder）的产物。
# BuildKit 会并行构建互不依赖的阶段，此前前端 rsbuild 与后端 COPY/编译同时进行，
# 在内存较小的机器上叠加出超过物理内存的峰值，导致宿主机卡死。提前引用 dist 后，
# 后续所有步骤都要等前端构建完成；go mod download 位于栅栏之前，缓存仍可独立复用。
# 注意：.dockerignore 已排除 /web/dist，因此下面的 `COPY . .` 不会覆盖这里拷入的产物。
COPY --from=builder /build/web/dist ./web/dist

COPY . .
RUN go build -ldflags "-s -w -X 'github.com/axzcnzxis/myapi/common.Version=$(cat VERSION)'" -o myapi

FROM debian:bookworm-slim@sha256:f06537653ac770703bc45b4b113475bd402f451e85223f0f2837acbf89ab020a

RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates tzdata libasan8 wget \
    && rm -rf /var/lib/apt/lists/* \
    && update-ca-certificates

COPY --from=builder2 /build/myapi /
COPY LICENSE NOTICE THIRD-PARTY-LICENSES.md /licenses/
EXPOSE 32180
WORKDIR /data
ENTRYPOINT ["/myapi"]
