#!/usr/bin/env python3
"""Exercise disposable accounts against a smoke-test instance, never a user's database."""
import http.cookiejar
import json
import sys
import urllib.request

base, phase = sys.argv[1:]
jar = http.cookiejar.CookieJar()
client = urllib.request.build_opener(urllib.request.HTTPCookieProcessor(jar))


def request(path, payload=None):
    data = json.dumps(payload).encode() if payload is not None else None
    req = urllib.request.Request(base + path, data=data, headers={"Content-Type": "application/json"})
    with client.open(req, timeout=10) as response:
        return json.load(response)


credentials = {"username": "container-smoke", "password": "disposable smoke notebook phrase"}
if phase == "seed":
    request("/v1/auth/register", {**credentials, "name": "Container smoke test"})
    note = request("/v1/notes", {"title": "Persistence check", "note": "Survives container restart"})
    assert note["version"] == 1
else:
    request("/v1/auth/login", credentials)
    notes = request("/v1/notes")
    assert notes["total"] == 1
    assert notes["notes"][0]["note"] == "Survives container restart"
    export = request("/v1/export")
    assert len(export["notes"]) == 1
print(f"Container API {phase} passed.")
