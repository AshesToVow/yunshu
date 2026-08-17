#!/usr/bin/env python
# -*- coding: utf-8 -*-
"""
将 Kubernetes Pod 同步注册到 Consul（供 Prometheus Consul SD / Yunshu 监控对象使用）。

兼容：Python 2.7 / 3.x，仅标准库 + 本机 kubectl。

用法：
  export CONSUL_TOKEN=<metrics-register SecretID>
  export KUBECONFIG=/path/to/kubeconfig   # 或配置里写 kubeconfig
  python consul_k8s_pods_sync.py -c consul-k8s-pods.json sync
  python consul_k8s_pods_sync.py -c consul-k8s-pods.json list
  python consul_k8s_pods_sync.py -c consul-k8s-pods.json deregister

建议 cron 每 1～2 分钟在监控机执行 sync。
"""

from __future__ import print_function

import argparse
import json
import os
import re
import subprocess
import sys

try:
    from urllib.request import Request, urlopen
    from urllib.error import HTTPError, URLError
except ImportError:
    from urllib2 import Request, urlopen, HTTPError, URLError  # type: ignore

MANAGED_BY = u"yunshu-k8s-pods"


def to_text(b):
    if b is None:
        return u""
    if isinstance(b, type(u"")):
        return b
    try:
        return b.decode("utf-8")
    except Exception:
        try:
            return b.decode("latin-1")
        except Exception:
            return type(u"")("%s" % (b,))


def ensure_text(v):
    if v is None:
        return u""
    if isinstance(v, type(u"")):
        return v
    if isinstance(v, type(b"")):
        return to_text(v)
    try:
        return type(u"")("%s" % (v,))
    except Exception:
        return to_text(repr(v))


def to_native_str(s):
    t = ensure_text(s)
    if sys.version_info[0] < 3:
        return t.encode("utf-8")
    return t


def die(msg, code=1):
    print("ERROR: %s" % ensure_text(msg), file=sys.stderr)
    sys.exit(code)


def load_config(path):
    try:
        import io

        with io.open(path, "r", encoding="utf-8") as f:
            raw = f.read()
    except TypeError:
        import codecs

        with codecs.open(path, "r", "utf-8") as f:
            raw = f.read()
    # 去掉整行 # 注释（勿在字符串值里写未转义的 "）
    lines = []
    for i, line in enumerate(raw.splitlines(), 1):
        s = line
        # 仅去掉行首空白后以 # 开头的整行注释
        if s.lstrip().startswith("#"):
            continue
        lines.append(s)
    text = u"\n".join(lines)
    try:
        return json.loads(text)
    except ValueError as e:
        # 帮助定位：打印出错行附近
        msg = ensure_text(e)
        print("ERROR: invalid JSON in %s: %s" % (path, msg), file=sys.stderr)
        print("---- file preview (with line numbers) ----", file=sys.stderr)
        for i, line in enumerate(text.splitlines(), 1):
            if i <= 12 or (i >= 1 and abs(i - 5) <= 2):
                mark = ">>" if i == 5 else "  "
                print("%s %3d: %s" % (mark, i, line), file=sys.stderr)
        print(
            "Hint: JSON 不能写 // 或行尾注释；字符串用双引号；"
            "属性之间要有逗号；不要多余逗号。token 行常见错误："
            "\"token\": \"uuid\", 后面又跟了字/注释。",
            file=sys.stderr,
        )
        sys.exit(1)


def sanitize_id(s):
    return re.sub(r"[^A-Za-z0-9._-]", "-", ensure_text(s))


def consul_token(cfg):
    consul = cfg.get("consul") or {}
    if consul.get("token"):
        return ensure_text(consul["token"]).strip()
    env_name = consul.get("token_env") or "CONSUL_TOKEN"
    tok = (os.environ.get(env_name) or "").strip()
    if not tok:
        die("missing token: set %s or consul.token" % env_name)
    return tok


def consul_request(addr, token, method, path, body=None):
    url = to_native_str(ensure_text(addr).rstrip("/") + ensure_text(path))
    data = None
    headers = {to_native_str("X-Consul-Token"): to_native_str(token)}
    if body is not None:
        payload = json.dumps(body, ensure_ascii=True)
        data = to_native_str(payload)
        headers[to_native_str("Content-Type")] = to_native_str("application/json")
    req = Request(url, data=data, headers=headers)
    req.get_method = lambda: method
    try:
        resp = urlopen(req, timeout=30)
        raw = resp.read()
        return resp.getcode(), to_text(raw)
    except HTTPError as e:
        raw = e.read()
        return e.code, to_text(raw) if raw else ensure_text(e)
    except URLError as e:
        return 0, ensure_text(e.reason if hasattr(e, "reason") else e)


