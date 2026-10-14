#!/bin/sh
set -eu

image=${1:?Usage: smoke-container.sh IMAGE}
container="notely-smoke-${GITHUB_RUN_ID:-local}"
page=$(mktemp)
trap 'docker rm -f "$container" >/dev/null 2>&1 || true; rm -f "$page"' EXIT
docker run -d --name "$container" -e PORT=8080 -p 127.0.0.1:18080:8080 "$image"
for attempt in 1 2 3 4 5; do
    if curl --fail --silent http://127.0.0.1:18080/ > "$page"; then
        if ! grep -q 'Welcome to Notely' "$page"; then
            printf '%s\n' 'Smoke test failed: expected heading missing.' >&2
            exit 1
        fi
        curl --fail --silent http://127.0.0.1:18080/v1/healthz
        printf '\n%s\n' 'Container HTTP smoke checks passed.'
        exit 0
    fi
    sleep 1
done
docker logs "$container"
exit 1
