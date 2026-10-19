# wcs-repo CLI

`wcs-repo` 负责读取软件索引、获取指定文件、校验 SHA-256 和维护缓存。它不解压、不安装、不构建镜像，也不根据版本名称推断软件包或依赖。安装哪些包、如何使用这些包，由调用它的 `install.sh` 决定。

程序使用 Go 标准库，无第三方运行依赖。仓库提供 `bin/linux-amd64/wcs-repo`，使用 `CGO_ENABLED=0` 编译；目标机器无需安装 Go、Python 或 Node.js。`bin/linux-amd64/SHA256SUMS` 保存二进制校验值。该二进制只适用于 Linux x86_64。

## 开始使用

在 `wcs-repos` 根目录执行：

```sh
./bin/linux-amd64/wcs-repo --help
./bin/linux-amd64/wcs-repo --repo . list
./bin/linux-amd64/wcs-repo --repo . list python
./bin/linux-amd64/wcs-repo --repo . show python 3.13t.16
./bin/linux-amd64/wcs-repo --repo . check --json
```

仓库必须通过 `--repo` 或 `SOFTWARE_REPO_URL` 显式指定，没有默认仓库。支持本地目录和 `https://github.com/<owner>/<repo>`；GitHub 分支通过 `--ref` 或 `SOFTWARE_REPO_REF` 指定，默认 `main`。

```sh
export SOFTWARE_REPO_URL=https://github.com/lxy21214/wcs-repos
export SOFTWARE_REPO_REF=main
./bin/linux-amd64/wcs-repo list python
```

远程 GitHub 操作开始时把 ref 解析成一个具体 commit，本次操作的索引及相对缓存路径均使用该 commit。`fetch --json` 返回 `ref`，多次调用需要同一快照时，将这个 commit 显式传给后续调用。上游绝对 URL 不会被重写。网络使用系统代理环境变量；目前只支持匿名访问公开 GitHub 仓库。

## 命令

| 命令 | 行为 |
| --- | --- |
| `list [software]` | 列软件，或列该软件已收录的版本。版本是原样字符串，不按 SemVer 排序、不推导 latest |
| `show <software> <version>` | 输出原始索引 JSON |
| `fetch <software> <version> --package <name> --output <dir>` | 只获取明确选中的包；可重复 `--package` |
| `cache <software> <version> --package <name>` | 只对本地仓库操作；将选中包保存到版本目录，补充相对 `cache` 字段 |
| `check` | 只对本地仓库操作；检查版本索引的结构、身份、路径、声明缓存的校验和以及已存在汇总文件的一致性，不下载 |
| `index` | 只对本地仓库操作；由各版本索引重建 `manifest.json` 和 `index-bundle.json`，不下载 |

选项可以放在命令前或后。`--json` 提供机器可读输出；下载进度与错误写入 stderr。成功返回 0，失败返回 1。CLI 不执行 Git add、commit 或 push。

### 供安装脚本下载

```sh
wcs-repo fetch python 3.13t.16 \
  --package python --package uv \
  --output /tmp/python-downloads --json
```

输出结构：

```text
/tmp/python-downloads/
├── python/<索引中的 filename>
└── uv/<索引中的 filename>
```

JSON 的 `packages` 是包名到绝对文件路径的映射。`install.sh` 使用这些路径解包、编译或安装；CLI 不解释 Python、uv、pip、CPU 或 free-threaded 的业务规则。

`fetch` 和 `cache` 必须给出至少一个 `--package`，不默认下载全部包或全部版本；`fetch` 必须给出 `--output`。所有包名在下载前检查，未知包名直接报错。

有 `cache` 字段时读取仓库缓存，否则使用 `url`。声明的缓存丢失或校验失败时直接报错，不静默改用上游。已有目标文件校验正确则复用；不同内容报错并保留。下载通过同目录临时文件完成，只有 SHA-256 匹配才发布正式文件，不覆盖并发写入者。

临时网络故障最多重试五次；404 不重试，校验失败不作为网络故障重试。传输仅使用 HTTPS。远程 GitHub 缓存自动识别 Git LFS 指针并获取实体；本地 LFS 指针会提示先运行 `git lfs pull`。

### 缓存与索引维护

```sh
wcs-repo --repo . cache chrome-devtools-mcp 1.11.0 \
  --package chrome-devtools-mcp
wcs-repo --repo . index
wcs-repo --repo . check
```

`cache` 保留包内容及上游 URL，新增 `cache: "./<filename>"`。已有同名文件只有校验匹配才复用。它不会改写软件包版本，也不会自动提交文件；需要发布时由维护者检查 Git 忽略规则、LFS 设置并自行提交。

`index` 只扫描 `<软件>/<版本>/index.json`。不递归进入任意深度，也不检查 `handoff/`、隐藏目录或 `_` 开头的目录。它不自动删除历史版本。Python CPU variant 和 Chrome 都使用标准的 `<软件>/<版本>/index.json`，与其他软件按相同规则扫描。

## 协议与现有数据

全部正式索引统一使用 [v4 协议](../protocol.md) 和 [JSON Schema](../schema/software-index-v4.schema.json)。CLI 只读取 v4；不兼容 `components` 列表或裸描述符。`show` 和 `cache` 保留合法的 `metadata`。

当前安装脚本已改为读取 v4 的 `packages`，但尚未改成调用此二进制。协议格式迁移和安装脚本的工具选择是两件事；安装逻辑仍在每个组件内。

## 编译与测试

需要 Go 1.22 或更高版本，且无需下载第三方 Go 模块：

```sh
cd cli
go test ./...
./build.sh
```

构建默认输出 `../bin/linux-amd64/wcs-repo` 和 `SHA256SUMS`。需要其他 Linux 架构时显式设置，例如 `GOARCH=arm64 ./build.sh`。构建关闭 CGO、去除宿主路径及 VCS 构建注入；当前发布版本为 0.2.0。

测试使用临时目录及模拟 HTTP 传输，不访问上游、不安装软件、不生成 Component artifact。覆盖按需选择、v4 与旧格式拒绝、缓存复用、校验失败、网络重试、路径边界、GitHub commit 固定及 LFS 读取。
