#!/usr/bin/env python
# -*- coding: utf-8 -*-
"""
统一向 Consul 注册 / 反注册监控目标（Telegraf、ICMP、HTTP、TCP、Pushgateway）。

兼容：Python 2.7 / 3.x，仅标准库。

用法：
  export CONSUL_TOKEN=<metrics-register SecretID>
  python consul_targets_sync.py -c consul-targets.json register
  python consul_targets_sync.py -c consul-targets.json deregister
  python consul_targets_sync.py -c consul-targets.json sync
  python consul_targets_sync.py -c consul-targets.json list
"""

from __future__ import print_function

import argparse
import json
import os
import re
import socket
import sys

try:
    from urllib.request import Request, urlopen
    from urllib.error import HTTPError, URLError
except ImportError:
    from urllib2 import Request, urlopen, HTTPError, URLError  # type: ignore


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
    """Consul Meta / JSON 文本：Py2 返回 unicode，避免 str(中文) 触发 ascii encode。"""
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
    """Py2 urllib2/httplib 要求 URL/Header/Body 为 byte str，不能 unicode 与 utf-8 混拼。"""
    t = ensure_text(s)
    if sys.version_info[0] < 3:
        return t.encode("utf-8")
    return t


def die(msg, code=1):
    print("ERROR: %s" % ensure_text(msg), file=sys.stderr)
    sys.exit(code)


def load_config(path):
    # Py2 默认 ascii；配置里常见中文 meta，必须按 utf-8 读
    try:
        import io

        with io.open(path, "r", encoding="utf-8") as f:
            return json.load(f)
    except TypeError:
        import codecs

        with codecs.open(path, "r", "utf-8") as f:
            return json.load(f)


def detect_ip(prefer_iface=None):
    if prefer_iface:
        try:
            import subprocess

            out = subprocess.check_output(
                ["ip", "-4", "addr", "show", prefer_iface],
                stderr=subprocess.STDOUT,
            )
            out = to_text(out)
            m = re.search(r"inet\s+(\d+\.\d+\.\d+\.\d+)", out)
            if m:
                return m.group(1)
        except Exception:
            pass
    try:
        s = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
        s.connect(("8.8.8.8", 80))
        ip = s.getsockname()[0]
        s.close()
        if ip and not ip.startswith("127."):
            return ip
    except Exception:
        pass
    try:
        return socket.gethostbyname(socket.gethostname())
    except Exception:
        return "127.0.0.1"


def sanitize_id(s):
    return re.sub(r"[^A-Za-z0-9._-]", "-", ensure_text(s))


def parse_host_port(ep):
    """Parse '10.10.10.4:3306' or '[::1]:9092' → (host, port)."""
    ep = ensure_text(ep).strip()
    if not ep:
        return u"", 0
    if ep.startswith("["):
        m = re.match(r"^\[([^\]]+)\]:(\d+)$", ep)
        if m:
            return ensure_text(m.group(1)), int(m.group(2))
        return ep.strip("[]"), 0
    if ":" in ep:
        host, port_s = ep.rsplit(":", 1)
        try:
            return ensure_text(host), int(port_s)
        except ValueError:
            return ep, 0
    return ep, 0


def tcp_instance_id(svc_name, host, port):
    """Consul Service.ID：tcp-<服务名>-<ip>-<端口>"""
    name = sanitize_id(svc_name) or "target"
    hip = sanitize_id(host) or "unknown"
    return "tcp-%s-%s-%s" % (name, hip, int(port or 0))


def config_dir_of(cfg):
    return ensure_text(cfg.get("_config_dir") or ".")


def load_lines_file(path, base_dir):
    """读批量清单：# 注释、空行忽略；相对路径相对配置文件目录。"""
    path = ensure_text(path).strip()
    if not path:
        return []
    if not os.path.isabs(path):
        path = os.path.join(ensure_text(base_dir), path)
    if not os.path.isfile(path):
        die("list file not found: %s" % path)
    try:
        import io

        f = io.open(path, "r", encoding="utf-8")
    except TypeError:
        import codecs

        f = codecs.open(path, "r", "utf-8")
    lines = []
    try:
        for raw in f:
            line = ensure_text(raw).strip()
            if not line or line.startswith("#"):
                continue
            lines.append(line)
    finally:
        f.close()
    return lines


