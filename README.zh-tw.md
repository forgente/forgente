<div align="center">
  <img src="public/assets/img/logo.svg" alt="Forgente" width="96"/>

# Forgente

[![](https://github.com/forgente/forgente/actions/workflows/release-nightly.yml/badge.svg?branch=main)](https://github.com/forgente/forgente/actions/workflows/release-nightly.yml?query=branch%3Amain "Release Nightly")
[![](https://img.shields.io/badge/License-MIT-blue.svg)](https://opensource.org/licenses/MIT "License: MIT")

**您完全擁有的完整軟體協作平台。**

</div>

[English](./README.md) | [简体中文](./README.zh-cn.md)

Forgente 是一個一體化的軟體開發服務：Git 代管、程式碼審查、問題追蹤、專案看板、
Wiki、套件登錄檔，以及與 GitHub Actions 工作流程相容的 CI/CD。使用 Go 撰寫，可作
為單一二進位檔案運行在 Linux、macOS、FreeBSD/OpenBSD 和 Windows 上──執行在您自
己的硬體上，由您掌控，沒有遙測。

Forgente 的發展方向請見 [ROADMAP.md](ROADMAP.md)；它的建置與發佈方式記錄在
[FORGENTE.md](FORGENTE.md) 中。

## 安裝

**容器**（建議）：

```bash
docker run -p 3000:3000 -p 2222:22 forgente/forgente:latest
```

映像檔發佈於 [Docker Hub](https://hub.docker.com/r/forgente/forgente) 與
[GHCR](https://github.com/forgente/forgente/pkgs/container/forgente)，標籤包含每
次發佈對應的 `latest`、`<major>`、`<major.minor>`、`<version>`，以及
`main-nightly` 開發建置版本（皆提供 `-rootless` 變體）。既有的 Gitea 容器設定（資
料卷、`GITEA_*` 環境變數）無需更動即可繼續使用。

**二進位檔案**：每個平台的簽章建置版本會附加在
[GitHub Releases](https://github.com/forgente/forgente/releases) 中，並鏡像至
[dl.forgente.com](https://dl.forgente.com/forgente/)（每夜建置版本位於
[`main-nightly`](https://dl.forgente.com/forgente/main-nightly/)）──每個檔案都附
有 SHA-256 校驗碼、GPG 簽章（金鑰
`67129BAD57A2C8D2186032489D6FD2FD6E0B9BA5`）以及 sigstore 簽章包。

**Snap**：

```bash
sudo snap install forgente --edge
```

## 從原始碼建置

請參閱 [docs/build-setup.md](docs/build-setup.md) 了解前置需求，以及
[docs/development.md](docs/development.md) 了解開發環境。

```bash
TAGS="bindata" make build
./forgente web
```

## 文件

文件位於 [docs.forgente.com](https://docs.forgente.com)。Forgente 特有的行為與維
運說明記錄在 [FORGENTE.md](FORGENTE.md) 中。

## 基於 Gitea 建置

Forgente 基於 [Gitea](https://github.com/go-gitea/gitea) 建置，並將其作為上游持
續追蹤：Gitea 的改進與安全修復會持續合併進來，同時 Forgente 在此基礎上建置自己的
功能，並與 Gitea 保持設定與 API 相容。非常感謝 Gitea 的維護者與貢獻者。追蹤機制的
細節記錄在 [FORGENTE.md](FORGENTE.md) 中。

## 貢獻

Fork → 修改 → 推送 → 發起 Pull Request。請先閱讀
[貢獻者指南](CONTRIBUTING.md)。安全性問題請私下寫信至 security@forgente.com。

## 授權條款

Forgente 依據 [MIT 授權條款](LICENSE) 授權，其所基於的 Gitea 程式碼亦同。Gitea
的名稱與標誌是其各自所有者的商標；Forgente 使用自己的名稱與標誌。
