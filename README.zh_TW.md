<div align="center">

![myapi](/web/public/logo.png)

# myapi

**以 new-api 為基礎二次開發的自託管 AI 閘道**

<a href="./README.md">简体中文</a> |
<a href="./README.en.md">English</a> |
<strong>繁體中文</strong> |
<a href="./README.fr.md">Français</a> |
<a href="./README.ja.md">日本語</a>

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

## 關於本倉庫

**myapi** 是 [new-api](https://github.com/QuantumNous/new-api)（自託管 AI 閘道）的二次開發分支，**不是**官方倉庫。

- **本倉庫的文件以簡體中文 [`README.md`](./README.md) 為準**：安裝、部署，以及與上游 `v1.0.0-rc.41` 的完整差異清單。
- 本頁僅作為語言入口，刻意不維護完整翻譯，以免與 `README.md` 不一致。
- 上游專案：<https://github.com/QuantumNous/new-api>。本分支的問題請勿回報到上游。

## 快速開始

```bash
curl -fsSL https://raw.githubusercontent.com/axzcnzxis/myapi/main/install.sh | bash
```

開啟 <http://localhost:32180> 完成初始化設定。腳本會隨機產生 PostgreSQL／Redis 密碼與持久化的 `SESSION_SECRET` 並寫入 `.env`；手動 Compose 步驟、單一容器（SQLite）用法與完整設定變數見 [`README.md`](./README.md)。

## 授權與致謝

基於 [new-api](https://github.com/QuantumNous/new-api) 二次開發，採用 **AGPL-3.0**（含 Section 7 附加條款），詳見 [`LICENSE`](./LICENSE) 與 [`NOTICE`](./NOTICE)。保留上游署名 `Frontend design and development by New API contributors.` 及指向原專案的可見連結 <https://github.com/QuantumNous/new-api>。
