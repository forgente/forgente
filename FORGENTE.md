# Forgente

Forgente is a soft fork of [Gitea](https://github.com/go-gitea/gitea), hosted at
[github.com/forgente/forgente](https://github.com/forgente/forgente) with its home
at [forgente.com](https://forgente.com).

"Soft fork" means: Forgente tracks upstream Gitea closely and regularly merges its
changes, while carrying its own commits on top. The GitHub repository is a regular
repository (not a GitHub fork), so it has no "forked from" association.

## Repository layout

| Remote     | URL                                        | Purpose                     |
|------------|--------------------------------------------|-----------------------------|
| `origin`   | https://github.com/forgente/forgente.git   | Forgente (this project)     |
| `upstream` | https://github.com/go-gitea/gitea.git      | Gitea (sync source)         |

Clone and set up:

```bash
git clone https://github.com/forgente/forgente.git
cd forgente
git remote add upstream https://github.com/go-gitea/gitea.git
```

## Syncing from upstream Gitea

Run the helper script:

```bash
contrib/forgente/sync-upstream.sh
```

It fetches `upstream`, merges `upstream/main` into `main`, and syncs tags and
release branches. Resolve any merge conflicts (they will be in files Forgente has
modified), then push:

```bash
git push origin main --tags
```

To keep merges tractable, prefer additive changes (new files under `contrib/forgente/`,
new packages) over edits to upstream files where possible.

## Development

Everything from upstream applies unchanged — see [README.md](README.md) and
[CONTRIBUTING.md](CONTRIBUTING.md). Quick reference: `make help`.
