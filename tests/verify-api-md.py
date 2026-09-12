#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""Replay docs/api.md against a running CasaDrop and compare the documented
response shapes with what the server really sends.

A reference others follow has to be executed, not proof-read: every field name
in api.md was once plausible and wrong (`maxDownloads` was silently ignored,
`files[]` was rejected outright, the metrics were `zima_*` not `casadrop_*`).

Usage:
    # start an instance with a throwaway data dir first, then:
    CASADROP_URL=http://127.0.0.1:8811 \
    CASADROP_PASSWORD=testpass1234 \
    CASADROP_SHARE_ROOT=/tmp/casadrop-apitest/media \
        python3 tests/verify-api-md.py

CASADROP_SHARE_ROOT must be inside the instance's SHARE_ALLOWED_PATHS.
Exits 0 when every documented key is present, 1 otherwise.

NOTE: the login endpoint is rate-limited to 5 attempts/minute per IP; this
script logs in twice, so back-to-back runs may need a minute in between.
"""
import json, os, subprocess, sys, tempfile, urllib.error, urllib.request

BASE     = os.environ.get("CASADROP_URL", "http://127.0.0.1:8811").rstrip("/")
PASSWORD = os.environ.get("CASADROP_PASSWORD", "testpass1234")
ROOT     = os.environ.get("CASADROP_SHARE_ROOT", "")
if not ROOT:
    sys.exit("CASADROP_SHARE_ROOT is required (a dir inside SHARE_ALLOWED_PATHS)")
os.makedirs(ROOT, exist_ok=True)

# Fixtures this script owns, so it does not depend on files left by a previous run.
SP = tempfile.mkdtemp(prefix="casadrop-apimd-")
with open(os.path.join(SP, "document.pdf"), "w") as f:
    f.write("fixture")
with open(os.path.join(ROOT, "f.txt"), "w") as f:
    f.write("fixture")

fails, checks = [], 0

def http(method, path, body=None, ctype=None, cookie=None, raw=False):
    req = urllib.request.Request(BASE + path, method=method)
    if cookie: req.add_header("Cookie", cookie)
    if ctype:  req.add_header("Content-Type", ctype)
    data = body.encode() if isinstance(body, str) else body
    try:
        with urllib.request.urlopen(req, data) as r:
            out = r.read()
            return r.status, (out if raw else json.loads(out or b"null"))
    except urllib.error.HTTPError as e:
        return e.code, e.read().decode(errors="replace")

def check(name, documented, actual):
    """documented/actual are key SETS."""
    global checks
    checks += 1
    missing = documented - actual      # documented but not sent  -> doc is wrong
    if missing:
        fails.append("%s: documented keys the server does NOT send: %s" % (name, sorted(missing)))
    else:
        print("  ok  %-34s (%d documented keys all present)" % (name, len(documented)))

# --- login ---
st, _ = http("POST", "/login", json.dumps({"password":PASSWORD}), "application/json")
cookie = None
# simplest: re-do with curl to grab the cookie
cookie = subprocess.run(
    ["curl","-s","-c","-","-X","POST",BASE+"/login","-H","Content-Type: application/json",
     "-d",json.dumps({"password":PASSWORD})],capture_output=True,text=True).stdout
tok = [l.split()[-1] for l in cookie.splitlines() if "casadrop_session" in l][0]
CK = "casadrop_session=" + tok

# --- auth status ---
st, d = http("GET", "/api/auth/status", cookie=CK)
check("GET /api/auth/status", {"authenticated","setupRequired","envPassword","sessionExpiry"}, set(d))

# --- upload (documented field names must WORK) ---
out = subprocess.run(["curl","-s","-b",CK,"-X","POST",BASE+"/api/upload",
    "-F","file=@"+os.path.join(SP,"document.pdf"),"-F","expires_in=168","-F","max_downloads=10"],
    capture_output=True,text=True).stdout
up = json.loads(out)
check("POST /api/upload response", {"id","file_name","file_size","mime_type","has_password",
      "expires_at","created_at","downloads","max_downloads","share_url"}, set(up))
checks += 1
if up["max_downloads"] != 10:
    fails.append("POST /api/upload: documented field max_downloads had no effect (got %r)" % up["max_downloads"])
else:
    print("  ok  documented max_downloads actually applied")

# expires_in=0 => never
out = subprocess.run(["curl","-s","-b",CK,"-X","POST",BASE+"/api/upload",
    "-F","file=@"+os.path.join(SP,"document.pdf"),"-F","expires_in=0"],capture_output=True,text=True).stdout
checks += 1
if not json.loads(out)["expires_at"].startswith("9999"):
    fails.append("expires_in=0 documented as 'never', got %s" % json.loads(out)["expires_at"])
else:
    print("  ok  expires_in=0 => never, as documented")

# --- multi upload: 'files' must work, 'files[]' must NOT ---
out = subprocess.run(["curl","-s","-b",CK,"-X","POST",BASE+"/api/upload/multi",
    "-F","files=@"+os.path.join(SP,"document.pdf"),"-F","expires_in=24"],capture_output=True,text=True).stdout
m = json.loads(out)
check("POST /api/upload/multi", {"shares","success","failed"}, set(m))
code = subprocess.run(["curl","-s","-o","/dev/null","-w","%{http_code}","-b",CK,"-X","POST",
    BASE+"/api/upload/multi","-F","files[]=@"+os.path.join(SP,"document.pdf")],capture_output=True,text=True).stdout
checks += 1
if code != "400":
    fails.append("doc says files[] is rejected with 400; got %s" % code)
else:
    print("  ok  files[] rejected with 400, as documented")

# --- shares list is a bare array ---
st, d = http("GET", "/api/shares", cookie=CK)
checks += 1
if not isinstance(d, list):
    fails.append("GET /api/shares documented as a bare array, got %s" % type(d).__name__)
else:
    print("  ok  GET /api/shares is a bare array")
check("GET /api/shares entry", {"id","file_name","file_size","mime_type","has_password",
      "expires_at","created_at","downloads","max_downloads","share_url"}, set(d[0]))

# --- share-from-path / share-folder ---
st, d = http("POST","/api/share-from-path",
    json.dumps({"path":os.path.join(ROOT,"f.txt"),"expires_in":168,"max_downloads":0,"use_symlink":True}),
    "application/json", CK)
checks += 1
if st != 200: fails.append("share-from-path with documented body: HTTP %s %s" % (st,d))
else: print("  ok  share-from-path accepts the documented body")
st, d = http("POST","/api/share-folder",
    json.dumps({"path":ROOT,"expires_in":168,"max_downloads":0}),"application/json",CK)
checks += 1
if st != 200 or not d.get("is_directory"):
    fails.append("share-folder documented body/is_directory: HTTP %s %s" % (st,d))
else: print("  ok  share-folder returns is_directory:true")

# --- receive links ---
st, rl = http("POST","/api/receive-links", json.dumps({"name":"Project Files","expires_in":168,
    "max_uploads":10,"max_file_size":104857600,"allowed_extensions":".pdf,.docx","auto_share":True}),
    "application/json", CK)
check("POST /api/receive-links", {"id","name","has_password","max_uploads","auto_share",
      "created_at","expires_at","receive_url"}, set(rl))
st, lst = http("GET","/api/receive-links", cookie=CK)
check("GET /api/receive-links entry", {"id","name","has_password","max_uploads","current_uploads",
      "files_count","total_size","auto_share","created_at","expires_at","receive_url"}, set(lst[0]))
out = subprocess.run(["curl","-s","-X","POST",BASE+"/r/"+rl["id"]+"/upload",
    "-F","file=@"+os.path.join(SP,"document.pdf")],capture_output=True,text=True).stdout
check("POST /r/{id}/upload", {"success","file_id","file_name","file_size","share_id","share_url"},
      set(json.loads(out)))
st, fl = http("GET","/api/receive-links/"+rl["id"]+"/files", cookie=CK)
check("GET .../files entry", {"id","receive_link_id","file_name","original_name","file_size",
      "mime_type","uploader_ip","uploader_agent","created_at","share_id"}, set(fl[0]))

# --- network / tunnel / webhook / stats ---
st, n = http("GET","/api/network", cookie=CK)
check("GET /api/network", {"localIp","tunnelUrl","tailscaleUrl","easytierIp","customUrl","port",
      "primaryNetwork","primaryUrl","maxFileSizeGB","networks"}, set(n))
check("GET /api/network networks{}", {"local","cloudflare","tailscale","easytier","custom"}, set(n["networks"]))
check("GET /api/network networks.local", {"enabled","url","detected"}, set(n["networks"]["local"]))
st, t = http("GET","/api/tunnel", cookie=CK)
check("GET /api/tunnel", {"config","isExternal","url"}, set(t))
check("GET /api/tunnel config{}", {"enabled","url","cloudflareUrl","tailscaleUrl","easytierIp",
      "customUrl","localIp","primaryNetwork","maxFileSizeGB","allowedExtensions","blockedExtensions"},
      set(t["config"]))
http("POST","/api/webhook", json.dumps({"enabled":True,"url":"https://hooks.example.com/casadrop",
     "on_download":True,"on_limit_reached":True,"on_expire":True,"secret":"hmac-secret"}),
     "application/json", CK)
st, w = http("GET","/api/webhook", cookie=CK)
check("GET /api/webhook", {"enabled","url","on_download","on_expire","on_limit_reached","secret_set"}, set(w))
checks += 1
if "secret" in w: fails.append("GET /api/webhook leaked the secret")
else: print("  ok  GET /api/webhook never returns the secret")
st, stt = http("GET","/api/stats", cookie=CK)
check("GET /api/stats", {"total_shares","total_downloads","total_size","protected_shares","expiring_soon"}, set(stt))

# --- metrics names ---
out = subprocess.run(["curl","-s","-b",CK,BASE+"/api/metrics"],capture_output=True,text=True).stdout
served = {l.split()[2] for l in out.splitlines() if l.startswith("# TYPE zima")}
doc_always = {"zima_downloads_total","zima_upload_bytes_total","zima_download_bytes_total",
  "zima_shares_created_total","zima_shares_deleted_total","zima_shares_expired_total",
  "zima_active_shares","zima_storage_used_bytes","zima_http_requests_total",
  "zima_http_request_duration_seconds"}
check("GET /api/metrics (always present)", doc_always, served)
checks += 1
if any(m.startswith("casadrop_") for m in out.splitlines()):
    fails.append("a casadrop_* metric exists after all")
else: print("  ok  no casadrop_* metric exists, as documented")

print()
print("%d checks, %d failures" % (checks, len(fails)))
for f in fails: print("  FAIL", f)
sys.exit(1 if fails else 0)