def meta_defaults(cfg, cluster_cfg):
    d = dict(cfg.get("defaults") or {})
    d.update(cluster_cfg.get("meta") or {})
    out = {}
    for k, v in d.items():
        out[ensure_text(k)] = ensure_text(v)
    return out


def run_kubectl(cfg, args, cluster_cfg=None):
    kcfg = dict(cfg.get("kubectl") or {})
    if cluster_cfg and cluster_cfg.get("kubectl"):
        kcfg.update(cluster_cfg.get("kubectl") or {})
    bin_path = ensure_text(kcfg.get("bin") or "kubectl").strip() or "kubectl"
    cmd = [bin_path]
    kubeconfig = ensure_text(kcfg.get("kubeconfig") or os.environ.get("KUBECONFIG") or "").strip()
    if kubeconfig:
        cmd.extend(["--kubeconfig", kubeconfig])
    context = ensure_text(kcfg.get("context") or "").strip()
    if context:
        cmd.extend(["--context", context])
    cmd.extend(args)
    try:
        out = subprocess.check_output(cmd, stderr=subprocess.STDOUT)
    except subprocess.CalledProcessError as e:
        die("kubectl failed: %s\n%s" % (" ".join(cmd), to_text(e.output)))
    except OSError as e:
        die("kubectl not found (%s): %s" % (bin_path, e))
    return json.loads(to_text(out))


def pick_port(pod, cluster_cfg):
    """
    端口（写入 Consul Address 旁的 Port，便于 UI 显示）：
      1) 注解 prometheus.io/port / yunshu.io/metrics-port（显式配置才算指标口）
      2) 否则第一个 containerPort（仅展示/拨测，不代表有 /metrics）
      3) 都没有则 0（仍注册 PodIP）
    不再默认 8080。
    """
    meta = (pod.get("metadata") or {})
    ann = meta.get("annotations") or {}
    for key in cluster_cfg.get("port_annotations") or [
        "yunshu.io/metrics-port",
        "prometheus.io/port",
    ]:
        v = ensure_text(ann.get(key) or "").strip()
        if v.isdigit():
            return int(v), True

    first = None
    for c in ((pod.get("spec") or {}).get("containers") or []):
        for p in c.get("ports") or []:
            cp = p.get("containerPort")
            if not cp:
                continue
            cp = int(cp)
            if first is None:
                first = cp
            name = ensure_text(p.get("name") or "").lower()
            if name in ("metrics", "http-metrics", "prometheus"):
                return cp, False
    if first is not None:
        return first, False
    return 0, False


def pick_metrics_path(pod, cluster_cfg):
    """
    指标路径：仅当 Pod（或集群配置显式）配置了 path 注解/字段时才返回。
    不默认 /metrics，避免 nginx 等无指标端点被 Prom 刮挂。
    """
    ann = ((pod.get("metadata") or {}).get("annotations") or {})
    for key in cluster_cfg.get("path_annotations") or [
        "yunshu.io/metrics-path",
        "prometheus.io/path",
    ]:
        v = ensure_text(ann.get(key) or "").strip()
        if v:
            return v if v.startswith("/") else ("/" + v)
    # 集群级仅当配置里显式写了非空 metrics_path 才用（不要写默认 /metrics）
    v = ensure_text(cluster_cfg.get("metrics_path") or "").strip()
    if v:
        return v if v.startswith("/") else ("/" + v)
    return u""


def should_register(pod, cluster_cfg):
    """是否注册到 Consul：label_selector 已过滤；显式 scrape=false 仍可登记（无 metrics_path）。"""
    ann = ((pod.get("metadata") or {}).get("annotations") or {})
    scrape = ensure_text(ann.get("prometheus.io/scrape") or "").lower()
    # 仅当要求必须带 scrape 注解时才拦截
    require = cluster_cfg.get("require_scrape_annotation")
    if require and scrape not in ("true", "1", "yes"):
        return False
    return True


def wants_metrics_scrape(pod, mpath):
    """有显式 path 且未声明 scrape=false 时，才视为可被 Prom 抓取。"""
    if not mpath:
        return False
    ann = ((pod.get("metadata") or {}).get("annotations") or {})
    scrape = ensure_text(ann.get("prometheus.io/scrape") or "").lower()
    if scrape in ("false", "0", "no"):
        return False
    return True


