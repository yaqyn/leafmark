# Operations

## Deployment boundary

The repository supplies a runnable app and container. No public host, domain, external database upgrade, or remote repository rename is performed by the local build. Verify your own environment before accepting real users.

The default target is a single app instance behind an HTTPS reverse proxy. The app does not trust forwarded IP headers. Apply per-client request and login limits at the proxy; its own login limiter sees the immediate network peer. Restrict access to the application port, terminate TLS at the proxy, preserve Host/Origin headers, and set `COOKIE_SECURE=true`. For a private notebook, create accounts and then set `ALLOW_SIGNUP=false`.

Container runs as UID/GID 10001 with no root requirement. Compose drops capabilities, uses a read-only root filesystem and a writable database volume, and binds only to loopback. Put secrets into deployment-managed environment configuration. Never put a Turso token in the image, source, screenshot, or shell history. The Docker build excludes `.env` and database files.

## Startup and health

Startup pings the database and applies schema migrations before opening the listener. Failure exits without serving partial functionality. Migrations are additive, transactional per file, and tracked in `leafmark_migrations`. Course-era Goose tables remain untouched. Do not use Goose to roll back a schema managed by Leafmark.

`GET /v1/healthz` means the process responds. `GET /v1/readyz` performs a database ping with a two-second deadline. Use readiness for routing and liveness for process supervision. The server bounds headers, JSON bodies, and HTTP read/write/idle time. SIGTERM permits ten seconds for active requests before shutdown.

## Backups and restore drill

For local Compose SQLite, stop the app to produce a consistent snapshot:

```sh
umask 077
docker compose stop leafmark
mkdir -p backups
docker compose run --rm --no-deps --user 0 --entrypoint tar leafmark -C /data -czf - . > backups/leafmark-data.tar.gz
docker compose start leafmark
```

The archive contains sensitive account hashes, session hashes, and note content. Restrict permissions (`chmod 600 backups/leafmark-data.tar.gz`), encrypt off-device backups, and follow your own retention policy. Backups are deliberately excluded from Git. This brief stop is appropriate for a personal deployment; use SQLite's online backup API or provider backups when uptime requirements demand it.

Restore into a new isolated volume before touching the original:

```sh
docker volume create leafmark-restore-test
docker run --rm -i --user 0 -v leafmark-restore-test:/data --entrypoint tar leafmark:local -C /data -xzf - < backups/leafmark-data.tar.gz
docker run --rm --name leafmark-restore-check -e COOKIE_SECURE=false -e ALLOW_SIGNUP=false -p 127.0.0.1:18095:8080 -v leafmark-restore-test:/data leafmark:local
```

In another terminal, verify `curl -f http://localhost:18095/v1/readyz`, sign in, and verify real notes and an export. Stop the test server before removing the test volume. For Turso, use the provider's backup/restore process and test against an isolated restored database. A note export is portable content, not a complete account/database backup.

## Upgrade and rollback

1. Back up and test restoration; retain the prior image and database snapshot.
2. Run CI checks and build the new image. Verify startup against a restored copy first.
3. Stop the previous instance, install the new image with the same data mount and configuration, and verify readiness plus sign-in, edit, search, export, and restart persistence.
4. If rollout fails, stop the new instance. Restore the pre-upgrade database snapshot and prior image together. Do not run down-migrations against live notes.

No automatic migration is run against your existing remote database during local tests. Starting the upgraded app with its DATABASE_URL is the action that applies the schema.

## Known operating limits

No automated account recovery, email verification, distributed rate limiter, automatic backup schedule, data-at-rest encryption managed by the app, end-to-end encryption, or high-availability topology. Disk/provider encryption, monitoring, retention, and recovery ownership belong to the deployment operator. The JSON export is collected in memory; very large notebooks need a streaming export and explicit quotas before a public multi-user launch.
