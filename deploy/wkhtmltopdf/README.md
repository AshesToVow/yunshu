# wkhtmltopdf 离线包（推荐）

巡检报告 PDF 通过 **静态链接** 的 `wkhtmltopdf` 将 HTML 转为 PDF。

**切勿** 在 Alpine 3.19 上用 `apk add wkhtmltopdf`（会拉取 3.14 动态库，运行时报错并崩溃）：

```
Cannot mix incompatible Qt library (5.15.10) with this library (5.15.3)
```

## Docker 构建（默认）

`Dockerfile.backend` 从华为云代理拉取静态二进制：

`swr.cn-north-4.myhuaweicloud.com/ddn-k8s/docker.io/surnet/alpine-wkhtmltopdf:3.23.2-0.12.6-full`

（`full` 含 `libwkhtmltox` 动态库；若改用 `...-small` 静态单文件版，可去掉 `/lib` 拷贝。）

构建时会做 **smoke test**（真实生成 PDF），避免 `--version` 通过但转换崩溃。

## 离线包（代理不可用时）

将 Linux amd64 静态二进制放到：

```
deploy/wkhtmltopdf/wkhtmltopdf
```

构建时若存在离线包，会覆盖镜像拷贝的版本。

### 导出静态二进制

```bash
docker pull swr.cn-north-4.myhuaweicloud.com/ddn-k8s/docker.io/surnet/alpine-wkhtmltopdf:3.23.2-0.12.6-full
docker run --rm --entrypoint cat \
  swr.cn-north-4.myhuaweicloud.com/ddn-k8s/docker.io/surnet/alpine-wkhtmltopdf:3.23.2-0.12.6-full \
  /bin/wkhtmltopdf > deploy/wkhtmltopdf/wkhtmltopdf
chmod +x deploy/wkhtmltopdf/wkhtmltopdf
```

## 已部署环境热修复（无需立刻重建镜像）

```bash
# 在能拉镜像的机器上导出后 scp 到 k8s-master，或直接在 master 上执行 pull
docker run --rm --entrypoint cat \
  swr.cn-north-4.myhuaweicloud.com/ddn-k8s/docker.io/surnet/alpine-wkhtmltopdf:3.23.2-0.12.6-full \
  /bin/wkhtmltopdf > /tmp/wkhtmltopdf

docker cp /tmp/wkhtmltopdf yunshu-backend:/usr/local/bin/wkhtmltopdf
# full 版还需库文件（若热修复后仍报错）：
docker run --rm -v /tmp/wkhtml-lib:/out \
  swr.cn-north-4.myhuaweicloud.com/ddn-k8s/docker.io/surnet/alpine-wkhtmltopdf:3.23.2-0.12.6-full \
  sh -c 'cp /lib/libwkhtmltox* /out/'
docker cp /tmp/wkhtml-lib/. yunshu-backend:/usr/local/lib/
docker-compose exec backend sh -c 'chmod +x /usr/local/bin/wkhtmltopdf && rm -f /usr/bin/wkhtmltopdf'
docker-compose exec backend sh -c 'wkhtmltopdf --version && echo test | wkhtmltopdf --quiet - - 2>/dev/null | head -c 4'
# 应输出 %PDF
```

然后 **重新执行巡检** 生成新 PDF。

## 运行时

```bash
INSPECT_WKHTMLTOPDF_BIN=/usr/local/bin/wkhtmltopdf
```