def pod_app_label(pod):
    labels = ((pod.get("metadata") or {}).get("labels") or {})
    for k in (
        "app.kubernetes.io/name",
        "app",
        "k8s-app",
        "name",
    ):
        v = ensure_text(labels.get(k) or "").strip()
        if v:
            return v
    return u""


def expand_pods(cfg):
    services = []
    defaults = cfg.get("defaults") or {}
    for cluster_cfg in cfg.get("clusters") or []:
        if not cluster_cfg.get("enabled", True):
            continue
        cluster = ensure_text(cluster_cfg.get("name") or "k8s").strip() or "k8s"
        tags = list(cluster_cfg.get("tags") or ["yunshu-metrics", "k8s"])
        base_meta = meta_defaults(cfg, cluster_cfg)
        base_meta["managed_by"] = MANAGED_BY
        base_meta["cluster"] = cluster
        base_meta.setdefault("exporter_role", "k8s_pod")

        namespaces = cluster_cfg.get("namespaces") or []
        label_selector = ensure_text(cluster_cfg.get("label_selector") or "").strip()
        field_selector = ensure_text(
            cluster_cfg.get("field_selector") or "status.phase=Running"
        ).strip()
        only_ready = bool(cluster_cfg.get("only_ready", False))

        ns_list = namespaces if namespaces else [u""]
        for ns in ns_list:
            args = ["get", "pods", "-o", "json"]
            if ns:
                args.extend(["-n", ensure_text(ns)])
            else:
                args.append("--all-namespaces")
            if label_selector:
                args.extend(["-l", label_selector])
            if field_selector:
                args.extend(["--field-selector", field_selector])
            doc = run_kubectl(cfg, args, cluster_cfg)
            for pod in doc.get("items") or []:
                if not should_register(pod, cluster_cfg):
                    continue
                meta_obj = pod.get("metadata") or {}
                status = pod.get("status") or {}
                pod_name = ensure_text(meta_obj.get("name") or "")
                namespace = ensure_text(meta_obj.get("namespace") or ns or "default")
                pod_ip = ensure_text(status.get("podIP") or "").strip()
                if not pod_name or not pod_ip:
                    continue
                if only_ready:
                    ready = False
                    for c in status.get("conditions") or []:
                        if ensure_text(c.get("type")) == "Ready" and ensure_text(
                            c.get("status")
                        ) == "True":
                            ready = True
                            break
                    if not ready:
                        continue

                port, _port_from_ann = pick_port(pod, cluster_cfg)
                mpath = pick_metrics_path(pod, cluster_cfg)
                if not wants_metrics_scrape(pod, mpath):
                    mpath = u""
                app = pod_app_label(pod)
                node = ensure_text(status.get("hostIP") or "")
                if port > 0:
                    sid = "k8s-%s-%s-%s-%s" % (
                        sanitize_id(cluster),
                        sanitize_id(namespace),
                        sanitize_id(pod_name),
                        int(port),
                    )
                else:
                    sid = "k8s-%s-%s-%s" % (
                        sanitize_id(cluster),
                        sanitize_id(namespace),
                        sanitize_id(pod_name),
                    )
                if len(sid) > 120:
                    sid = "k8s-%s-%s-%s" % (
                        sanitize_id(cluster),
                        sanitize_id(pod_ip),
                        int(port) if port > 0 else 0,
                    )

                meta = dict(base_meta)
                meta["namespace"] = namespace
                meta["pod"] = pod_name
                meta["node"] = node
                meta["pod_ip"] = pod_ip
                if mpath:
                    meta["metrics_path"] = mpath
                    meta["scrape"] = "true"
                else:
                    meta["scrape"] = "false"
                if app:
                    meta["app"] = app
                    meta.setdefault("service", app)
                meta.setdefault(
                    "yunshu_project",
                    ensure_text(defaults.get("yunshu_project") or "1"),
                )

                # 无指标路径 → 只进目录服务 k8s-pod（Yunshu/Consul 展示 PodIP）
                # 有 prometheus.io/path → 进 k8s-pod-metrics，才被 Prom 采集
                metrics_service = ensure_text(
                    cluster_cfg.get("metrics_service") or "k8s-pod-metrics"
                ).strip() or "k8s-pod-metrics"
                catalog_service = ensure_text(
                    cluster_cfg.get("service") or "k8s-pod"
                ).strip() or "k8s-pod"
                if mpath:
                    reg_name = metrics_service
                    svc_tags = list(tags)
                    if "has-metrics" not in svc_tags:
                        svc_tags.append("has-metrics")
                else:
                    reg_name = catalog_service
                    svc_tags = list(tags)

                svc = {
                    "ID": sid,
                    "Name": reg_name,
                    "Tags": svc_tags,
                    # 始终写真实 PodIP，供 Consul UI / Yunshu 监控对象展示
                    "Address": pod_ip,
                    "Port": int(port),
                    "Meta": meta,
                }
                if mpath and cluster_cfg.get("check_http"):
                    scheme = ensure_text(cluster_cfg.get("check_scheme") or "http")
                    svc["Check"] = {
                        "HTTP": "%s://%s:%s%s"
                        % (scheme, pod_ip, int(port), mpath),
                        "Interval": ensure_text(cluster_cfg.get("check_interval") or "30s"),
                        "Timeout": ensure_text(cluster_cfg.get("check_timeout") or "5s"),
                    }
                elif port > 0 and cluster_cfg.get("check_tcp"):
                    svc["Check"] = {
                        "TCP": "%s:%s" % (pod_ip, int(port)),
                        "Interval": ensure_text(cluster_cfg.get("check_interval") or "30s"),
                        "Timeout": ensure_text(cluster_cfg.get("check_timeout") or "5s"),
                    }
                services.append(svc)
    return services


