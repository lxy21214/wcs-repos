# Chrome 固定版本镜像

这里同时保存 Chrome 软件包和下载索引，用于固定构建输入。v4 索引以 `url` 记录上游地址，以 `cache` 记录仓库内缓存。

当前收录 Chrome for Testing Stable **155.0.8059.79**，Linux x86_64，查询日期为 2026-10-15。Chrome for Testing 不自动更新，适用于本项目的浏览器自动化环境。

- [下载索引](155.0.8059.79/index.json)
- [本仓库固定提交中的安装包](https://media.githubusercontent.com/media/lxy21214/wcs-repos/f917716e8be62474ad65f65070bdd12e524618b2/chrome/155.0.8059.79/chrome-for-testing-155.0.8059.79-linux-x86_64.zip)
- [官方稳定版清单](https://googlechromelabs.github.io/chrome-for-testing/last-known-good-versions-with-downloads.json)
- [原始下载地址](https://storage.googleapis.com/chrome-for-testing-public/155.0.8059.79/linux64/chrome-linux64.zip)

索引采用 v4 的 `packages.chrome`，包含 `filename`、`url`、`sha256` 和相对 `cache`。安装包保持上游 ZIP 内容不变，仅使用包含精确版本和架构的本地文件名。

文件大小：198413482 字节。SHA-256：

```text
bcd9a79130d7e44d70f93be87042e9bcee7fe0f0b3898e06b8c9bed5fb619dcd
```

下载时已核对 Google 存储返回的 MD5（`10ca99188ddcd9a967711d65ceb08b7a`）、ZIP 各文件 CRC，以及 Chrome 程序的 ELF64 x86_64 标识。未运行或安装浏览器。

## 大文件存储

ZIP 超过 GitHub 普通 Git 的 100 MiB 文件限制，使用 Git LFS 保存。克隆仓库时需要安装 Git LFS 并运行 `git lfs pull`，才能得到 ZIP 实体文件。CLI 访问远程 GitHub 缓存时识别 LFS 指针，并通过 `media.githubusercontent.com` 获取实体。

历史缓存 URL 保存在包的 `metadata.mirror_url` 中。CLI 使用调用者选择的仓库快照读取相对缓存；需要固定快照时显式指定 commit。后续升级时新增版本目录与索引，不覆盖这一版本。
