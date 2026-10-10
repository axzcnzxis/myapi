<div align="center">

![myapi](/web/public/logo.png)

# myapi

**A self-hosted AI gateway, forked from new-api**

<a href="./README.md">简体中文</a> |
<strong>English</strong> |
<a href="./README.zh_TW.md">繁體中文</a> |
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

## About this repository

**myapi** is a derivative fork of [new-api](https://github.com/QuantumNous/new-api), a self-hosted AI gateway for models, applications and agents. It is **not** the official new-api repository.

- The **authoritative documentation is the Simplified Chinese [`README.md`](./README.md)**: installation, deployment, and the complete list of differences from upstream `v1.0.0-rc.41`.
- This page is only a language entry point. It is deliberately not a full translation, so that it cannot drift out of sync with `README.md`.
- Upstream project: <https://github.com/QuantumNous/new-api>. Problems of this fork must not be reported upstream.

## Quick start

```bash
git clone https://github.com/axzcnzxis/myapi.git
cd myapi

docker compose build      # build the myapi image from this source tree
docker compose up -d
```

Then open <http://localhost:3000> and complete the setup wizard. Change the default PostgreSQL / Redis passwords and set a persistent `SESSION_SECRET` before exposing it publicly. See [`README.md`](./README.md) for single-container (SQLite) usage and all configuration variables.

## License and attribution

Based on [new-api](https://github.com/QuantumNous/new-api), licensed under **AGPL-3.0** with additional terms under Section 7 — see [`LICENSE`](./LICENSE) and [`NOTICE`](./NOTICE). The upstream attribution `Frontend design and development by New API contributors.` and a visible link to the original project <https://github.com/QuantumNous/new-api> are preserved.
