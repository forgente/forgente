<div align="center">
  <img src="public/assets/img/logo.svg" alt="Forgente" width="96"/>

# Forgente

[![](https://github.com/forgente/forgente/actions/workflows/release-nightly.yml/badge.svg?branch=main)](https://github.com/forgente/forgente/actions/workflows/release-nightly.yml?query=branch%3Amain "Release Nightly")
[![](https://img.shields.io/badge/License-MIT-blue.svg)](https://opensource.org/licenses/MIT "License: MIT")

**The complete software forge you fully own.**

</div>

Forgente is an all-in-one software development service: Git hosting, code
review, issue tracking, project boards, wiki, package registry, and CI/CD
compatible with GitHub Actions workflows. Written in Go, it runs as a single
binary on Linux, macOS, FreeBSD/OpenBSD, and Windows — on your hardware,
under your control, with no telemetry.

Where Forgente is heading is laid out in [ROADMAP.md](ROADMAP.md); how it is
built and shipped is documented in [FORGENTE.md](FORGENTE.md).

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

Documentation lives at [docs.forgente.com](https://docs.forgente.com).
Forgente-specific behavior and operations are documented in
[FORGENTE.md](FORGENTE.md).

## Built on Gitea

Forgente builds on [Gitea](https://github.com/go-gitea/gitea) and tracks it
as an upstream: Gitea's improvements and security fixes are merged
continuously while Forgente's own features are built on top, and Forgente
stays configuration- and API-compatible. Enormous credit belongs to the Gitea
maintainers and contributors. The tracking mechanics are documented in
[FORGENTE.md](FORGENTE.md).

## Contributing

Fork → patch → push → pull request. Read the
[contributors guide](CONTRIBUTING.md) first. Security issues: write privately
to security@forgente.com.

## License

Forgente is licensed under the [MIT License](LICENSE), as is the Gitea code
it builds on. Gitea's name and logo are trademarks of their respective
owners; Forgente uses its own name and mark.