def parse_tcp_spec(line, default_meta=None):
    """
    紧凑 TCP 行 / 字符串，返回 dict: host(host:port), id, meta
      mysql@10.10.10.4:3306
      mysql 10.10.10.4 3306
      mysql 10.10.10.4 3306 yunshu db
      10.10.10.4:3306
    """
    line = ensure_text(line).strip()
    meta = dict(default_meta or {})
    pid = u""
    ep = u""

    if "@" in line and "://" not in line.split("@", 1)[0]:
        # service@host:port
        left, right = line.split("@", 1)
        pid = left.strip()
        ep = right.strip()
    else:
        parts = line.replace(",", " ").split()
        if len(parts) == 1 and ":" in parts[0]:
            ep = parts[0]
        elif len(parts) >= 3 and parts[2].isdigit():
            pid = parts[0]
            ep = "%s:%s" % (parts[1], parts[2])
            if len(parts) >= 4:
                meta["app"] = parts[3]
            if len(parts) >= 5:
                meta["component"] = parts[4]
        elif len(parts) >= 2 and ":" in parts[0]:
            ep = parts[0]
            pid = parts[1]
        else:
            die("bad tcp spec: %s (want service@ip:port or service ip port)" % line)

    host, port = parse_host_port(ep)
    if not host or port <= 0:
        die("bad tcp host:port in: %s" % line)
    if pid:
        meta.setdefault("service", pid)
    return {
        "id": pid or sanitize_id("%s-%s" % (host, port)),
        "host": "%s:%s" % (host, port),
        "meta": meta,
    }


def parse_http_spec(line, default_module, default_meta=None):
    """
    紧凑 HTTP 行：
      id|url|module|service|app
      id url [module]
      https://example.com/health
    """
    line = ensure_text(line).strip()
    meta = dict(default_meta or {})
    module = ensure_text(default_module or "http_2xx").strip() or u"http_2xx"
    pid = u""
    url = u""

    if "|" in line:
        parts = [p.strip() for p in line.split("|")]
        if len(parts) < 2:
            die("bad http spec: %s" % line)
        pid = parts[0]
        url = parts[1]
        if len(parts) >= 3 and parts[2]:
            module = parts[2]
        if len(parts) >= 4 and parts[3]:
            meta["service"] = parts[3]
        if len(parts) >= 5 and parts[4]:
            meta["app"] = parts[4]
    else:
        parts = line.split()
        if not parts:
            return None
        if parts[0].startswith("http://") or parts[0].startswith("https://"):
            url = parts[0]
            if len(parts) >= 2:
                module = parts[1]
        else:
            pid = parts[0]
            if len(parts) < 2:
                die("bad http spec (need id url): %s" % line)
            url = parts[1]
            if len(parts) >= 3:
                module = parts[2]
            if len(parts) >= 4:
                meta["service"] = parts[3]
            if len(parts) >= 5:
                meta["app"] = parts[4]

    if not url:
        die("bad http url in: %s" % line)
    if pid:
        meta.setdefault("service", pid)
    return {"id": pid, "url": url, "module": module, "meta": meta}


