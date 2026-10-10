<div align="center">

![myapi](/web/public/logo.png)

# myapi

**new-api をベースにしたセルフホスト型 AI ゲートウェイ（派生版）**

<a href="./README.md">简体中文</a> |
<a href="./README.en.md">English</a> |
<a href="./README.zh_TW.md">繁體中文</a> |
<a href="./README.fr.md">Français</a> |
<strong>日本語</strong>

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

## このリポジトリについて

**myapi** は [new-api](https://github.com/QuantumNous/new-api)（セルフホスト型 AI ゲートウェイ）の派生フォークであり、**公式リポジトリではありません**。

- **本リポジトリのドキュメントは簡体字中国語の [`README.md`](./README.md) が正式です**（インストール、デプロイ、上流 `v1.0.0-rc.41` との差分の全一覧）。
- このページは言語入口のみです。`README.md` と食い違わないよう、完全な翻訳は意図的に保守していません。
- 上流プロジェクト: <https://github.com/QuantumNous/new-api>。本フォークの問題を上流へ報告しないでください。

## クイックスタート

```bash
git clone https://github.com/axzcnzxis/myapi.git
cd myapi

docker compose build      # このソースツリーから myapi イメージをビルド
docker compose up -d
```

<http://localhost:3000> を開き、初期設定を完了します。外部公開の前に PostgreSQL / Redis の既定パスワードを変更し、永続的な `SESSION_SECRET` を設定してください。単一コンテナ（SQLite）での利用方法と全設定変数は [`README.md`](./README.md) を参照してください。

## ライセンスと謝辞

[new-api](https://github.com/QuantumNous/new-api) をベースに、**AGPL-3.0**（Section 7 の追加条項を含む）で提供されます。詳細は [`LICENSE`](./LICENSE) と [`NOTICE`](./NOTICE) を参照してください。上流のクレジット `Frontend design and development by New API contributors.` と、原プロジェクトへの可視リンク <https://github.com/QuantumNous/new-api> を保持しています。
