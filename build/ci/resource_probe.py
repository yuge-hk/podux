"""Short Linux-only resource/lifecycle probe using a test connector, no Edge.
Uses only a new temporary directory, random loopback port and disposable account.
"""
import argparse
import base64
import json
import os
from pathlib import Path
import secrets
import signal
import socket
import sqlite3
import statistics
import subprocess
import tempfile
import time
import urllib.error
import urllib.request

p = argparse.ArgumentParser()
p.add_argument("--podux", required=True)
p.add_argument("--connector", required=True)
p.add_argument("--output", required=True)
p.add_argument("--cycles", type=int, default=12)
p.add_argument("--idle-seconds", type=int, default=5)
a = p.parse_args()
if not Path("/proc/self/status").exists():
    raise SystemExit("Skipped: Linux /proc is required; not a cross-platform resource claim")
with tempfile.TemporaryDirectory(prefix="podux-resource-") as root:
    with socket.socket() as s:
        s.bind(("127.0.0.1", 0))
        port = s.getsockname()[1]
    env = os.environ.copy()
    env["PODUX_CLOUDFLARED_BINARY"] = str(Path(a.connector).resolve())
    # Preseed an existing PocketBase superuser in this disposable directory so
    # its first-run installer does not launch an unrelated desktop browser.
    setup = subprocess.run([str(Path(a.podux).resolve()), "superuser", "upsert", "probe@example.invalid", secrets.token_urlsafe(24), "--dir", root+"/data"], stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
    if setup.returncode: raise RuntimeError("Disposable superuser setup failed")
    proc = subprocess.Popen([str(Path(a.podux).resolve()), "serve", "--http", f"127.0.0.1:{port}", "--dir", root + "/data"], env=env, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
    auth = None
    token = base64.b64encode(json.dumps({"a": "test", "t": "00000000-0000-0000-0000-000000000000", "s": "synthetic!"}).encode()).decode()
    def request(method, path, body=None):
        headers = {"Content-Type": "application/json"}
        if auth:
            headers["Authorization"] = "Bearer " + auth
        req = urllib.request.Request(f"http://127.0.0.1:{port}/api" + path, method=method, headers=headers, data=json.dumps(body).encode() if body is not None else None)
        with urllib.request.urlopen(req, timeout=10) as response:
            raw = response.read()
        assert token.encode() not in raw and b'"ciphertext"' not in raw
        return json.loads(raw or b"null")
    def until(test):
        deadline = time.monotonic() + 20
        while time.monotonic() < deadline:
            try:
                if test():
                    return
            except (OSError, urllib.error.URLError):
                pass
            time.sleep(0.05)
        raise RuntimeError("Disposable probe condition timed out")
    def child_pids():
        children = []
        for entry in Path("/proc").iterdir():
            if not entry.name.isdigit():
                continue
            try:
                text = (entry / "stat").read_text().rsplit(")", 1)[1].split()
                if int(text[1]) == proc.pid:
                    children.append(int(entry.name))
            except (OSError, ValueError, IndexError):
                pass
        return children
    def ticks():
        fields = Path(f"/proc/{proc.pid}/stat").read_text().rsplit(")", 1)[1].split()
        return int(fields[11]) + int(fields[12])
    def snapshot(label):
        before, start = ticks(), time.monotonic()
        child_before={}
        for pid in child_pids():
            fields=Path(f"/proc/{pid}/stat").read_text().rsplit(")",1)[1].split()
            child_before[pid]=int(fields[11])+int(fields[12])
        time.sleep(a.idle_seconds)
        duration = time.monotonic() - start
        status = dict(line.split(":", 1) for line in Path(f"/proc/{proc.pid}/status").read_text().splitlines() if ":" in line)
        child_stats=[]
        for pid in child_pids():
            child_status=dict(line.split(":",1) for line in Path(f"/proc/{pid}/status").read_text().splitlines() if ":" in line)
            fields=Path(f"/proc/{pid}/stat").read_text().rsplit(")",1)[1].split()
            cpu=(int(fields[11])+int(fields[12])-child_before[pid])/os.sysconf("SC_CLK_TCK")/duration*100 if pid in child_before else None
            child_stats.append({"pid":pid,"name":child_status.get("Name", "").strip(),"state":child_status.get("State", "").strip(),"rssKiB":int(child_status["VmRSS"].split()[0]) if "VmRSS" in child_status else None,"threads":int(child_status["Threads"]),"fdCount":len(list(Path(f"/proc/{pid}/fd").iterdir())),"cpuPercentOneCore":cpu})
        return {"label": label, "rssKiB": int(status["VmRSS"].split()[0]), "threads": int(status["Threads"]), "fdCount": len(list(Path(f"/proc/{proc.pid}/fd").iterdir())), "childPids": child_pids(), "childStats":child_stats, "cpuPercentOneCore": (ticks() - before) / os.sysconf("SC_CLK_TCK") / duration * 100, "sampleSeconds": duration, "sqliteBytes": sum(f.stat().st_size for f in Path(root + "/data").glob("*.db*") if f.is_file())}
    observed = set()
    try:
        until(lambda: request("GET", "/health") is not None)
        password = secrets.token_urlsafe(24)
        request("POST", "/system/initialize", {"email": "resource@example.invalid", "password": password, "language": "en"})
        auth = request("POST", "/collections/fh_users/auth-with-password", {"identity": "resource@example.invalid", "password": password})["token"]
        config = {"name": "resource", "description": "test fixture only", "protocol": "quic", "edgeIPVersion": "auto", "gracePeriodSeconds": 1, "maxRestartAttempts": 2, "autoRestart": True, "token": token}
        profiles = [request("POST", "/cloudflared/tunnels", config) for _ in range(2)]
        records = [snapshot("Podux only")]
        def control(profile, action):
            return request("POST", f"/cloudflared/tunnels/{profile['id']}/{action}")
        def ready(profile):
            return request("GET", f"/cloudflared/tunnels/{profile['id']}/status")["ready"]
        control(profiles[0], "start")
        until(lambda: ready(profiles[0]))
        records.append(snapshot("one test connector"))
        control(profiles[1], "start")
        until(lambda: ready(profiles[1]))
        records.append(snapshot("two test connectors"))
        for sample in records:
            observed.update(sample["childPids"])
        control(profiles[1], "stop")
        control(profiles[0], "stop")
        start_times, stop_times, restart_times = [], [], []
        for _ in range(a.cycles):
            start = time.monotonic()
            control(profiles[0], "start")
            until(lambda: ready(profiles[0]))
            start_times.append(time.monotonic() - start)
            observed.update(child_pids())
            start = time.monotonic()
            control(profiles[0], "restart")
            until(lambda: ready(profiles[0]))
            restart_times.append(time.monotonic() - start)
            observed.update(child_pids())
            start = time.monotonic()
            control(profiles[0], "stop")
            stop_times.append(time.monotonic() - start)
        records.append(snapshot("after repeated start/restart/stop"))
        logs = request("GET", f"/cloudflared/tunnels/{profiles[0]['id']}/logs?after=0")
        assert len(logs["items"]) <= 500
        remaining = child_pids()
        if remaining:
            details=[]
            for pid in remaining:
                try: details.append({"pid":pid,"comm":Path(f"/proc/{pid}/comm").read_text().strip(),"stat":Path(f"/proc/{pid}/stat").read_text()})
                except OSError: pass
            Path(a.output).write_text(json.dumps({"status":"Failed","reason":"children observed after Stop","children":details,"snapshots":records},indent=2))
        assert not remaining, "children observed after Stop; see diagnostic JSON"
        control(profiles[0], "start")
        until(lambda: ready(profiles[0]))
        observed.update(child_pids())
        start = time.monotonic()
        proc.send_signal(signal.SIGTERM)
        assert proc.wait(timeout=10) == 0
        shutdown = time.monotonic() - start
        assert not any(Path(f"/proc/{pid}").exists() for pid in observed)
        assert not list(Path(root + "/data/cloudflared").iterdir())
        with sqlite3.connect(root + "/data/data.db") as db:
            assert db.execute("PRAGMA quick_check").fetchone()[0] == "ok"
        duration = lambda values: {"medianSeconds": statistics.median(values), "maxSeconds": max(values)}
        result = {"status": "Passed", "scope": "Linux, test-only Go connector, no Cloudflare Edge", "cycles": a.cycles, "snapshots": records, "start": duration(start_times), "restart": duration(restart_times), "stop": duration(stop_times), "shutdownSeconds": shutdown, "remainingChildren": 0, "remainingCredentialDirectories": 0, "databaseReopenQuickCheck": "ok", "retainedLogEvents": len(logs["items"]), "longSoak": "Skipped: 1–2 hour stability not performed"}
        Path(a.output).write_text(json.dumps(result, indent=2))
        print(json.dumps(result, indent=2))
    finally:
        if proc.poll() is None:
            proc.terminate()
            try:
                proc.wait(timeout=10)
            except subprocess.TimeoutExpired:
                proc.kill()
                proc.wait()
