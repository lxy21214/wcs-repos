# WebCloneSuite 软件索引仓库

本仓库为 WebCloneSuite 统一管理软件版本、下载地址和 SHA-256 校验值。构建环境时，安装脚本先读取对应的索引，再从索引指定的上游地址下载软件包，校验通过后完成安装。

仓库只保存 JSON 索引与源数据文件，软件包本体不提交进仓库；唯一例外是 `chrome/` 下的 Chrome for Testing zip，通过 Git LFS 存储。本地缓存可放在 `.cache/`，该目录已被 Git 忽略。语言环境索引（schema v3）由 2026-10-15 的一次全量采集生成，覆盖 python、nodejs、rust、go、java、gcc、php、ruby、uv，数据来源见下文。

## 目录总览

| 路径 | 内容 |
| --- | --- |
| `<软件>/<版本>/index.json` | 语言环境下载索引（schema v3），共 612 个版本目录 |
| `python/<版本>/x86_64_v*.json` | Python 每 CPU 等级（v1–v4）细粒度索引，早期格式，主项目安装脚本在用 |
| `chrome/` | Chrome for Testing 镜像（Git LFS），说明见 [chrome/README.md](chrome/README.md) |
| `manifest.json` | 各软件的版本线清单与计数 |
| `schema/` | `index.json` 的 JSON Schema（`software-index-v3.schema.json`） |
| `metadata/` | 采集器输出的原始上游元数据（含全部 url + sha256），用于重新渲染或审计索引树 |
| `index-bundle.json` | 全部 `index.json` 的单文件汇总（每条含 `_path` 相对路径），适合一次性加载 |

## 语言环境索引（schema v3）

面向 Linux x86_64 的语言环境下载索引。每个软件版本一个目录，目录内的 `index.json` 直接列出该版本的整套组件（运行时、包管理器、源码、标准库等），消费者按"软件 + 版本"定位后无需再找组件。

### 布局

```text
<软件>/<版本>/index.json        例：nodejs/22.11.0/index.json、python/3.14.8-freethreaded/index.json
manifest.json                  每软件的版本列表与计数
schema/software-index-v3.schema.json
```

free-threaded Python 构建使用独立版本目录（如 `3.14.8-freethreaded`），内容同样是完整组件集（含配对 uv）。

### index.json 格式

```json
{
  "schema_version": 3,
  "software": "nodejs",
  "version": "22.11.0",
  "platform": {"os": "linux", "arch": "x86_64", "libc": "gnu"},
  "components": [
    {"name": "nodejs", "version": "22.11.0", "filename": "...", "url": "...", "sha256": "...",
     "role": "runtime", "provides": ["node", "corepack"], "bundled_components": {"npm": "10.9.0"}},
    {"name": "npm", "version": "10.9.0", "filename": "...", "url": "...", "sha256": "...",
     "role": "package-manager", "paired_with": {"nodejs": "22.11.0"}}
  ]
}
```

组件字段：`role`（runtime / package-manager / source / toolchain / installer / std）、`provides`、`format`、`abi`、`upstream`、`paired_with`、`bundled_components`、`freethreaded`。部分索引顶层另有 `release_date` 与 `install`（php/ruby/gcc 等源码构建语言为 `source build`）。

### 版本收录规则

- 范围：**2016-01-01 之后首次发布的所有 x.y 版本线**（跨 2016 年的版本线默认收录）；仅 Linux x86_64。
- 每条 x.y 线只收录**最新的 patch z**，不要求全部 x.y.z。
- 不稳定/预发布版本线也收录（如 Java 24/26、Node.js 0.x、Rust 全部 1.x）。

### 各软件覆盖

| 软件 | 版本线数 | 每版本组件 |
| --- | --- | --- |
| python | 14（+3 freethreaded） | python-source（全部）、python 二进制（python-build-standalone，3.11+）、配对 uv |
| python `X.Y.Z-freethreaded` | 3 | freethreaded python-build-standalone 构建 + 配对 uv |
| nodejs | 367 | nodejs 运行时（含 corepack，记录捆绑 npm 版本）+ 按版本配对的独立 npm tarball |
| rust | 95（1.0.0–1.99.0） | rustup-init、rust（合并包）、rustc、cargo、rust-std、rust-source |
| gcc | 52（4.9.4–16.2.0） | 源码 tarball（源码构建） |
| go | 23（1.5.4–1.27.2） | 官方 linux-amd64 工具链 tarball |
| java | 19（8u504–26.0.2.1） | JDK tarball，来源按版本线混合（见下表） |
| php | 13（5.5.38–8.5.11） | 源码 tar.xz（源码构建） |
| ruby | 13（2.1.10–4.0.7） | 源码 tar.gz（源码构建） |
| uv | 13（0.1.45–0.13.1） | manylinux x86_64 wheel |

