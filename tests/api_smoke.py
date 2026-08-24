#!/usr/bin/env python3
"""API smoke for GoRocksDB. Cost: ¥0. Run inside compose network or against localhost."""
import json
import os
import time
import urllib.error
import urllib.request

BASE = os.environ.get("GOROCKSDB_SMOKE_BASE", "http://127.0.0.1:28741")
TS = str(int(time.time() * 1000))


def call(method, path, data=None, expect=200):
    req = urllib.request.Request(
        BASE + path,
        data=None if data is None else json.dumps(data).encode(),
        method=method,
        headers={"Content-Type": "application/json"},
    )
    try:
        with urllib.request.urlopen(req, timeout=15) as resp:
            body = json.loads(resp.read().decode())
            assert resp.status == expect, (resp.status, body)
            return body
    except urllib.error.HTTPError as e:
        raw = e.read().decode()
        raise AssertionError(f"{method} {path} -> {e.code} {raw}") from e


def main():
    h = call("GET", "/api/health")
    assert h["ok"] is True
    assert h["data"]["status"] == "ok"

    key = f"smoke-{TS}"
    put = call("PUT", f"/api/kv/{key}", {"value": "hello-lsm"})
    assert put["data"]["value"] == "hello-lsm"

    got = call("GET", f"/api/kv/{key}")
    assert got["data"]["value"] == "hello-lsm"

    st = call("GET", "/api/lsm/state")
    assert "levels" in st["data"]
    assert st["data"]["profile"] in ("demo", "production", "test")

    met = call("GET", "/api/metrics")
    assert "puts" in met["data"]

    scan = call("GET", f"/api/scan?start={key}&end={key}z&limit=10")
    assert scan["data"]["count"] >= 1

    call("POST", "/api/admin/flush", {})
    call("DELETE", f"/api/kv/{key}")
    print("SMOKE PASS")


if __name__ == "__main__":
    main()
