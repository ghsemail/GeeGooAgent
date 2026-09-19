# Vendored Archify skill (optional)

GeeGooAgent compiles typed IR in `internal/diagram` and can optionally
render it with [Archify](https://github.com/tt-a1i/archify) at generate time.

Runtime never requires Node. If this folder has no `bin/archify.mjs`,
`go generate ./internal/diagram` uses the builtin HTML renderer.

## Install (generate-time only)

From the repo root:

```bash
node scripts/diagram/fetch-archify.mjs
```

Or copy the `archify/` skill package from a local Archify checkout to this
directory so that `third_party/archify/bin/archify.mjs` exists.

Pinned upstream: `tt-a1i/archify` skill package, MIT.
