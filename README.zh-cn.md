<div align="center">
  <img src="public/assets/img/logo.svg" alt="Forgente" width="96"/>

# Forgente

[![](https://github.com/forgente/forgente/actions/workflows/release-nightly.yml/badge.svg?branch=main)](https://github.com/forgente/forgente/actions/workflows/release-nightly.yml?query=branch%3Amain "Release Nightly")
[![](https://img.shields.io/badge/License-MIT-blue.svg)](https://opensource.org/licenses/MIT "License: MIT")

**您完全拥有的完整软件协作平台。**

</div>

[English](./README.md) | [繁體中文](./README.zh-tw.md)

Forgente 是一个一体化的软件开发服务：Git 托管、代码审查、问题跟踪、项目看板、
Wiki、软件包仓库，以及与 GitHub Actions 工作流兼容的 CI/CD。使用 Go 编写，可作为
单一二进制文件运行在 Linux、macOS、FreeBSD/OpenBSD 和 Windows 上——运行在您自己
的硬件上，由您掌控，没有遥测。

Forgente 的发展方向见 [ROADMAP.md](ROADMAP.md)；它的构建与发布方式记录在
[FORGENTE.md](FORGENTE.md) 中。

## 安装

**容器**（推荐）：

```bash
docker run -p 3000:3000 -p 2222:22 forgente/forgente:latest
```

镜像发布在 [Docker Hub](https://hub.docker.com/r/forgente/forgente) 和
[GHCR](https://github.com/forgente/forgente/pkgs/container/forgente)，标签包括每次
发布对应的 `latest`、`<major>`、`<major.minor>`、`<version>`，以及 `main-nightly`
开发构建（均提供 `-rootless` 变体）。现有的 Gitea 容器配置（数据卷、`GITEA_*`
环境变量）无需更改即可继续使用。

**二进制文件**：为每个平台签名构建的版本会附加在
[GitHub Releases](https://github.com/forgente/forgente/releases) 中，并镜像到
[dl.forgente.com](https://dl.forgente.com/forgente/)（每夜构建位于
[`main-nightly`](https://dl.forgente.com/forgente/main-nightly/)）——每个文件都附
有 SHA-256 校验和、GPG 签名（密钥
`67129BAD57A2C8D2186032489D6FD2FD6E0B9BA5`）以及 sigstore 签名包。

**Snap**：

```bash
sudo snap install forgente --edge
```

## 从源代码构建

请参阅 [docs/build-setup.md](docs/build-setup.md) 了解前置条件，以及
[docs/development.md](docs/development.md) 了解开发环境。

```bash
TAGS="bindata" make build
./forgente web
```

## 文档

文档位于 [docs.forgente.com](https://docs.forgente.com)。Forgente 特有的行为与运
维说明记录在 [FORGENTE.md](FORGENTE.md) 中。

## 基于 Gitea 构建

Forgente 基于 [Gitea](https://github.com/go-gitea/gitea) 构建，并将其作为上游持
续跟踪：Gitea 的改进和安全修复会持续合并进来，同时 Forgente 在此基础上构建自己的
功能，并与 Gitea 保持配置和 API 兼容。非常感谢 Gitea 的维护者和贡献者。跟踪机制的
细节记录在 [FORGENTE.md](FORGENTE.md) 中。

## 贡献

Fork → 修改 → 推送 → 发起 Pull Request。请先阅读
[贡献者指南](CONTRIBUTING.md)。安全问题请私下写信至 security@forgente.com。

## 许可证

Forgente 基于 [MIT 许可证](LICENSE) 授权，其所基于的 Gitea 代码同样如此。Gitea
的名称和标志是其各自所有者的商标；Forgente 使用自己的名称和标志。
