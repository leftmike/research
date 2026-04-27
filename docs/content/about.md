---
title: "About"
---

## How this site is published

This site is built with [Hugo](https://gohugo.io/) and automatically deployed to
[GitHub Pages](https://pages.github.com/) using a GitHub Actions workflow.

### Workflow overview

1. A push to `main` triggers `.github/workflows/hugo.yml`.
2. Hugo builds the site from the `docs/` directory.
3. The resulting `public/` directory is uploaded as a Pages artifact.
4. GitHub Pages serves the artifact at `https://<owner>.github.io/<repo>/`.

### Enabling GitHub Pages in your repository

1. Go to **Settings → Pages** in your repository.
2. Under **Source**, choose **GitHub Actions**.
3. Push a commit — the workflow handles the rest.

No `gh-pages` branch or manual builds needed.