def list_managed_agent_services(cfg):
    addr = (cfg.get("consul") or {}).get("addr") or "http://127.0.0.1:8500"
    token = consul_token(cfg)
    code, body = consul_request(addr, token, "GET", "/v1/agent/services", None)
    if code != 200:
        die("list agent services failed HTTP %s %s" % (code, body))
    try:
        data = json.loads(body)
    except Exception as e:
        die("parse agent services: %s" % e)
    out = []
    for sid, svc in (data or {}).items():
        meta = svc.get("Meta") or {}
        if ensure_text(meta.get("managed_by")) == MANAGED_BY:
            out.append(ensure_text(sid))
    return out


def do_register(cfg, services):
    addr = (cfg.get("consul") or {}).get("addr") or "http://127.0.0.1:8500"
    token = consul_token(cfg)
    ok = fail = 0
    for svc in services:
        code, body = consul_request(
            addr, token, "PUT", "/v1/agent/service/register", svc
        )
        if code == 200:
            print(
                "OK  register %s %s:%s"
                % (svc["ID"], svc.get("Address"), svc.get("Port"))
            )
            ok += 1
        else:
            print("FAIL register %s HTTP %s %s" % (svc["ID"], code, body))
            fail += 1
    print("done register: ok=%s fail=%s" % (ok, fail))
    return fail == 0


def do_deregister_ids(cfg, ids):
    addr = (cfg.get("consul") or {}).get("addr") or "http://127.0.0.1:8500"
    token = consul_token(cfg)
    ok = fail = 0
    for sid in ids:
        code, body = consul_request(
            addr,
            token,
            "PUT",
            "/v1/agent/service/deregister/%s" % to_native_str(sid),
            None,
        )
        if code == 200:
            print("OK  deregister %s" % sid)
            ok += 1
        else:
            print("FAIL deregister %s HTTP %s %s" % (sid, code, body))
            fail += 1
    print("done deregister: ok=%s fail=%s" % (ok, fail))
    return fail == 0


def do_sync(cfg, services):
    desired = set(ensure_text(s["ID"]) for s in services)
    existing = set(list_managed_agent_services(cfg))
    stale = sorted(existing - desired)
    if stale:
        print("stale managed services: %s" % len(stale))
        do_deregister_ids(cfg, stale)
    return do_register(cfg, services)


def main():
    ap = argparse.ArgumentParser(description="Sync K8s pods to Consul")
    ap.add_argument("-c", "--config", default="consul-k8s-pods.json")
    ap.add_argument(
        "action",
        choices=["sync", "register", "deregister", "list"],
        help="sync=调和（推荐） | register | deregister 全部托管项 | list",
    )
    args = ap.parse_args()

    cfg = load_config(args.config)
    services = expand_pods(cfg)

    if args.action == "list":
        print(json.dumps(services, indent=2, ensure_ascii=True))
        print("# count=%s" % len(services), file=sys.stderr)
        return

    if args.action == "deregister":
        ids = list_managed_agent_services(cfg)
        if not ids:
            print("no managed services")
            return
        ok = do_deregister_ids(cfg, ids)
        sys.exit(0 if ok else 2)

    if not services:
        print("WARN: no pods matched; will still purge stale managed services")
    if args.action == "register":
        ok = do_register(cfg, services)
    else:
        ok = do_sync(cfg, services)
    sys.exit(0 if ok else 2)


if __name__ == "__main__":
    main()
