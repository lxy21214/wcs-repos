# WebCloneSuite 软件索引仓库

本仓库为 WebCloneSuite 整理跨软件生态的构建输入：具体版本的软件包、上游下载地址、SHA-256，以及按需保存的原始文件缓存。所有正式软件索引统一使用 **v4**；安装操作由主项目各 Component 的 `install.sh` 决定。

索引与包名是明确的数据，不要求版本符合 SemVer，也不根据版本、CPU 或平台名称猜测下载链接。`python/3.13t.16`、`java/8u504` 都是合法选择。

## 目录与工具

| 路径 | 内容 |
| --- | --- |
| `<软件>/<版本>/index.json` | 649 份 v4 软件索引 |
| `manifest.json` | 生成的软件版本目录与计数 |
| `index-bundle.json` | 生成的全部索引汇总，每项包含 `_path` |
| `protocol.md` | v4 协议 |
| `schema/software-index-v4.schema.json` | v4 JSON Schema |
| `cli/` | wcs-repo 的 Go 源码、测试和构建脚本 |
| `bin/linux-amd64/` | 静态 CLI 二进制和 SHA256SUMS |
| `metadata/` | 历史上游采集记录与辅助代码，不是消费者读取的正式索引 |
| `chrome/` | Chrome for Testing 索引及 Git LFS 缓存 |
| `chrome-devtools-mcp/` | MCP 索引及原始 npm 包缓存 |

`wcs-repo` 只读取 v4，提供 `list / show / fetch / cache / check / index`，使用方式见 [cli/README.md](cli/README.md)。运行时无需安装 Go、Python 或 Node.js。

```sh
./bin/linux-amd64/wcs-repo --repo . list python
./bin/linux-amd64/wcs-repo --repo . show python 3.13t.16
./bin/linux-amd64/wcs-repo --repo . check --json
```

仓库地址必须显式提供，可以是本地目录，或者通过 `--repo` / `SOFTWARE_REPO_URL` 指定 GitHub 仓库；`--ref` / `SOFTWARE_REPO_REF` 默认 `main`。需要固定构建输入时使用 commit。

## 索引结构

顶层使用 `schema_version`、`software`、`version`、`packages`。`packages` 按包名索引，每项必需 `filename`、`url`、`sha256`，可选 `version`、`cache`、`metadata`。软件特有的 `role`、`format`、平台、ABI、配对关系等记录在 `metadata`，不设通用枚举。

`url` 始终记录上游来源，`cache` 是相对于该索引的仓库内路径。Chrome 和 MCP 已显式记录缓存；CLI 优先使用声明的缓存。缓存缺失或内容不匹配时明确失败，不静默换源。所有现有归档文件保持原样。

## 当前收录

| 软件 | 索引数 | 说明 |
| --- | ---: | --- |
| python | 41 | 17 个基础/free-threaded 索引，加 24 个 CPU variant；每个都包含配对 uv |
| nodejs | 367 | Node.js 及配对 npm |
| rust | 95 | 固定版本工具链及其他可选包 |
| gcc | 52 | 源码包 |
| go | 23 | 官方工具链 |
| java | 19 | JDK |
| php | 13 | 源码包 |
| ruby | 13 | 源码包 |
| uv | 13 | uv 包的独立下载资料，不决定安装过程如何组合 |
| chrome | 1 | 固定版本 Chrome for Testing |
| chrome-devtools-mcp | 12 | 所有稳定 `1.x` 线各取最大补丁版本 |

语言基础版本最初采集于 2026-10-15，MCP 采集于 2026-10-19。收录索引不表示所有历史版本都已在当前基础镜像上构建验证。

## Python 版本与 CPU variant

free-threaded 选择标识使用 `X.Yt.Z`，例如 `3.13t.16`；包的 `version` 保留上游版本 `3.13.16`，构建特征记录于 `metadata`。不改写上游归档文件名、下载路径或校验和。

原有 24 个零散 CPU JSON 已迁移为标准版本目录，例如：

```text
python/3.13.16-x86_64_v3/index.json
```

这些索引保留原 CPU 构建的 Python 包，并收录基础版本索引中已经配对的 uv。Python 的安装脚本明确选择所需 Python 包与 uv；CLI 不解析 `x86_64_v3`，不推断版本配对。

## 数据来源

| 软件 | 下载来源 | sha256 来源 |
| --- | --- | --- |
| nodejs / npm | nodejs.org/dist + npm registry | 官方 SHASUMS256 / npm tarball 实测 |
| go | go.dev/dl JSON（URL 指向 dl.google.com/go/） | 官方 sha256 |
| rust | static.rust-lang.org/dist | 官方 .sha256 sidecar |
| php | php.net/releases JSON API + 归档页 | 官方 sha256 |
| ruby | ruby-lang.org 各 release 新闻页 | 官方 sha256 |
| python | python.org 发布页 + astral-sh/python-build-standalone release 20261013 | 发布页（3.10+）/ 源码包实测（2.7–3.9）/ PBS SHA256SUMS |
| uv | PyPI JSON | 官方 digest |
| java 18–26 | Oracle archive（download.oracle.com/java/N/archive/） | 官方 .sha256 |
| java 8u / 11 / 17 | Eclipse Temurin（Adoptium API） | 官方 checksum |
| java 9 / 10 / 12–16 | OpenJDK GA 构建（jdk.java.net/archive） | 官方 .sha256 |
| gcc | ftp.gnu.org（经清华 tuna 镜像下载实测校验，URL 保留官方） | 实测下载计算 |


Chrome 的来源与缓存说明见 [chrome/README.md](chrome/README.md)，MCP 见 [chrome-devtools-mcp/README.md](chrome-devtools-mcp/README.md)。历史采集记录保留原始字段作为来源证据，不作为 v4 安装输入。

## 使用与维护

```sh
# 只下载明确选择的包；不下载同索引内的其他包，也不执行安装。
./bin/linux-amd64/wcs-repo --repo . fetch python 3.13t.16 \
  --package python --package uv --output /tmp/python-downloads --json

# 修改版本索引或缓存记录后重建两个汇总文件，再校验。
./bin/linux-amd64/wcs-repo --repo . index
./bin/linux-amd64/wcs-repo --repo . check
```

唯一维护入口是各版本的 `index.json`；不手工同步 manifest 和汇总索引。`check` 检查协议字段、目录身份、相对路径、声明缓存的 SHA-256，以及已存在汇总文件的一致性。

主项目的 `arguments/<variant>.sh` 显式设置 `SOFTWARE_INDEX=<软件>/<版本>/index.json`，由容器内安装脚本读取；渲染器只复制脚本，不解析其中的变量。多 variant 的生成必须明确选择，不批量生成 artifact 或镜像。

缓存软件包只按明确需求保存。Chrome 使用 Git LFS，克隆后需 `git lfs pull` 获取实体；MCP 使用普通 Git。未声明的临时文件放入 `.cache/`。CLI 不自动提交或 push，发布时需同时提交索引及其引用的缓存文件。