def collect_tcp_endpoints(t, cfg):
    """合并 endpoints / endpoints_file / by_host / 紧凑字符串 → 统一 dict 列表。"""
    out = []
    base_dir = config_dir_of(cfg)
    type_meta = t.get("meta") or {}

    for item in t.get("endpoints") or []:
        if isinstance(item, dict):
            out.append(item)
        else:
            out.append(parse_tcp_spec(item, type_meta))

    for line in load_lines_file(t.get("endpoints_file") or "", base_dir):
        out.append(parse_tcp_spec(line, type_meta))

    by_host = t.get("by_host") or {}
    if isinstance(by_host, dict):
        for hip, ports in by_host.items():
            hip = ensure_text(hip).strip()
            if not hip:
                continue
            if not isinstance(ports, list):
                die("by_host[%s] must be a list" % hip)
            for p in ports:
                if not isinstance(p, dict):
                    # "3306" or 3306 or "mysql:3306"
                    s = ensure_text(p).strip()
                    if ":" in s and not s[0].isdigit():
                        # mysql:3306
                        name, port_s = s.rsplit(":", 1)
                        out.append(
                            parse_tcp_spec(
                                "%s@%s:%s" % (name, hip, port_s), type_meta
                            )
                        )
                    else:
                        out.append(
                            {
                                "host": "%s:%s" % (hip, s),
                                "meta": dict(type_meta),
                            }
                        )
                    continue
                port = p.get("port")
                if port is None:
                    die("by_host entry missing port: %s" % p)
                name = ensure_text(p.get("service") or p.get("id") or "").strip()
                extra = dict(type_meta)
                extra.update(p.get("meta") or p.get("labels") or {})
                if name:
                    extra.setdefault("service", name)
                for k in ("app", "component", "team"):
                    if p.get(k) is not None:
                        extra[k] = p.get(k)
                out.append(
                    {
                        "id": name or sanitize_id("%s-%s" % (hip, port)),
                        "host": "%s:%s" % (hip, port),
                        "meta": extra,
                    }
                )
    return out


def collect_http_probes(t, cfg):
    """合并 probes / urls / probes_file / 紧凑字符串。"""
    out = []
    base_dir = config_dir_of(cfg)
    default_module = ensure_text(t.get("module") or "http_2xx").strip() or u"http_2xx"
    type_meta = t.get("meta") or {}

    probes = list(t.get("probes") or [])
    if not probes and t.get("urls"):
        for url in t.get("urls") or []:
            probes.append({"url": url, "module": default_module})

    for p in probes:
        if isinstance(p, dict):
            out.append(p)
        else:
            out.append(parse_http_spec(p, default_module, type_meta))

    for line in load_lines_file(t.get("probes_file") or "", base_dir):
        out.append(parse_http_spec(line, default_module, type_meta))
    return out


def consul_token(cfg, target):
    if target.get("token"):
        return target["token"]
    consul = cfg.get("consul") or {}
    if consul.get("token"):
        return consul["token"]
    env_name = consul.get("token_env") or "CONSUL_TOKEN"
    tok = (os.environ.get(env_name) or "").strip()
    if not tok:
        die("missing token: set %s or consul.token / target.token" % env_name)
    return tok


def consul_request(addr, token, method, path, body=None):
    url = to_native_str(ensure_text(addr).rstrip("/") + ensure_text(path))
    data = None
    # Token 来自 JSON 时在 Py2 是 unicode；若 Header 为 unicode 而 Body 为 utf-8 bytes，
    # httplib 会在 msg += message_body 处 UnicodeDecodeError。
    headers = {
        to_native_str("X-Consul-Token"): to_native_str(token),
    }
    if body is not None:
        # ensure_ascii=True → 中文变成 \uXXXX，body 纯 ASCII，Py2.7 最稳
        payload = json.dumps(body, ensure_ascii=True)
        data = to_native_str(payload)
        headers[to_native_str("Content-Type")] = to_native_str("application/json")
    req = Request(url, data=data, headers=headers)
    req.get_method = lambda: method
    try:
        resp = urlopen(req, timeout=15)
        raw = resp.read()
        code = resp.getcode()
        return code, to_text(raw)
    except HTTPError as e:
        raw = e.read()
        return e.code, to_text(raw) if raw else ensure_text(e)
    except URLError as e:
        return 0, ensure_text(e.reason if hasattr(e, "reason") else e)


