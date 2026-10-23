"""Linux-only disposable acceptance; uses the test connector, never Cloudflare Edge."""
import base64
import json
import os
from pathlib import Path
import secrets
import signal
import socket
import sqlite3
import subprocess
import tempfile
import time
import urllib.error
import urllib.request

BINARY = '/tmp/podux-phase4-tools/podux'
CONNECTOR = '/tmp/podux-phase4-tools/connector'
OUTPUT = Path('/root/公共/podux-phase4-validation/recovery.json')
assert Path('/proc/self/stat').exists(), 'Linux-only child inspection'
with tempfile.TemporaryDirectory(prefix='podux-phase4-recovery-') as folder:
    root = Path(folder)
    data = root / 'data'
    key = root / 'keys/master.key'
    with socket.socket() as listener:
        listener.bind(('127.0.0.1', 0))
        port = listener.getsockname()[1]
    endpoint = f'http://127.0.0.1:{port}/api'
    env = os.environ.copy()
    env.update(PODUX_CLOUDFLARED_BINARY=CONNECTOR, PODUX_CLOUDFLARED_KEY_FILE=str(key))
    password = secrets.token_urlsafe(24)
    subprocess.run([BINARY, 'superuser', 'upsert', 'seed@example.invalid', password, '--dir', str(data)], env=env, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL, check=True)
    token = base64.b64encode(json.dumps({'a':'test','t':'00000000-0000-0000-0000-000000000000','s':'synthetic!'}).encode()).decode()
    auth = None
    process = None
    sink = open(root/'backend.txt', 'ab')
    checks = []
    def call(method, path, body=None):
        headers = {'Content-Type':'application/json'}
        if auth: headers['Authorization'] = 'Bearer '+auth
        req = urllib.request.Request(endpoint+path, method=method, headers=headers, data=json.dumps(body).encode() if body is not None else None)
        with urllib.request.urlopen(req, timeout=5) as response: raw = response.read()
        assert token.encode() not in raw and b'"ciphertext"' not in raw and str(key).encode() not in raw
        return json.loads(raw or b'null')
    def until(predicate):
        deadline = time.monotonic()+20
        while time.monotonic()<deadline:
            try:
                if predicate(): return
            except (OSError, urllib.error.URLError): pass
            time.sleep(.05)
        raise RuntimeError('Disposable acceptance condition timed out')
    def start():
        global process
        process = subprocess.Popen([BINARY,'serve','--http',f'127.0.0.1:{port}','--dir',str(data)],env=env,stdout=sink,stderr=sink)
        until(lambda: call('GET','/health') is not None)
    def stop():
        global process
        if process is not None:
            process.send_signal(signal.SIGTERM)
            process.wait(timeout=15)
            assert process.returncode == 0
            process = None
        assert not list((data/'cloudflared').glob('*/token'))
    def login():
        global auth
        auth = call('POST','/collections/fh_users/auth-with-password',{'identity':'probe@example.invalid','password':password})['token']
    def status(profile): return call('GET',f"/cloudflared/tunnels/{profile['id']}/status")
    def control(profile, action): return call('POST',f"/cloudflared/tunnels/{profile['id']}/{action}")
    def released(address):
        port=int(address.rsplit(':',1)[1])
        with socket.socket() as probe:
            probe.settimeout(.5)
            assert probe.connect_ex(('127.0.0.1',port)) != 0, 'Metrics still accepts connections'
        # Match Go's Unix net.Listen reuse behavior; TIME_WAIT is not a listener leak.
        with socket.socket() as listener:
            listener.setsockopt(socket.SOL_SOCKET,socket.SO_REUSEADDR,1)
            listener.bind(('127.0.0.1',port))
    def children():
        found=[]
        for entry in Path('/proc').iterdir():
            if not entry.name.isdigit(): continue
            try:
                fields=(entry/'stat').read_text().rsplit(')',1)[1].split()
                if int(fields[1])==process.pid: found.append(int(entry.name))
            except (OSError,ValueError,IndexError): pass
        return found
    try:
        start()
        call('POST','/system/initialize',{'email':'probe@example.invalid','password':password,'language':'en'})
        login()
        cfg={'name':'Auto profile','autoStart':True,'autoRestart':True,'maxRestartAttempts':2,'protocol':'quic','edgeIPVersion':'auto','gracePeriodSeconds':1,'connectionRetries':0,'metricsPort':0}
        a=call('POST','/cloudflared/tunnels',{**cfg,'token':token})
        b=call('POST','/cloudflared/tunnels',{**cfg,'name':'Manual profile','autoStart':False,'token':token})
        control(a,'start');control(b,'start')
        until(lambda: status(a)['ready'] and status(b)['ready'])
        addresses=[status(a)['metricsAddress'],status(b)['metricsAddress']]
        assert len(set(addresses))==2 and len(children())==2
        checks.append('Two independent real fixture processes with distinct loopback Metrics')
        original_key=key.read_bytes()
        stop()
        for address in addresses: released(address)
        with sqlite3.connect(data/'data.db') as database:
            ciphertext=[row[0] for row in database.execute('SELECT ciphertext FROM fh_cloudflare_tunnels ORDER BY id')]
        assert all(value.startswith('v1:') and token not in value for value in ciphertext)
        checks.append('Persisted Tokens are encrypted and API responses omit secrets')
        start();login()
        until(lambda: status(a)['ready'])
        assert status(b)['phase']=='stopped' and len(children())==1
        assert key.read_bytes()==original_key
        checks.append('Actual Podux restart restores only autoStart profile with the same external key')
        generation=status(a)['generation']
        os.kill(children()[0],signal.SIGKILL)
        until(lambda: status(a)['restartCount']==1 and status(a)['ready'])
        assert status(a)['generation']==generation and len(children())==1
        checks.append('Unexpected fixture child exit recovers through bounded automatic retry')
        control(a,'stop')
        assert status(a)['phase']=='stopped' and not children()
        stop();start();login()
        until(lambda: status(a)['ready'])
        assert status(b)['phase']=='stopped'
        checks.append('Manual Stop preserves autoStart policy for the next Podux restart')
        address=status(a)['metricsAddress']
        stop();released(address)
        sink.flush()
        logs=(root/'backend.txt').read_text(errors='replace')
        assert token not in logs and all(value not in logs for value in ciphertext)
        checks.append('Development SQL logs exclude both plaintext and encrypted Token values')
        checks.append('Normal exits reap children, remove Token files and release Metrics ports')
        OUTPUT.write_text(json.dumps({'status':'Passed','scope':'Linux, synthetic credentials and test connector; no Edge connection','passed':len(checks),'checks':checks,'realEdge':'Blocked: authorized nonproduction credentials not supplied'},indent=2)+'\n')
        print(OUTPUT.read_text())
    finally:
        if process is not None:
            process.terminate()
            try: process.wait(timeout=15)
            except subprocess.TimeoutExpired: process.kill();process.wait()
        sink.close()
