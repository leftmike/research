---
title: "Getting Started with Hugo and GitHub Pages"
date: 2026-04-27
---

[Hugo](https://gohugo.io/) is a fast static site generator written in Go.
Combined with [GitHub Pages](https://pages.github.com/), you can host a site
for free with automatic deployments on every push.

## Prerequisites

- A GitHub repository
- Hugo installed locally (`brew install hugo` or see [hugo releases](https://github.com/gohugoio/hugo/releases))

## Project layout

```
docs/              ← Hugo source (hugo.toml, content/, themes/, …)
.github/
  workflows/
    hugo.yml       ← GitHub Actions deployment workflow
```

## Creating content

```bash
# From the docs/ directory
hugo new content posts/my-post.md
```

Edit the generated file, set `draft: false`, then push. GitHub Actions builds
and publishes the update automatically.

## Local preview

```bash
cd docs
hugo server
```

Open `http://localhost:1313` to preview before pushing.