### 数据来源与校验

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

注意事项：

- Oracle 的 `/otn/` 老链接（Java 8–16）需要登录，因此 8u/11/17 改用 Eclipse Temurin、9/10/12–16 改用 OpenJDK GA 构建，18+ 直接来自 Oracle archive；版本线覆盖完整（8u–26，含 24/26 非 LTS）。如需纯 Oracle 需带凭证重新采集。
- Node.js 的 npm 按 Node 发布日期就近选择配对版本，极旧 Node（0.x）为尽量契合版本；Node tarball 本身自带 npm，索引中的 npm 是独立组件供选用。
- v3 索引中 Python 2.7–3.10 仅有源码 + uv；3.11+ 含 python-build-standalone 二进制（x86_64_v3、pgo+lto-full）。3.9.25 与 3.10.22 的 v1–v4 二进制见下节"Python CPU 等级索引"。
- PHP/Ruby/GCC 无官方 Linux x86_64 二进制，索引只提供源码包（`install: source build`），安装器需自行编译。

## Python CPU 等级索引

每种 Python 版本目录下除 `index.json` 外，还有按 CPU 等级划分的细粒度索引：

```text
python/
├── 3.9.25/
├── 3.10.22/
├── 3.11.17/
├── 3.12.15/
├── 3.13.16/
│   ├── index.json
│   ├── x86_64_v1-unknown-linux-gnu-pgo+lto-full.json
│   ├── x86_64_v2-unknown-linux-gnu-pgo+lto-full.json
│   ├── x86_64_v3-unknown-linux-gnu-pgo+lto-full.json
│   └── x86_64_v4-unknown-linux-gnu-pgo+lto-full.json
└── 3.14.8/
```

每个 CPU 等级 JSON 文件包含三个字段：

| 字段 | 含义 |
| --- | --- |
| `filename` | 下载到本地时使用的标准文件名 |
| `url` | 指向具体软件包的 HTTPS 下载地址 |
| `sha256` | 软件包的完整 SHA-256，64 位十六进制字符串 |

例如 [Python 3.13.16 / v3](python/3.13.16/x86_64_v3-unknown-linux-gnu-pgo+lto-full.json)：

```json
{
  "filename": "cpython-3.13.16+20261013-x86_64_v3-unknown-linux-gnu-pgo+lto-full.tar.zst",
  "url": "https://github.com/astral-sh/python-build-standalone/releases/download/20261013/cpython-3.13.16%2B20261013-x86_64_v3-unknown-linux-gnu-pgo%2Blto-full.tar.zst",
  "sha256": "3d621f22b88b95211ebde067b65e597bc5988e2e4f703cbb611618e91b386582"
}
```