def meta_base(cfg, target):
    d = dict(cfg.get("defaults") or {})
    d.update(target.get("meta") or {})
    out = {}
    for k, v in d.items():
        out[ensure_text(k)] = ensure_text(v)
    return out


def merge_meta(base, extra):
    """Merge per-target meta; Consul Meta values must be strings (unicode on Py2)."""
    out = dict(base or {})
    if not extra:
        return out
    if not isinstance(extra, dict):
        return out
    for k, v in extra.items():
        if v is None:
            continue
        out[ensure_text(k)] = ensure_text(v)
    return out


def expand_services(cfg):
    """Return (services, stale_ids). stale_ids 为旧短 ID（如 tcp-mysql），sync 时清理。"""
    services = []
    stale_ids = []
    iface = (cfg.get("consul") or {}).get("prefer_iface")
    local_ip = detect_ip(iface)

    for t in cfg.get("targets") or []:
        if not t.get("enabled", True):
            continue
        typ = (t.get("type") or "").strip().lower()
        service = t.get("service") or typ
        tags = list(t.get("tags") or [])
        base = meta_base(cfg, t)

        if typ == "telegraf":
            port = int(t.get("port") or 9273)
            addr = t.get("address") or local_ip
            sid = t.get("id") or ("telegraf-%s-%s" % (addr, port))
            meta = dict(base)
            meta.setdefault("exporter_role", "telegraf")
            svc = {
                "ID": sid,
                "Name": service,
                "Tags": tags or ["yunshu-metrics", "exporter"],
                "Address": addr,
                "Port": port,
                "Meta": meta,
            }
            if t.get("check_http", True):
                svc["Check"] = {
                    "HTTP": "http://%s:%s/metrics" % (addr, port),
                    "Interval": "30s",
                    "Timeout": "5s",
                }
            services.append(svc)

        elif typ == "icmp":
            # hosts / hosts_file；Address = 真实探测 IP
            host_items = []
            for item in t.get("hosts") or []:
                if isinstance(item, dict):
                    host_items.append(item)
                else:
                    host_items.append({"host": item})
            for line in load_lines_file(t.get("hosts_file") or "", config_dir_of(cfg)):
                host_items.append({"host": line})
            for item in host_items:
                extra = {}
                pid = u""
                if isinstance(item, dict):
                    host = ensure_text(item.get("host") or item.get("address") or "").strip()
                    pid = ensure_text(item.get("id") or "").strip()
                    extra = item.get("meta") or item.get("labels") or {}
                else:
                    host = ensure_text(item).strip()
                if not host:
                    continue
                hip, _ = parse_host_port(host)
                if not hip:
                    hip = host
                sid = pid or ("icmp-%s" % sanitize_id(hip))
                if pid and not sid.startswith("icmp-"):
                    sid = "icmp-%s" % sanitize_id(pid)
                meta = merge_meta(base, extra)
                meta["probe_host"] = hip
                meta.setdefault("exporter_role", "blackbox_target")
                svc_name = ensure_text(meta.get("service") or "").strip() or pid
                if svc_name:
                    meta["service"] = ensure_text(svc_name)
                services.append(
                    {
                        "ID": sid,
                        "Name": service,
                        "Tags": tags or ["probe-icmp", "yunshu-metrics"],
                        "Address": hip,
                        "Port": 0,
                        "Meta": meta,
                    }
                )

        elif typ == "http":
            # probes / urls / probes_file / 紧凑行，见 collect_http_probes
            default_module = ensure_text(t.get("module") or "http_2xx").strip() or u"http_2xx"
            for p in collect_http_probes(t, cfg):
                if not isinstance(p, dict):
                    continue
                url = ensure_text(p.get("url") or "").strip()
                module = ensure_text(p.get("module") or default_module).strip() or u"http_2xx"
                pid = ensure_text(p.get("id") or "").strip()
                extra = p.get("meta") or p.get("labels") or {}
                if not url:
                    continue
                sid = pid or ("http-%s" % sanitize_id(url.replace("://", "-")))
                if not sid.startswith("http-") and pid:
                    sid = "http-%s" % sanitize_id(pid)
                meta = merge_meta(base, extra)
                meta["probe_url"] = url
                meta["probe_module"] = module
                meta.setdefault("exporter_role", "blackbox_target")
                # Yunshu 地址列：尽量填 URL 主机
                haddr = u"127.0.0.1"
                hport = 0
                try:
                    from urlparse import urlparse  # Py2
                except ImportError:
                    from urllib.parse import urlparse  # Py3
                try:
                    u = urlparse(url)
                    if u.hostname:
                        haddr = ensure_text(u.hostname)
                    if u.port:
                        hport = int(u.port)
                    elif u.scheme == "https":
                        hport = 443
                    elif u.scheme == "http":
                        hport = 80
                except Exception:
                    pass
                services.append(
                    {
                        "ID": sid,
                        "Name": service,
                        "Tags": tags or ["probe-http", "yunshu-metrics"],
                        "Address": haddr,
                        "Port": hport,
                        "Meta": meta,
                    }
                )

        elif typ == "tcp":
            # endpoints / endpoints_file / by_host / 紧凑串
            for item in collect_tcp_endpoints(t, cfg):
                if not isinstance(item, dict):
                    continue
                ep = ensure_text(
                    item.get("host")
                    or item.get("endpoint")
                    or item.get("address")
                    or ""
                ).strip()
                pid = ensure_text(item.get("id") or "").strip()
                extra = item.get("meta") or item.get("labels") or {}
                if not ep:
                    continue
                # host 字段可能是纯 IP，port 另给
                if ":" not in ep and item.get("port") is not None:
                    ep = "%s:%s" % (ep, item.get("port"))
                host, port = parse_host_port(ep)
                if not host:
                    continue
                if port <= 0:
                    die("tcp endpoint needs host:port, got: %s" % ep)
                meta = merge_meta(base, extra)
                meta["probe_host"] = "%s:%s" % (host, port)
                meta.setdefault("exporter_role", "blackbox_target")
                svc_name = (
                    ensure_text(meta.get("service") or "").strip()
                    or pid
                    or host
                )
                meta["service"] = ensure_text(svc_name)
                sid = tcp_instance_id(svc_name, host, port)
                if pid:
                    legacy = "tcp-%s" % sanitize_id(pid)
                    if legacy != sid:
                        stale_ids.append(legacy)
                services.append(
                    {
                        "ID": sid,
                        "Name": service,
                        "Tags": tags or ["probe-tcp", "yunshu-metrics"],
                        "Address": host,
                        "Port": port,
                        "Meta": meta,
                    }
                )

        elif typ == "pushgateway":
            port = int(t.get("port") or 9091)
            addr = t.get("address") or "127.0.0.1"
            sid = t.get("id") or ("pushgateway-%s-%s" % (addr, port))
            meta = dict(base)
            meta.setdefault("exporter_role", "pushgateway")
            svc = {
                "ID": sid,
                "Name": service,
                "Tags": tags or ["yunshu-metrics"],
                "Address": addr,
                "Port": port,
                "Meta": meta,
            }
            if t.get("check_http", True):
                svc["Check"] = {
                    "HTTP": "http://%s:%s/metrics" % (addr, port),
                    "Interval": "30s",
                    "Timeout": "5s",
                }
            services.append(svc)

        else:
            die("unknown target type: %s" % typ)

    seen = set()
    uniq_stale = []
    for i in stale_ids:
        if i not in seen:
            seen.add(i)
            uniq_stale.append(i)
    return services, uniq_stale


