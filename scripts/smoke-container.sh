#!/bin/sh
set -eu
image=${1:?Usage: smoke-container.sh IMAGE}
container="leafmark-smoke-${GITHUB_RUN_ID:-$$}"
volume="${container}-data"
page=$(mktemp)
trap 'docker rm -f "$container" >/dev/null 2>&1 || true; docker volume rm "$volume" >/dev/null 2>&1 || true; rm -f "$page"' EXIT
docker run -d --name "$container" --read-only --tmpfs /tmp --cap-drop ALL --security-opt no-new-privileges:true --mount "type=volume,source=$volume,target=/data" -e PORT=8080 -e COOKIE_SECURE=false -p 127.0.0.1:18094:8080 "$image" >/dev/null
wait_ready() {
    for attempt in 1 2 3 4 5 6 7 8 9 10; do
        if curl --fail --silent http://127.0.0.1:18094/v1/readyz >/dev/null; then return 0; fi
        sleep 1
    done
    docker logs "$container"
    return 1
}
wait_ready
curl --fail --silent http://127.0.0.1:18094/ > "$page"
if ! grep -q 'Leafmark' "$page"; then
    printf '%s\n' 'Smoke test failed: Leafmark page missing.' >&2
    exit 1
fi
curl --fail --silent http://127.0.0.1:18094/v1/healthz >/dev/null
python3 scripts/check-api.py http://127.0.0.1:18094 seed
docker restart "$container" >/dev/null
wait_ready
python3 scripts/check-api.py http://127.0.0.1:18094 verify
printf '%s\n' 'Container HTTP, account, note, export, and restart persistence checks passed.'
