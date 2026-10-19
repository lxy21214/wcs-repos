# wcs-repo 索引协议 v4

协议只描述可下载文件及其获取位置。索引维护者整理上游差异，消费者选择需要的包，Component 的 `install.sh` 决定如何安装。通用客户端不推导软件版本、不解析配对关系、不执行包内代码。

## 索引位置和身份

索引位置为 `<software>/<version>/index.json`。顶层 `software`、`version` 必须与路径相符；它们都是不透明字符串，不要求 SemVer。`3.13t.16`、`8u504` 等标识可直接使用。各路径段不能是空字符串、`.`、`..`，不能包含斜杠、反斜杠或控制字符。

`manifest.json` 和 `index-bundle.json` 是可由 `wcs-repo index` 重建的目录与汇总，不是独立维护入口。版本选择规则（例如每条 `1.x` 线取最大补丁）属于具体软件的采集过程，不属于本协议。

## 最小结构

以下仅为格式示意，地址和校验值不代表实际软件包：

```json
{
  "schema_version": 4,
  "software": "python",
  "version": "3.13t.16",
  "packages": {
    "python": {
      "version": "3.13.16",
      "filename": "python.tar.zst",
      "url": "https://example.org/python.tar.zst",
      "sha256": "0000000000000000000000000000000000000000000000000000000000000000",
      "metadata": {"freethreaded": true}
    }
  }
}
```

顶层必需字段只有 `schema_version`、`software`、`version`、`packages`。`packages` 是非空映射，键为索引内唯一的包名，例如 `python`、`python-source`、`uv`。这些是下载文件的选择名，不是框架的 Component。

| 包字段 | 要求 | 含义 |
| --- | --- | --- |
| `filename` | 必需 | 获取后的文件名，单个安全路径段 |
| `url` | 必需 | 上游 HTTPS 地址，不从版本号或文件名猜测 |
| `sha256` | 必需 | 原始文件的 SHA-256，64 位十六进制 |
| `version` | 可选 | 上游版本，原样字符串，与顶层选择标识独立 |
| `cache` | 可选 | 相对于该 `index.json` 的仓库缓存路径，如 `./python.tar.zst`，不能越出版本目录 |
| `metadata` | 可选 | 软件特有的来源、平台、构建特征或说明，客户端不据此自动安装 |

顶层也可有 `metadata`。顶层及包对象只允许本协议列出的字段，其他信息统一放进 `metadata`；通用客户端不解释其中的软件特有内容。平台、CPU、ABI、包角色不设统一枚举，也不强制每个索引都具备这些属性。

## 获取和缓存

仓库位置由调用者通过 `--repo` / `SOFTWARE_REPO_URL` 提供，不允许内置默认仓库。GitHub ref 可通过 `--ref` / `SOFTWARE_REPO_REF` 配置，默认 `main`；一次操作解析成一个具体 commit，用于读取索引与相对缓存文件。

存在 `cache` 时读取它，不存在时读取 `url`。两者应表示相同字节，由同一个 SHA-256 约束。缓存缺失、损坏或下载内容校验失败都明确报错。上游来源不因镜像缓存而丢失。

软件包只有在调用者明确选择时才下载。客户端复用校验正确的已有目标文件，先将新内容写入临时文件，校验成功后再发布正式文件；既有不同内容不被覆盖。

`cache` 操作保存上游原始文件并补充相对路径，不重打包，不改变包的版本。同一文件名不能对应不同校验和。已发布版本的包不应原地替换；索引变更通过 Git 快照追踪，消费者可固定 commit。

## 校验与格式版本

正式索引只允许 v4，CLI 不再读取 v3 的 `components` 或早期裸描述符。软件特有信息保留在 `metadata` 中；迁移不改写上游归档文件或 SHA-256。

JSON Schema 定义字段结构、类型与路径模式。CLI 使用相同的 v4 字段约束，并进一步检查实际目录身份、缓存文件、HTTPS 地址及汇总索引一致性。元数据可以自由扩展，但必须是 JSON 对象。

`check` 校验已存在的 manifest 和汇总索引；缺失时可先运行 `index` 创建。更新索引后，运行 `index` 重建汇总，再运行 `check`。