def resolve_token(cfg, svc_name):
    for t in cfg.get("targets") or []:
        name = t.get("service") or t.get("type")
        if name == svc_name and t.get("token"):
            return t["token"]
    return consul_token(cfg, {})


def do_register(cfg, services):
    addr = (cfg.get("consul") or {}).get("addr") or "http://127.0.0.1:8500"
    ok = fail = 0
    for svc in services:
        token = resolve_token(cfg, svc["Name"])
        code, body = consul_request(
            addr, token, "PUT", "/v1/agent/service/register", svc
        )
        if code == 200:
            print("OK  register %s name=%s addr=%s:%s" % (
                svc["ID"], svc["Name"], svc.get("Address"), svc.get("Port"),
            ))
            ok += 1
        else:
            print("FAIL register %s HTTP %s %s" % (svc["ID"], code, body))
            fail += 1
    print("done register: ok=%s fail=%s" % (ok, fail))
    return fail == 0


def do_deregister(cfg, services):
    addr = (cfg.get("consul") or {}).get("addr") or "http://127.0.0.1:8500"
    ok = fail = 0
    for svc in services:
        token = resolve_token(cfg, svc["Name"])
        code, body = consul_request(
            addr,
            token,
            "PUT",
            "/v1/agent/service/deregister/%s" % to_native_str(svc["ID"]),
            None,
        )
        if code == 200:
            print("OK  deregister %s" % svc["ID"])
            ok += 1
        else:
            print("FAIL deregister %s HTTP %s %s" % (svc["ID"], code, body))
            fail += 1
    print("done deregister: ok=%s fail=%s" % (ok, fail))
    return fail == 0


