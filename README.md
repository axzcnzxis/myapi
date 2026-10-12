<div align="center">

![myapi](/web/public/logo.png)

# myapi

**基于 new-api 的自托管 AI 网关（二次开发版本）**

<p align="center">
  <strong>简体中文</strong> |
  <a href="./README.en.md">English</a> |
  <a href="./README.zh_TW.md">繁體中文</a> |
  <a href="./README.fr.md">Français</a> |
  <a href="./README.ja.md">日本語</a>
</p>

<p align="center">
  <a href="https://github.com/axzcnzxis/myapi/blob/main/LICENSE">
    <img src="https://img.shields.io/github/license/axzcnzxis/myapi?color=brightgreen" alt="license">
  </a><!--
  --><a href="https://github.com/axzcnzxis/myapi/releases">
    <img src="https://img.shields.io/github/v/release/axzcnzxis/myapi?color=brightgreen&include_prereleases" alt="release">
  </a>
</p>

</div>

---

## 📌 这是什么

**myapi** 是 [new-api](https://github.com/QuantumNous/new-api)（AI 网关 / 多模型管理与计费平台）的**二次开发分支**。它保留了上游的全部核心能力（渠道与模型管理、令牌、计费、订阅、OAuth/OIDC、Passkey、任务插件、多语言控制台等），并在其上叠加了一组面向运营的功能与若干修复。

| 项 | 值 |
| --- | --- |
| 本仓库 | <https://github.com/axzcnzxis/myapi> |
| 上游项目 | <https://github.com/QuantumNous/new-api> |
| 上游基线 | `v1.0.0-rc.41`（git 提交 `973cf8ef4`） |
| 本仓库版本号 | 见 [`VERSION`](./VERSION)（当前 `v0.0.10`） |
| 许可 | AGPL-3.0（含 Section 7 附加条款，见 [`LICENSE`](./LICENSE) / [`NOTICE`](./NOTICE)） |

> 本仓库是**衍生作品**，不是 new-api 官方仓库。上游与本仓库的文档、Issue、Release 互不相通，请勿把本仓库的问题反馈到上游。

---

## 🔀 与原版 new-api 的差异

以下差异以**上游 `v1.0.0-rc.41`（提交 `973cf8ef4`）**为基准，按功能点列出。差异清单由实际代码对比得出，涉及文件均已列出。

### 1. 品牌与仓库标识（重命名）

| 项 | 上游 | 本仓库 |
| --- | --- | --- |
| Go 模块路径 | `github.com/QuantumNous/new-api` | `github.com/axzcnzxis/myapi` |
| 子模块 | `.../new-api/relaykit` | `github.com/axzcnzxis/myapi/relaykit` |
| 默认系统名 | `New API` | `myapi` |
| 构建产物 / 容器入口 | `/new-api` | `/myapi` |
| Docker 镜像 | `calciumion/new-api` | `ghcr.io/axzcnzxis/myapi`（预构建，也可从源码构建，见下） |
| 数据库名（Compose 默认） | `new-api` | `myapi` |

涉及文件：`go.mod`、`relaykit/go.mod`、全仓库 Go import、`Dockerfile`、`docker-compose.yml`、`common/constants.go`（`SystemName`）、`common/init.go`（`--help` 文案）、`main.go`（启动日志）、`web/index.html`、`web/src/lib/constants.ts`、`web/src/components/layout/components/system-brand.tsx`、`web/src/components/layout/components/footer.tsx`、`web/src/assets/logo.tsx`、`web/src/i18n/locales/*.json`。

> 版权声明（`Copyright (C) 2023-2026 QuantumNous`）、[`LICENSE`](./LICENSE)、[`NOTICE`](./NOTICE)、第三方插件作者署名等**上游归属信息保持不变**。

### 2. 平台公告系统（新增）

一套完整的平台公告管理：草稿 / 定时发布 / 已发布 / 已取消 / 已过期，支持「通知中心」与「弹窗」两种投递方式，可配置每日展示次数与展示时段。

- 新增数据表 `announcements`（`model/announcement.go`，随 `model/main.go` 的 AutoMigrate 增量创建）。
- 公开接口：`GET /api/announcements/current`、`GET /api/announcements/popup`。
- 管理接口（AdminAuth）：`GET|POST|PUT /api/announcement/`、`GET|DELETE /api/announcement/:id`、`POST /api/announcement/:id/publish`、`POST /api/announcement/:id/cancel`。
- 定时任务：`service/announcement_task.go`（30s ticker，仅 master 节点执行，处理定时发布与过期）。
- `/api/status` 的 `announcements` 字段改为**新表优先、旧的 console_setting 配置兜底**（`controller/misc.go`）。
- 审计动作新增：`announcement.create` / `update` / `delete` / `publish` / `cancel`。
- 前端：`web/src/features/announcements/`、`web/src/routes/_authenticated/announcements/`、全局弹窗挂载点 `web/src/routes/__root.tsx`、侧边栏入口 `web/src/hooks/use-sidebar-data.ts`、首页公告面板 `web/src/features/dashboard/components/overview/announcements-panel.tsx`。

### 3. 列表「序号」列 seq（新增）

后台各列表的 ID 列改为展示**按创建顺序的 1-based 序号**，序号为查询期派生值，不落库；同时多个列表的默认排序由 `id DESC` 调整为 `id ASC`。

- `model/seq.go`（新增）：`AttachUserSeq` / `AttachRedemptionSeq` / `AttachModelSeq` / `AttachVendorSeq` / `AttachChannelSeq` / `AttachSubscriptionPlanSeq` / `AttachUserSubscriptionSeq`。
- 响应新增 `seq` 字段（`gorm:"-:all"`，不入库），见 `model/{user,redemption,model_meta,vendor_meta,channel,subscription}.go`。
- 默认排序变化：用户 `id asc`、渠道由 `priority desc` 改为 `id asc`、模型 / 供应商 / 兑换码 / 套餐由 `desc` 改为 `asc`（见对应 `model/*.go` 与 `controller/*.go`）。
- `controller/channel_authz.go` 把 `seq` 加入只读字段白名单，禁止写入。
- 前端：`web/src/features/{users,channels,redemption-codes,models,subscriptions}/**` 的列定义与类型新增 `seq`。

### 4. 订阅套餐多分组 + 用户多分组（新增）

套餐的 `upgrade_group` 由单值扩展为**逗号分隔的多分组集合**；用户 `group` 同样可同时持有多个分组（首个为默认使用分组）。订阅到期 / 删除后，按「仍有效订阅分组的并集」重算用户分组。

- `model/user_group_set.go`（新增）：`SplitUserGroups` / `JoinUserGroups` / `NormalizeUserGroups` / `PrimaryUserGroup` / `DefaultUserGroup`。
- `model/subscription.go`（大改）：`NormalizeSubscriptionGroups` / `subscriptionServedGroups` / `recomputeUserGroupForSubscriptionsTx` / `SyncUserGroupWithSubscriptionsTx`，重写降级与过期逻辑。
- `service/group.go`：`GetUserUsableGroups` 改为多成员并集；`setting/ratio_setting/group_ratio.go`：`GetGroupGroupRatio` 按集合逐成员匹配。
- `middleware/auth.go`：`c.Set("group", PrimaryUserGroup(...))`，完整集合放 `user_group`；`middleware/distributor.go`、`middleware/model-rate-limit.go`、`relay/common/relay_info.go` 相应适配。
- 前端：`web/src/components/group-badge.tsx`（新增 `GroupBadgeList`）、用户与订阅编辑抽屉的 Group 改为多选（MultiSelect）。

> ⚠️ **数据语义变化**：`users.group` 与 `subscription_plans.upgrade_group` 现在承载多值字符串（如 `default,vip`）。上游单值假设会误判这些数据。

### 5. 订阅计费按分组隔离（修改）

请求只扣减与「本次使用分组」匹配的订阅（`upgrade_group` 为空视为 `default` 分组）；钱包回退判定同样按分组隔离。分组内订阅额度全部耗尽时返回明确错误（`ErrSubscriptionQuotaExhausted`）。

- `model/subscription.go`：`HasActiveUserSubscription(userId, group)`、`UserActiveSubscriptionsAllowWalletOverflow(userId, group)`、`PreConsumeUserSubscription(..., group, ...)`、`GetActiveUserSubscriptionGroups`。
- `service/funding_source.go`、`service/billing_session.go`（传入 `relayInfo.UsingGroup` 并映射错误码）。

### 6. 订阅套餐删除（新增）

管理员可**硬删除**未被任何用户订阅或未完成订单引用的套餐。

- `model/subscription.go`：`AdminDeleteSubscriptionPlan`（引用校验）；`controller/subscription.go`：同名接口 + `requirePaymentCompliance`。
- 新路由：`DELETE /api/subscription/admin/plans/:id`；审计动作 `subscription.plan_delete`。
- 前端：`web/src/features/subscriptions/components/dialogs/delete-plan-dialog.tsx`（新增）等。

### 7. 订阅重启（新增）

把已失效 / 已过期的用户订阅重新激活，可选**顺延一个套餐周期**与**追加补偿额度**，并同步恢复用户分组。

- `model/subscription.go`：`AdminRestartUserSubscription(id, extendDuration, addAmount)`。
- 新路由：`POST /api/subscription/admin/user_subscriptions/:id/restart`。
- 前端：用户订阅对话框新增 Restart（Extend duration / Compensation amount）。

### 8. Key 支持策略（新增）

一组管理员策略：

1. **按订阅自动派生 Key 分组**：新建 / 编辑 `auto` 分组的令牌时，按用户有效订阅推导分组集合（可融合多订阅分组、按优先级排序、限制数量、控制控制台是否展开真实分组）。
2. **钱包可用范围白名单（列表即生效，无开关）**：非空列表同时起两种作用——**限制**钱包余额只为列出的分组 / 模型付费，并**授予**这些分组对 `auto` 令牌可达，因此仅有余额（无有效订阅）的账户也能用 `auto` 密钥路由到它们；列表留空表示既不限制、也不额外授予。该列表没有启用开关，配置列表本身即是开关。

- 新配置模块 `setting/key_support_setting/`（`key_support_setting.*`，6 个选项：`fusion_enabled`、`expose_groups`、`max_groups`、`group_priority`、`balance_groups`、`balance_models`）。
- `service/key_support.go`：`ResolveKeyGroups` / `EffectiveAutoGroups` / `SyncDerivedTokenGroupsTx`；`model/derived_token_groups.go` 以钩子方式避免 model→service 循环依赖。
- 新路由：`GET /api/token/key-support`；`middleware/auth.go` 对 `auto` 令牌做请求期分组收缩（可用分组 = 有效订阅分组 ∪ 钱包白名单分组，按 `GroupPriority` 排序；钱包分组绕过 `IsUserSelectableGroup`，但仍需 `ratio_setting.ContainsGroupRatio`；该授予仅作用于 `auto` 令牌，显式指定具体分组的令牌校验不变）；`service/billing_session.go` 钱包路径执行 `CheckWalletAllowed`。
- 前端：`web/src/features/system-settings/billing/key-support-section.tsx`（新增）与令牌页相关组件。

### 9. 界面视觉改版（新增）

公共界面与后台共用的一套基础主题令牌被重做，统一为「近白画布 + 深蓝墨色 + 单一信号蓝」，替换上游的默认配色与字体。

- **配色**（`web/src/styles/theme.css` 的 `:root` / `.dark`）：亮色为近白画布 `#fcfcfc`、正文墨色 `#14141e`、实心 CTA 深蓝 `#0f1f3d`；新增 `--signal` / `--color-signal`（`#145aff`），**只用于焦点环、描边与标签**，不作实心按钮填充；状态色统一为成功 `#16ca2e` / 警告 `#ffa64d` / 危险 `#f26052` / 信息 `#0099ff`。深色为反相的深蓝画布 `#0b0f1a`，信号蓝提亮为 `#4d7dff`。
- **圆角与字体**：圆角基准 `--radius` 由 `1rem` 调整为 `0.75rem`；新增 `--font-mono`，`--font-sans` 首选族改为 Inter，并新增依赖 `@fontsource-variable/inter`、`@fontsource-variable/roboto-mono`，在 `web/src/styles/index.css` 全局引入（字体本地自托管，不请求外部 CDN）。
- **首页与 CTA**：徽章与背景光晕改用信号蓝，标题渐变由蓝→紫改为 `primary → signal → info`（`web/src/features/home/components/sections/hero.tsx`、`cta.tsx`）。
- **公共顶栏**：未滚动时导航高度由 64px 调整为 72px（`web/src/components/layout/components/public-header.tsx`）。
- **容器宽度（预置，暂无可见差异）**：`max-w-container` 由 1280px 调整为 1200px（`web/src/styles/index.css`）。该工具类目前没有被任何页面引用（首页各区块仍使用 `max-w-6xl`），因此只改动了取值本身。

> 仅外观改动：不新增路由、不改接口、不动数据。`web/src/styles/theme-presets.css` 中的可选配色预设与用户本地主题偏好不受影响——未选择预设时落到上述基础令牌。

### 10. 站内文档页 `/docs`（新增）

控制台的「文档」入口不再跳转 GitHub，而是在站内直接渲染仓库的 README。

- 新增公开前端路由 `/docs`（`web/src/routes/docs/index.tsx`）与页面 `web/src/features/docs/`（`index.tsx` / `api.ts` / `source.ts`）。
- 文档来源由 `docs_link` 设置解析：指向 GitHub 的 `blob` 或 `raw` markdown 地址时按该仓库渲染，否则回退到本仓库 `main` 分支的 `README.md`（`source.ts`）。
- README 内的相对图片改写到 raw 主机、相对链接指向 `blob` 目录，避免在控制台域名下 404（`resolveRepoImages`）。
- 顶部导航与首页 Docs 按钮统一指向站内 `/docs`（`web/src/hooks/use-top-nav-links.ts`、`hero.tsx`），忽略 `docs_link` 中的外部地址。
- 「拉取更新」按钮仅对 root 管理员（`role >= 100`）可见，以 `no-store` 强制重新拉取（`api.ts`，15s 超时）。

### 11. 其它修复与调整

| 项 | 文件 | 说明 |
| --- | --- | --- |
| Playground `/pg` 路径别名 | `relaykit/dto/channel_settings.go` | 修复：声明为 `/v1` 的高级自定义路由不再错误拒绝 `/pg` 请求 |
| 数据库时间读取 | `model/db_time.go` | 新增事务句柄版本 `GetDBTimestampTx`，避免单连接池 / 事务内二次取连接死锁 |
| 富文本自动识别 HTML | `web/src/components/rich-content.tsx`、`web/src/components/rich-content-utils.ts`（新增） | 内容渲染自动判定 Markdown / HTML（`mode: 'auto'` 为默认） |
| 文档链接 | `setting/operation_setting/general_setting.go`、`web/src/components/layout/components/footer.tsx` | `DocsLink` 默认指向本仓库 README，并作为站内 `/docs` 页的数据源（见「站内文档页」）；页脚文档链接指向 GitHub 上的本 README |
| 测试 | `model/*_test.go`、`controller/subscription_upgrade_group_test.go`、`service/group_usable_groups_test.go`、`relaykit/dto/playground_path_alias_test.go`、`web/src/components/rich-content-utils.test.ts` 等 | 覆盖上述新增与修改行为 |

### 12. 兼容性影响汇总

**数据库**

- 新增表 `announcements`（增量，可安全忽略）。
- 未新增列，但 `users.group` / `subscription_plans.upgrade_group` 的**存量列语义变化**（单值 → 逗号分隔多值）。
- `seq` 全部为派生字段，不落库。

**API 行为**

- 新增路由：`/api/announcements/current`、`/api/announcements/popup`、`/api/announcement/*`（7 个）、`DELETE /api/subscription/admin/plans/:id`、`POST /api/subscription/admin/user_subscriptions/:id/restart`、`GET /api/token/key-support`。
- 多个列表响应新增 `seq` 字段，且**默认排序改变**（影响分页顺序与首屏内容）。
- `/api/status` 的 `announcements` 数据来源改变（新表优先）。
- 计费语义变化：订阅按分组隔离；`auto` 令牌分组在请求期被收缩。

**配置默认值**

- `SystemName` 默认 `myapi`；`DocsLink` 默认指向本仓库 README；新增 `key_support_setting.*`（默认关闭）。

**界面**

- 基础主题令牌（配色、圆角）与界面字体改变，与上游外观不同；新增前端路由 `/docs`（站内渲染 README）。两项均不涉及后端接口与数据结构。

**部署**

- 二进制、镜像、容器、网络、数据库名均改为 `myapi`。与上游 Compose 不能直接混用。

---

## 🚀 快速开始（Docker）

> 本仓库**不含任何默认账号或密码**：数据库与 Redis 凭据由安装脚本随机生成（或手动写入 `.env`），控制台管理员账号在首次访问时通过**初始化向导**创建。

### 方式一：一键安装（推荐）

一行命令即可完成：获取源码 → 随机生成凭据写入 `.env` → 拉取预构建镜像并启动 **myapi + PostgreSQL + Redis**。

```bash
curl -fsSL https://raw.githubusercontent.com/axzcnzxis/myapi/main/install.sh | bash
```

- 默认安装到 `./myapi`，默认端口 `32180`。
- **安装过程不在本机构建**：直接拉取预构建镜像 `ghcr.io/axzcnzxis/myapi:latest`，因此对机器配置要求低（小内存机器也不会卡在构建阶段）。镜像同时发布到 Docker Hub（`aksgxc/myapi:latest`），可用 `MYAPI_IMAGE` 切换来源，例如 `MYAPI_IMAGE=aksgxc/myapi:latest`。
- 脚本随机生成 `POSTGRES_PASSWORD` / `REDIS_PASSWORD` / `SESSION_SECRET` 写入 `.env`（以 `umask 077` 创建），已存在的 `.env` 不会被覆盖。
- **重复执行是安全的，且能自动修复旧配置**：复用已有目录时，若其中的 `docker-compose.yml` 还是旧版（含 `build:`，或 `image` 指向 `myapi:latest` 这类只在本地构建出来的标签），脚本会先把原文件备份为 `docker-compose.yml.bak-<时间戳>`，再替换为仓库中「只拉取预构建镜像」的最新配置——否则 `docker compose pull` 会去拉一个不存在于任何 registry 的镜像而失败。不含 `build:` 的自定义配置不会被改动。
- 可用环境变量覆盖默认值：`MYAPI_DIR`（安装目录）、`MYAPI_PORT`（对外端口）、`MYAPI_IMAGE`（镜像地址）、`MYAPI_REF`（分支/标签）、`MYAPI_REPO`（源码仓库）、`MYAPI_RAW_BASE`（复用旧目录时获取最新 compose 的地址前缀），例如 `MYAPI_PORT=8080 bash install.sh`。

想先审阅脚本再执行：

```bash
curl -fsSLO https://raw.githubusercontent.com/axzcnzxis/myapi/main/install.sh
less install.sh && bash install.sh
```

安装完成后打开 `http://<服务器地址>:32180`，页面会自动进入初始化向导：

1. 设置**管理员用户名与密码**（即控制台 root 账号，密码由你在此处设定）；
2. 按需选择「自用模式」/「演示站点」；
3. 提交后初始化完成，使用该账号登录控制台。

初始化状态可通过 `GET /api/setup` 查询（`data.status` 表示是否已完成初始化）。

> ⚠️ 生产环境建议：仅通过 HTTPS 反向代理暴露（注意流式与 WebSocket 升级）、按需收紧 `MYAPI_PORT` 的绑定范围，并定期备份 `pg_data` 卷与 `./data` 目录。

### 方式二：手动 Docker Compose

不使用脚本时，仓库自带的 [`docker-compose.yml`](./docker-compose.yml) 也可手动部署。

**1. 获取源码并准备配置**

```bash
git clone https://github.com/axzcnzxis/myapi.git
cd myapi

cp .env.example .env
```

**2. 填写 `.env` 中的凭据（必填，无默认值）**

```ini
POSTGRES_USER=myapi
POSTGRES_DB=myapi
POSTGRES_PASSWORD=<自定义数据库密码>
REDIS_PASSWORD=<自定义 Redis 密码>
SESSION_SECRET=<openssl rand -hex 32 生成>
```

- `SESSION_SECRET` 用于签发登录会话并派生敏感字段的加密密钥；不设置时程序每次启动都会随机生成，重启即导致所有登录会话失效。
- 缺少 `POSTGRES_PASSWORD` / `REDIS_PASSWORD` / `SESSION_SECRET` 时 `docker compose` 会直接报错退出，不会回退到任何内置默认值。
- `POSTGRES_USER` / `POSTGRES_DB` 仅在数据卷首次初始化时生效；若复用已有的 `pg_data` 卷，必须与卷内账号保持一致。

**3. 启动（拉取预构建镜像）**

```bash
docker compose up -d        # 只拉取镜像，不在本机构建

docker compose logs -f myapi
```

需要从源码构建镜像（改动代码、或不想用预构建镜像）时叠加构建文件：

```bash
docker compose -f docker-compose.yml -f docker-compose.build.yml up -d --build
```

> 构建阶段要下载前端/后端依赖并编译，内存占用较高，小内存机器（如 2G）容易被拖死，普通部署建议直接用预构建镜像。

**4. 完成初始化向导**

打开 `http://<服务器地址>:32180`，页面会自动进入初始化向导（步骤同「方式一」）。

### 方式三：单容器（SQLite）

```bash
mkdir -p data

docker run --name myapi -d --restart unless-stopped \
  -p 127.0.0.1:32180:32180 \
  -e TZ=Asia/Shanghai \
  -e SESSION_SECRET="$(openssl rand -hex 32)" \
  -v "$(pwd)/data:/data" \
  ghcr.io/axzcnzxis/myapi:latest
```

`data` 目录用于持久化 SQLite 数据库；同样在首次访问 <http://localhost:32180> 时通过初始化向导创建管理员账号。

需要从源码构建镜像时：`docker build -t myapi:local .`，再把上面的镜像名换成 `myapi:local`。

### 更新版本

```bash
cd <安装目录>
docker compose pull             # 拉取新镜像
docker compose up -d --no-build # 用新镜像重建容器
```

数据保留在 `pg_data` 卷与 `./data`、`./logs` 目录中，`up -d` 只重建容器，不会动数据。

### 首次请求

1. 在控制台添加渠道（上游 API Key、可用模型、分组），并执行渠道测试。
2. 配置模型价格，确保用户有额度或有效订阅。
3. 在控制台创建 API Key（分组与模型需与渠道匹配）。
4. 把客户端 Base URL 设为 `http://localhost:32180/v1`。

```bash
export MYAPI_KEY=sk-xxxxxxxx
curl --fail-with-body http://localhost:32180/v1/models \
  -H "Authorization: Bearer ${MYAPI_KEY}"
```

---

## ⚙️ 配置

### Compose 部署变量（写入 `.env`，由 `docker-compose.yml` 读取）

| 变量 | 必填 | 说明 |
| --- | --- | --- |
| `POSTGRES_USER` | 否 | PostgreSQL 账号，默认 `myapi`；仅在 `pg_data` 卷首次初始化时生效 |
| `POSTGRES_PASSWORD` | **是** | PostgreSQL 密码，无默认值 |
| `POSTGRES_DB` | 否 | 数据库名，默认 `myapi` |
| `REDIS_PASSWORD` | **是** | Redis 密码，无默认值 |
| `SESSION_SECRET` | **是** | 会话与敏感字段加密密钥，无默认值 |
| `MYAPI_PORT` | 否 | 控制台对外端口，默认 `32180` |
| `MYAPI_IMAGE` | 否 | myapi 镜像地址，默认 `ghcr.io/axzcnzxis/myapi:latest`（预构建镜像）；Docker Hub 镜像为 `aksgxc/myapi:latest` |
| `NODE_NAME` | 否 | 节点名，默认 `myapi-node-1` |
| `TZ` | 否 | 时区，默认 `Asia/Shanghai` |

### 应用环境变量（在 `docker-compose.yml` 的 `environment` 中调整）

| 变量 | 说明 |
| --- | --- |
| `SQL_DSN` | 主数据库连接串；不设置则使用 SQLite |
| `LOG_SQL_DSN` | 可选的独立日志库（支持 ClickHouse） |
| `REDIS_CONN_STRING` | Redis 连接串 |
| `SESSION_SECRET` | 持久化认证密钥；多节点必须一致 |
| `CRYPTO_SECRET` | 默认取 `SESSION_SECRET` |
| `SESSION_COOKIE_SECURE` | HTTPS 控制台设为 `true` |
| `SESSION_COOKIE_TRUSTED_URL` | Secure 模式下必填的精确 HTTPS Origin |
| `TRUSTED_PROXIES` | 可信反向代理 IP/CIDR，或 `none` |
| `NODE_NAME` | 节点名，用于审计日志标识 |

完整变量说明见 [`docker-compose.yml`](./docker-compose.yml) 中的注释、[`.env.example`](./.env.example) 与 [`docs/authentication.md`](./docs/authentication.md)。

生产环境请：置于 HTTPS 反向代理之后（注意流式与 WebSocket 升级）、持久化并备份数据库与挂载目录、多节点共享主库与认证密钥。

---

## 🛠 开发

后端 Go + Gin；前端 React 19 + TypeScript + Rsbuild + TanStack + Tailwind CSS 4（使用 Bun）；界面字体 Inter / Roboto Mono，经 `@fontsource-variable/*` 本地自托管。

```bash
# 仓库根目录
cd web
bun install --frozen-lockfile
bun run build
cd ..
go run .
```

前端开发服务器：

```bash
cd web
bun run dev -- --port 5173
```

| 目录 | 职责 |
| --- | --- |
| `router/`、`middleware/`、`controller/` | HTTP 路由、鉴权、接口实现 |
| `relay/` | 上游适配与请求转发 |
| `service/`、`model/` | 业务逻辑与持久化 |
| `relaykit/` | 可独立构建的协议 DTO / 转换模块 |
| `plugins/tasks/` | JavaScript 任务插件 |
| `web/` | Web 控制台 |

校验：`make test`（Go）、`web/` 下 `bun run typecheck` / `bun run lint` / `bun run test` / `bun run build`；改动 RelayKit 需在 `relaykit/` 下执行 `GOWORK=off go build ./...`。

---

## 📚 文档

本仓库的文档以本 README 为准：

- 本 README（安装、部署、与原版差异）
- [`docs/`](./docs/) 目录下的补充文档
- [`relaykit/README.md`](./relaykit/README.md)、[`web/AGENTS.md`](./web/AGENTS.md)、[`electron/README.md`](./electron/README.md)

控制台内的「文档」入口渲染站内 `/docs` 页，内容即本 README（数据源由 `DocsLink` 设置决定，见「与原版差异 · 站内文档页」）；页脚文档链接指向 GitHub 上的本 README。

---

## 🗑️ 卸载

卸载脚本与安装脚本配套：默认只移除 **myapi 相关容器、网络与本地镜像**，**保留数据**；加 `--purge` 才会连数据一起删除。

```bash
# 默认：停止并移除容器/网络/镜像，保留数据库卷与 data、logs 目录
curl -fsSL https://raw.githubusercontent.com/axzcnzxis/myapi/main/uninstall.sh | bash

# 彻底清理：额外删除 pg_data 卷与 data、logs 目录（不可恢复）
curl -fsSL https://raw.githubusercontent.com/axzcnzxis/myapi/main/uninstall.sh | bash -s -- --purge
```

- 在安装目录（默认 `./myapi`）内执行，或用 `MYAPI_DIR=/path/to/myapi` 指定；`MYAPI_PURGE=1` 等价于 `--purge`。
- 只操作本项目的 Compose 工程（会先校验 `docker-compose.yml` 中含 `container_name: myapi`），不会执行 `docker system prune`，也不影响其它容器。
- 源码目录不会被删除；确认无误后可自行 `rm -rf myapi`。
- 想先审阅脚本再执行：`curl -fsSLO https://raw.githubusercontent.com/axzcnzxis/myapi/main/uninstall.sh && less uninstall.sh && bash uninstall.sh`。

---

## 📜 许可与致谢

本项目基于 [new-api](https://github.com/QuantumNous/new-api) 二次开发，采用 **GNU Affero General Public License v3.0（AGPL-3.0）**，并附带 Section 7 附加条款，详见 [`LICENSE`](./LICENSE) 与 [`NOTICE`](./NOTICE)。

按附加条款要求，本仓库保留：

- 上游版权与作者归属声明（`Copyright (C) 2023-2026 QuantumNous`）；
- 署名声明 `Frontend design and development by New API contributors.`（见控制台「关于」页）；
- 指向原项目的可见链接：<https://github.com/QuantumNous/new-api>（见控制台「关于」页）。

上游链路：new-api ← [One API](https://github.com/songquanpeng/one-api)（MIT）。第三方依赖声明见 [`THIRD-PARTY-LICENSES.md`](./THIRD-PARTY-LICENSES.md)。

> 商业授权相关事宜请联系上游：<support@quantumnous.com>。
