---
title: "Hugo Tips and Tricks"
date: 2026-04-27
---

A few handy things to know when working with Hugo.

## Front matter

Every content file starts with YAML, TOML, or JSON front matter between `---` delimiters:

```yaml
---
title: "My Post"
date: 2026-04-27
draft: false
tags: ["hugo", "web"]
---
```

## Shortcodes

Hugo ships with built-in shortcodes for common embeds:

```
{{</* youtube dQw4w9WgXcQ */>}}
{{</* figure src="/images/photo.jpg" alt="A photo" */>}}
```

## Taxonomies

Add `tags` or `categories` to front matter and Hugo automatically generates listing pages for them.

## Build flags

```bash
# Include draft posts in local preview
hugo server --buildDrafts

# Build for production (output goes to docs/public/)
hugo --minify
```