def do_deregister_ids(cfg, ids):
    """按 ID 列表反注册（清理旧短名等）。按 ID 前缀选 service token。"""
    if not ids:
        return True
    addr = (cfg.get("consul") or {}).get("addr") or "http://127.0.0.1:8500"
    ok = fail = 0
    for sid in ids:
        sid = ensure_text(sid)
        if sid.startswith("icmp-"):
            sname = "icmp"
        elif sid.startswith("http-"):
            sname = "http"
        elif sid.startswith("tcp-"):
            sname = "tcp"
        else:
            sname = "tcp"
        token = resolve_token(cfg, sname)
        code, body = consul_request(
            addr,
            token,
            "PUT",
            "/v1/agent/service/deregister/%s" % to_native_str(sid),
            None,
        )
        if code == 200:
            print("OK  deregister stale %s" % sid)
            ok += 1
        else:
            print("FAIL deregister stale %s HTTP %s %s" % (sid, code, body))
            fail += 1
    print("done stale deregister: ok=%s fail=%s" % (ok, fail))
    return fail == 0


def main():
    ap = argparse.ArgumentParser(description="Consul monitoring targets sync")
    ap.add_argument(
        "-c",
        "--config",
        default="consul-targets.json",
        help="JSON config path",
    )
    ap.add_argument(
        "action",
        choices=["register", "deregister", "sync", "list"],
        help="register | deregister | sync | list",
    )
    ap.add_argument(
        "--type",
        dest="only_type",
        default="",
        help="only one type: telegraf,icmp,http,tcp,pushgateway",
    )
    args = ap.parse_args()

    cfg = load_config(args.config)
    cfg["_config_dir"] = os.path.dirname(os.path.abspath(args.config)) or "."
    if args.only_type:
        want = args.only_type.strip().lower()
        cfg["targets"] = [
            t
            for t in (cfg.get("targets") or [])
            if (t.get("type") or "").lower() == want
        ]

    services, stale_ids = expand_services(cfg)
    if args.action == "list":
        # Py2.7 控制台编码可能非 UTF-8，用 ensure_ascii
        print(json.dumps(services, indent=2, ensure_ascii=True))
        return

    if not services:
        die("no services expanded (check enabled targets)")

    if args.action == "register":
        do_deregister_ids(cfg, stale_ids)
        ok = do_register(cfg, services)
    elif args.action == "deregister":
        do_deregister_ids(cfg, stale_ids)
        ok = do_deregister(cfg, services)
    else:
        do_deregister_ids(cfg, stale_ids)
        do_deregister(cfg, services)
        ok = do_register(cfg, services)

    sys.exit(0 if ok else 2)


if __name__ == "__main__":
    main()
