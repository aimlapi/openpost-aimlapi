# OpenPost social previews

This package is the shared metadata catalog for the marketing and documentation sites. Marketing pages use the static `assets/brand/og-image.png` card. Documentation pages use `assets/brand/og-docs.png` from the docs origin.

Both PNGs are 1200 x 630 and have editable SVG sources in `assets/brand/`. Static files keep social previews independent of an edge runtime and let each site serve its own image directly.

`scripts/social-images/catalog.mjs` keeps the small docs-page catalog in sync with Markdown headings. Asset synchronization refreshes it automatically; `bun run check -- social-images` rejects stale catalog metadata.

Run the focused checks from the repository root:

```sh
bun run check -- social-images
bun run build -- marketing
bun run build -- docs
```

Inspect both PNGs after changing their SVG sources, then run the focused checks above.
