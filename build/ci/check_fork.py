"""Read-only fork/identity/Actions preflight. Does not commit, add remotes or push."""
import argparse
import json
import re
import subprocess

p = argparse.ArgumentParser()
p.add_argument("repository", help="Personal owner/podux or https://github.com/owner/podux")
p.add_argument("--gh", default="gh")
a = p.parse_args()
repo = a.repository.removeprefix("https://github.com/").removesuffix(".git").rstrip("/")
if not re.fullmatch(r"[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+", repo) or repo.lower() == "luckjiawei/podux":
    raise SystemExit("Refusing an invalid target or the upstream repository")
def api(path):
    return json.loads(subprocess.check_output([a.gh, "api", path], text=True))
user = api("user")["login"]
data = api("repos/" + repo)
if not data["fork"] or data.get("parent", {}).get("full_name", "").lower() != "luckjiawei/podux":
    raise SystemExit("Not a direct Podux fork")
if data["owner"]["login"].lower() != user.lower() or not data.get("permissions", {}).get("push"):
    raise SystemExit("Not the authenticated user's writable personal fork")
actions = api("repos/" + repo + "/actions/permissions")
if not actions.get("enabled"):
    raise SystemExit("Fork Actions are not enabled")
print(json.dumps({"repository": data["full_name"], "authenticatedOwner": user, "parent": data["parent"]["full_name"], "writePermission": True, "actionsEnabled": True, "defaultBranch": data["default_branch"], "testBranch": "feat/local-cloudflare-tunnels", "approval": "Still requires explicit user authorization before first commit/push"}, indent=2))
