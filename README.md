# WebCloneSuite 软件索引仓库

本仓库为 WebCloneSuite 统一管理软件版本、下载地址和 SHA-256 校验值。构建环境时，安装脚本先读取对应的索引，再从索引指定的上游地址下载软件包，校验通过后完成安装。

仓库只保存小型 JSON 文件，无需提前下载、重新上传或通过 Git LFS 存储全部软件包。Python 是第一个收录的软件，后续可以加入 Node.js 等软件的索引。

## 目录与索引格式

每种软件使用一个顶层目录，再按精确版本和构建变体组织：

```text
python/
├── 3.9.25/
├── 3.10.22/
├── 3.11.17/
├── 3.12.15/
├── 3.13.16/
│   ├── x86_64_v1-unknown-linux-gnu-pgo+lto-full.json
│   ├── x86_64_v2-unknown-linux-gnu-pgo+lto-full.json
│   ├── x86_64_v3-unknown-linux-gnu-pgo+lto-full.json
│   └── x86_64_v4-unknown-linux-gnu-pgo+lto-full.json
└── 3.14.8/
```

每个版本目录都有 v1–v4 四个索引，共 24 个。每个 JSON 文件包含三个字段：

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

上游基础 x86_64 包的文件名没有 `_v1`；索引中的 `filename` 补齐 `_v1`，`url` 保留真实的上游路径。这样可以统一选择 CPU 等级，同时保持上游软件包内容不变。

## 当前 Python 版本

这些索引指向 [python-build-standalone Releases](https://github.com/astral-sh/python-build-standalone/releases) 中的 x86_64 Linux GNU 构建，采用 `pgo+lto-full.tar.zst` 格式。版本沿用本项目已选定的清单，不会自动跟随上游更新。

| Python 版本 | 上游 Release | CPU 等级 |
| --- | --- | --- |
| 3.9.25 | 20251031 | v1、v2、v3、v4 |
| 3.10.22 | 20261009 | v1、v2、v3、v4 |
| 3.11.17 | 20261013 | v1、v2、v3、v4 |
| 3.12.15 | 20261013 | v1、v2、v3、v4 |
| 3.13.16 | 20261013 | v1、v2、v3、v4 |
| 3.14.8 | 20261013 | v1、v2、v3、v4 |

下载 URL 和 SHA-256 均核对自对应 GitHub Release 的 asset 元数据（`browser_download_url` 和 `digest`）。索引收录不代表所有版本都已在本地安装测试。

## 在 WebCloneSuite 中使用

```bash
export SOFTWARE_REPO_URL='https://github.com/lxy21214/wcs-repos'
export SOFTWARE_REPO_REF='main'
```

`SOFTWARE_REPO_REF` 可以指定分支、标签或提交 SHA，默认是 `main`。需要固定索引内容时使用提交 SHA；当前安装脚本接受仅包含字母、数字、点、下划线和连字符的 ref。

Docker 构建时传入以上配置：

```bash
docker build \
  --build-arg SOFTWARE_REPO_URL \
  --build-arg SOFTWARE_REPO_REF \
  /path/to/component/artifact/component-fingerprint
```

Python 安装脚本通过 `raw.githubusercontent.com` 获取 `python/<精确版本>/x86_64_<CPU等级>-unknown-linux-gnu-pgo+lto-full.json`，再下载其中的 `url` 并验证 `sha256`。当前默认 CPU 等级为 v3；直接运行安装脚本时可通过 `PYTHON_CPU_LEVEL` 选择 v1–v4。安装脚本使用 `jq` 解析 JSON，构建 Blueprint 会安装该依赖。

本仓库只负责软件下载信息。Component、Blueprint、Recipe 和安装、配置、测试脚本由 WebCloneSuite 主项目维护；渲染出的构建产物保存在对应组件的 `artifact/` 下。系统基础依赖仍可通过镜像的包管理器安装。

## 维护约定

- 添加版本时创建对应 JSON 文件，使用固定版本的 HTTPS 地址，并从可信上游核实 SHA-256。相同构建变体的新版本无需修改安装脚本中的版本映射。
- 不将软件包提交到仓库。本地缓存可放在 `.cache/`，该目录已被 Git 忽略。
- 当前安装流程不包含私有仓库认证，索引和上游软件包需要能够匿名下载。
- 固定索引提交和 SHA-256 可以锁定下载内容，但不能保证上游文件一直存在。需要自行保存文件时，可以将索引中的 URL 改为自有存储地址。
