<div align="center">

![myapi](/web/public/logo.png)

# myapi

**Une passerelle IA auto-hébergée, fork de new-api**

<a href="./README.md">简体中文</a> |
<a href="./README.en.md">English</a> |
<a href="./README.zh_TW.md">繁體中文</a> |
<strong>Français</strong> |
<a href="./README.ja.md">日本語</a>

<p align="center">
  <a href="https://github.com/axzcnzxis/myapi/blob/main/LICENSE">
    <img src="https://img.shields.io/github/license/axzcnzxis/myapi?color=brightgreen" alt="licence">
  </a><!--
  --><a href="https://github.com/axzcnzxis/myapi/releases">
    <img src="https://img.shields.io/github/v/release/axzcnzxis/myapi?color=brightgreen&include_prereleases" alt="version">
  </a>
</p>

</div>

---

## À propos de ce dépôt

**myapi** est un fork dérivé de [new-api](https://github.com/QuantumNous/new-api), une passerelle IA auto-hébergée. Ce n'est **pas** le dépôt officiel.

- **La documentation de référence de ce dépôt est le [`README.md`](./README.md) en chinois simplifié** : installation, déploiement et liste complète des différences par rapport à l'amont `v1.0.0-rc.41`.
- Cette page n'est qu'un point d'entrée linguistique. Elle n'est volontairement pas une traduction complète, afin de ne pas diverger de `README.md`.
- Projet amont : <https://github.com/QuantumNous/new-api>. Les problèmes de ce fork ne doivent pas être signalés en amont.

## Démarrage rapide

```bash
curl -fsSL https://raw.githubusercontent.com/axzcnzxis/myapi/main/install.sh | bash
```

Ouvrez <http://localhost:32180> et terminez l'assistant d'initialisation. Le script génère des mots de passe PostgreSQL / Redis aléatoires et un `SESSION_SECRET` persistant dans `.env` ; les étapes Compose manuelles, l'usage en conteneur unique (SQLite) et toutes les variables de configuration sont décrits dans [`README.md`](./README.md).

## Licence et attribution

Basé sur [new-api](https://github.com/QuantumNous/new-api), sous **AGPL-3.0** avec les conditions supplémentaires de la section 7 — voir [`LICENSE`](./LICENSE) et [`NOTICE`](./NOTICE). L'attribution amont (`Frontend design and development by New API contributors.`) et un lien visible vers le projet original <https://github.com/QuantumNous/new-api> sont conservés.
