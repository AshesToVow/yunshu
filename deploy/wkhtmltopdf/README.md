# wkhtmltopdf 离线包（可选）

巡检报告 PDF 优先通过 `wkhtmltopdf` 将 HTML 转为 PDF，与 HTML 版式一致。
未安装或转换失败时，后端自动降级为 Go 结构化 PDF。

## Docker 构建

默认构建会尝试从 Alpine 3.14 仓库安装 `wkhtmltopdf`（无需 Docker Hub）。

若内网无法访问 `dl-cdn.alpinelinux.org`，可将 Linux amd64 二进制放到本目录并命名为 **`wkhtmltopdf`**（无后缀），构建时会优先使用离线包：

```
deploy/wkhtmltopdf/wkhtmltopdf
```

### 获取二进制（在有 Docker Hub 的机器上）

```bash
docker pull surnet/alpine-wkhtmltopdf:3.19.1-0.12.6-small
docker run --rm --entrypoint cat surnet/alpine-wkhtmltopdf:3.19.1-0.12.6-small /bin/wkhtmltopdf \
  > deploy/wkhtmltopdf/wkhtmltopdf
chmod +x deploy/wkhtmltopdf/wkhtmltopdf
```

华为云镜像代理（若已同步）：

```bash
docker pull swr.cn-north-4.myhuaweicloud.com/ddn-k8s/docker.io/surnet/alpine-wkhtmltopdf:3.19.1-0.12.6-small
```

## 运行时

默认在 `PATH` 中查找 `wkhtmltopdf`。自定义路径：

```bash
INSPECT_WKHTMLTOPDF_BIN=/usr/local/bin/wkhtmltopdf
```
