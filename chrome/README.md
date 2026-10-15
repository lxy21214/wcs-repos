# Chrome 固定版本镜像

这里同时保存 Chrome 软件包和下载索引，用于固定构建输入。与 Python 的上游索引不同，Chrome 的下载地址指向本仓库保存的文件。

当前收录 Chrome for Testing Stable **155.0.8059.79**，Linux x86_64，查询日期为 2026-10-15。Chrome for Testing 不自动更新，适用于本项目的浏览器自动化环境。

- [下载索引](155.0.8059.79/x86_64-unknown-linux-gnu.json)
- [本仓库固定提交中的安装包](https://media.githubusercontent.com/media/lxy21214/wcs-repos/f917716e8be62474ad65f65070bdd12e524618b2/chrome/155.0.8059.79/chrome-for-testing-155.0.8059.79-linux-x86_64.zip)
- [官方稳定版清单](https://googlechromelabs.github.io/chrome-for-testing/last-known-good-versions-with-downloads.json)
- [原始下载地址](https://storage.googleapis.com/chrome-for-testing-public/155.0.8059.79/linux64/chrome-linux64.zip)

索引格式仍为 `filename`、`url`、`sha256`。安装包保持上游 ZIP 内容不变，仅使用包含精确版本和架构的本地文件名。

文件大小：198413482 字节。SHA-256：

```text
bcd9a79130d7e44d70f93be87042e9bcee7fe0f0b3898e06b8c9bed5fb619dcd
```

下载时已核对 Google 存储返回的 MD5（`10ca99188ddcd9a967711d65ceb08b7a`）、ZIP 各文件 CRC，以及 Chrome 程序的 ELF64 x86_64 标识。未运行或安装浏览器。

## 大文件存储

ZIP 超过 GitHub 普通 Git 的 100 MiB 文件限制，使用 Git LFS 保存。克隆仓库时需要安装 Git LFS 并运行 `git lfs pull`，才能得到 ZIP 实体文件。索引使用 `media.githubusercontent.com` 返回软件包实体，避免把 Git LFS 指针当作安装包。

URL 固定到保存软件包的提交 `f917716e8be62474ad65f65070bdd12e524618b2`，不使用 `main` 或 `latest`。后续升级时新增版本目录与索引，不覆盖这一版本。
