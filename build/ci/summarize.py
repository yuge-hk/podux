"""Persist real step outcomes; never infer a pass from a skipped step."""
import json
import os
from pathlib import Path

folder = Path("ci-results")
folder.mkdir(exist_ok=True)
mapping = {"success": "Passed", "failure": "Failed", "skipped": "Skipped", "cancelled": "Skipped"}
results = {k.lower(): mapping.get(os.getenv(k), "Skipped") for k in ["BUILD", "FRPC", "FRPC_RACE"]}
results["sha"] = os.getenv("GITHUB_SHA", "uncommitted")
results["run_url"] = f"https://github.com/{os.getenv('GITHUB_REPOSITORY', '')}/actions/runs/{os.getenv('GITHUB_RUN_ID', '')}"
failed_tests=[]
for path in folder.glob("*.json"):
    for line in path.read_text(errors="replace").splitlines():
        try: event=json.loads(line)
        except ValueError: continue
        if isinstance(event,dict) and event.get("Action")=="fail":
            failed_tests.append({"log":path.name,"package":event.get("Package"),"test":event.get("Test","package")})
results["failed_tests"]=failed_tests
(folder / "outcomes.json").write_text(json.dumps(results, indent=2))
summary = "\n".join(f"- {k}: {v}" for k, v in results.items()) + "\n"
if os.getenv("GITHUB_STEP_SUMMARY"):
    with open(os.environ["GITHUB_STEP_SUMMARY"], "a") as out:
        out.write(summary)
print(summary)
