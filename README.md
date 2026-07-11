<div align="center">
  <img src="public/assets/img/logo.svg" alt="Forgente" width="96"/>

# Forgente

[![](https://github.com/forgente/forgente/actions/workflows/release-nightly.yml/badge.svg?branch=main)](https://github.com/forgente/forgente/actions/workflows/release-nightly.yml?query=branch%3Amain "Release Nightly")
[![](https://img.shields.io/badge/License-MIT-blue.svg)](https://opensource.org/licenses/MIT "License: MIT")

**A painless self-hosted Git service.**

</div>

Forgente is an all-in-one software development service: Git hosting, code
review, issue tracking, project boards, wiki, package registry, and CI/CD
compatible with GitHub Actions workflows. Written in Go, it runs as a single
binary on Linux, macOS, FreeBSD/OpenBSD, and Windows.

Forgente is a **soft fork of [Gitea](https://github.com/go-gitea/gitea)**: it
tracks upstream closely and merges Gitea's improvements and security fixes
continuously, while building its own identity and features on top. Enormous
credit belongs to the Gitea maintainers and contributors — see
[FORGENTE.md](FORGENTE.md) for exactly how the fork relates to upstream and
[ROADMAP.md](ROADMAP.md) for where it is heading.

## Install

**Container** (recommended):

```bash
docker run -p 3000:3000 -p 2222:22 forgente/forgente:latest
```

Images are published to [Docker Hub](https://hub.docker.com/r/forgente/forgente)
and [GHCR](https://github.com/forgente/forgente/pkgs/container/forgente) as
`latest`, `<major>`, `<major.minor>`, `<version>` per release plus a
`main-nightly` development build (all with `-rootless` variants). Existing
Gitea container setups (volumes, `GITEA_*` environment variables) work
unchanged.

**Binaries**: signed builds for every platform are attached to
[GitHub releases](https://github.com/forgente/forgente/releases) and mirrored
at [dl.forgente.com](https://dl.forgente.com/forgente/) (nightlies under
[`main-nightly`](https://dl.forgente.com/forgente/main-nightly/)) — each with
SHA-256 checksum, GPG signature (key
`67129BAD57A2C8D2186032489D6FD2FD6E0B9BA5`), and sigstore bundle.

**Snap**:

```bash
sudo snap install forgente --edge
```

## Building from source

See [docs/build-setup.md](docs/build-setup.md) for prerequisites and
[docs/development.md](docs/development.md) for the development environment.

```bash
TAGS="bindata" make build
./forgente web
```

## Documentation

Forgente is configuration- and API-compatible with Gitea, so the upstream
[documentation](https://docs.gitea.com/) applies. Forgente-specific
behavior is documented in [FORGENTE.md](FORGENTE.md).

## Contributing

Fork → patch → push → pull request. Read the
[contributors guide](CONTRIBUTING.md) first. Security issues: write privately
to security@forgente.com.

## License

Forgente is licensed under the [MIT License](LICENSE), as is the Gitea code
it builds on. Gitea's name and logo are trademarks of their respective
owners; Forgente uses its own name and mark.