这些索引指向 [python-build-standalone Releases](https://github.com/astral-sh/python-build-standalone/releases) 中的 x86_64 Linux GNU 构建，采用 `pgo+lto-full.tar.zst` 格式：

| Python 版本 | 上游 Release | CPU 等级 |
| --- | --- | --- |
| 3.9.25 | 20251031 | v1、v2、v3、v4 |
| 3.10.22 | 20261009 | v1、v2、v3、v4 |
| 3.11.17 | 20261013 | v1、v2、v3、v4 |
| 3.12.15 | 20261013 | v1、v2、v3、v4 |
| 3.13.16 | 20261013 | v1、v2、v3、v4 |
| 3.14.8 | 20261013 | v1、v2、v3、v4 |

上游基础 x86_64 包的文件名没有 `_v1`；索引中的 `filename` 补齐 `_v1`，`url` 保留真实的上游路径。这样可以统一选择 CPU 等级，同时保持上游软件包内容不变。下载 URL 和 SHA-256 均核对自对应 GitHub Release 的 asset 元数据（`browser_download_url` 和 `digest`）。

## Chrome for Testing

`chrome/` 镜像了 Chrome for Testing 的 linux-x64 构建本体（Git LFS 存储），并附指向固定提交的下载索引，详见 [chrome/README.md](chrome/README.md)。

## 在 WebCloneSuite 中使用

主项目在 `src/components/buildable/devlang/python/arguments/<variant-id>.json` 中配置脚本变量。例如 `3.13.16-x86_64_v3.json`：

```json
{
  "SOFTWARE_REPO_URL": "https://github.com/lxy21214/wcs-repos",
  "SOFTWARE_REPO_REF": "main",
  "PYTHON_PACKAGE": "python/3.13.16/x86_64_v3-unknown-linux-gnu-pgo+lto-full.json",
  "PYTHON_INSTALL_PREFIX": "/opt/python/3.13.16-x86_64_v3",
  "PYTHON_BIN_DIR": "/usr/local/bin"
}
```

这些键值直接作为变量传入 install、setup、test。devlang family 不规定版本号、CPU 等级或其他语言属性，只检查变量能否安全传入 Shell。变量名须为合法的 Shell 名称，变量值须为不含控制字符的字符串。

`PYTHON_PACKAGE` 是本索引仓库内的相对路径。安装脚本不拆解其中的版本或架构，通过 `raw.githubusercontent.com` 读取该文件，再下载其中的 `url` 并验证 `sha256`。`SOFTWARE_REPO_REF` 可以指定分支、标签或提交 SHA；需要固定索引内容时使用提交 SHA。当前脚本接受仅包含字母、数字、点、下划线和连字符的 ref。

在 WebCloneSuite 主项目中生成并构建：

```bash
node scripts/tests/render-buildable.cjs devlang python 3.13.16-x86_64_v3
# 使用上一步输出的产物目录：
docker build /path/to/component/artifact/component-fingerprint
```

省略最后的 variant ID 会生成 `arguments/` 下的所有配置。脚本变量已写入生成的构建上下文，无需额外传入 Docker build args。安装脚本使用 `jq` 解析软件包索引，构建 Blueprint 会安装该依赖。

本仓库只负责软件下载信息。Component、Blueprint、Recipe 和安装、配置、测试脚本由 WebCloneSuite 主项目维护；渲染出的构建产物保存在对应组件的 `artifact/` 下。系统基础依赖仍可通过镜像的包管理器安装。

## 维护约定

- 添加版本时创建对应 JSON 文件，使用固定版本的 HTTPS 地址，并从可信上游核实 SHA-256。相同构建变体的新版本无需修改安装脚本中的版本映射。
- 不将软件包提交到仓库（`chrome/` 的 LFS 镜像除外）。本地缓存可放在 `.cache/`，该目录已被 Git 忽略。
- 当前安装流程不包含私有仓库认证，索引和上游软件包需要能够匿名下载。
- 固定索引提交和 SHA-256 可以锁定下载内容，但不能保证上游文件一直存在。需要自行保存文件时，可以将索引中的 URL 改为自有存储地址。
- `metadata/` 保存采集器输出的原始条目（各软件 `*_items.json`，含全部 url + sha256，以及公共库 `wcs_common.py`）。采集脚本本身未保留，重新渲染索引树或补采时以 `metadata/` 和上文数据来源表为准。
- `index-bundle.json` 是索引树的生成物，修改任何 `index.json` 后需重新生成（或删除以免过期）。生成方式：

```bash
python3 - <<'PY'
import json, os
SW = ('gcc', 'go', 'java', 'nodejs', 'php', 'python', 'ruby', 'rust', 'uv')
indexes = []
for sw in SW:
    for ver in sorted(os.listdir(sw),
                      key=lambda v: (v.removesuffix('-freethreaded'),
                                     not v.endswith('-freethreaded'))):
        p = os.path.join(sw, ver, 'index.json')
        if os.path.isfile(p):
            d = json.load(open(p))
            d['_path'] = p
            indexes.append(d)
json.dump({'schema_version': 3, 'generated': '2026-10-15',
           'rule': 'all x.y lines since 2016-01-01, latest patch per line; linux x86_64',
           'indexes': indexes},
          open('index-bundle.json', 'w'), sort_keys=True, indent=1)
PY
```
