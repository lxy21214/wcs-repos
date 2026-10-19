# chrome-devtools-mcp 固定版本缓存

从 [官方 Git tags](https://github.com/ChromeDevTools/chrome-devtools-mcp/tags) 选择版本：仅收录稳定的 `1.x.y`，每个 `x` 保留数值最大的 `y`，不收录预发布版本。本次采集日期为 2026-10-19。

缓存的是对应 tag 的官方 npm 发布包，包含编译后的 JavaScript 入口。每个版本目录保存原始 `.tgz` 和 schema v4 `index.json`，不重新打包、不执行安装脚本。软件包使用普通 Git 保存。

索引 `url` 保留官方 npm 下载地址，`cache` 显式指向本版本目录内的原始 `.tgz` 文件。CLI 在本地仓库直接读取该文件，访问远程仓库时按选定 commit 获取缓存。远程使用需要先发布对应索引和缓存；后续不得覆盖已有版本的包。

## 收录版本

- [1.0.1](1.0.1/index.json)
- [1.1.1](1.1.1/index.json)
- [1.2.0](1.2.0/index.json)
- [1.3.0](1.3.0/index.json)
- [1.4.0](1.4.0/index.json)
- [1.5.0](1.5.0/index.json)
- [1.6.0](1.6.0/index.json)
- [1.7.0](1.7.0/index.json)
- [1.8.0](1.8.0/index.json)
- [1.9.0](1.9.0/index.json)
- [1.10.1](1.10.1/index.json)
- [1.11.0](1.11.0/index.json)

## 校验与使用

- 已校验 npm 官方 `dist.integrity`（SHA-512）和 `dist.shasum`（SHA-1），另计算 SHA-256 写入索引。
- npm 的 `gitHead` 与相应 Git tag 提交逐一匹配；已检查包内 `package.json` 的名称、版本及命令入口存在。
- 包的 `metadata.engines.node` 记录对应包的 Node.js 要求；Node.js 和 Chrome 由消费方另行提供，未在本次缓存中安装或构建它们。
- 索引顶层 `metadata.platform` 表示本项目的 Linux x86_64 使用目标；缓存仍是上游原始 npm 包。
- 来源、完整的已观察 1.x tags 和选中包的发布元数据保存在 [.metadata](../.metadata/chrome-devtools-mcp_items.json)。

每条版本线新增补丁时重新按 tags 筛选；是否替换旧缓存需显式执行，不自动更新安装环境。
