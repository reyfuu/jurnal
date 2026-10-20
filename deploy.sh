#!/usr/bin/env bash
set -euo pipefail

# Laporan Bisnis by Master Pajak - VPS Deployment Script
# Usage: ./deploy.sh [PORT]

APP_PORT="${1:-8085}"
# Password akun seed produksi; simpan di .env (tidak di-commit) atau export sebelum deploy.
# MIDTRANS_SERVER_KEY / MIDTRANS_CLIENT_KEY (opsional, di .env): pembayaran langganan;
# MIDTRANS_PRODUCTION=true untuk akun produksi (default sandbox).
# PG_PASSWORD: password user Postgres aplikasi; PG_RO_PASSWORD (opsional): user baca-saja
# dbeaver_ro untuk DBeaver. Buat dengan `openssl rand -hex 24` dan simpan di .env.
[ -f .env ] && . ./.env
: "${SEED_PASSWORD:?SEED_PASSWORD wajib di-set (export atau .env) sebelum deploy}"
: "${PG_PASSWORD:?PG_PASSWORD wajib di-set (.env) sebelum deploy}"
IMAGE_NAME="laporan-bisnis:latest"
CONTAINER_NAME="laporan-bisnis"
NETWORK="laporan-bisnis-net"
DB_CONTAINER="laporan-bisnis-db"
DB_PORT="${PG_HOST_PORT:-5432}" # hanya 127.0.0.1: akses dari luar lewat SSH tunnel

echo "=== [1/4] Pulling latest updates from origin main ==="
git pull origin main

echo "=== [2/4] Building Docker container ==="
docker build -t "$IMAGE_NAME" .

echo "=== [3/4] Postgres + restarting container on port $APP_PORT ==="
docker network inspect "$NETWORK" >/dev/null 2>&1 || docker network create "$NETWORK"
if ! docker container inspect "$DB_CONTAINER" >/dev/null 2>&1; then
  docker run -d \
    --name "$DB_CONTAINER" \
    --network "$NETWORK" \
    --restart unless-stopped \
    -p "127.0.0.1:${DB_PORT}:5432" \
    -e POSTGRES_USER=laporan \
    -e POSTGRES_PASSWORD="$PG_PASSWORD" \
    -e POSTGRES_DB=laporan_bisnis \
    -v laporan-bisnis-pg:/var/lib/postgresql/data \
    postgres:17-alpine
fi
docker start "$DB_CONTAINER" >/dev/null
until docker exec "$DB_CONTAINER" pg_isready -U laporan -d laporan_bisnis >/dev/null 2>&1; do sleep 1; done

# data/: berkas JSON lama (diimpor sekali ke Postgres saat DB masih kosong) & screenshot bug.
mkdir -p data
docker rm -f "$CONTAINER_NAME" 2>/dev/null || true
docker run -d \
  --name "$CONTAINER_NAME" \
  --network "$NETWORK" \
  --restart unless-stopped \
  -p "127.0.0.1:${APP_PORT}:8080" \
  -e SEED_PASSWORD="$SEED_PASSWORD" \
  -e DATABASE_URL="postgres://laporan:${PG_PASSWORD}@${DB_CONTAINER}:5432/laporan_bisnis?sslmode=disable" \
  -e MIDTRANS_SERVER_KEY="${MIDTRANS_SERVER_KEY:-}" \
  -e MIDTRANS_CLIENT_KEY="${MIDTRANS_CLIENT_KEY:-}" \
  -e MIDTRANS_PRODUCTION="${MIDTRANS_PRODUCTION:-false}" \
  -e DATA_DIR=/data \
  -v "$(pwd)/data:/data" \
  "$IMAGE_NAME"

echo "=== [4/4] Verifying deployment ==="
sleep 2
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "http://localhost:${APP_PORT}/" || echo "000")
if [ "$HTTP_CODE" = "200" ]; then
  echo "✓ Laporan Bisnis successfully deployed & running on http://localhost:${APP_PORT}"
else
  echo "⚠ Warning: Expected HTTP 200, got: $HTTP_CODE"
fi

# Tabel dibuat aplikasi saat boot; hak baca dbeaver_ro diberikan setelahnya.
if [ -n "${PG_RO_PASSWORD:-}" ]; then
  docker exec -i "$DB_CONTAINER" psql -q -U laporan -d laporan_bisnis -v ON_ERROR_STOP=1 -v pw="$PG_RO_PASSWORD" <<'SQL'
SELECT 'CREATE ROLE dbeaver_ro LOGIN' WHERE NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'dbeaver_ro') \gexec
ALTER ROLE dbeaver_ro PASSWORD :'pw';
GRANT CONNECT ON DATABASE laporan_bisnis TO dbeaver_ro;
GRANT USAGE ON SCHEMA public TO dbeaver_ro;
GRANT SELECT ON ALL TABLES IN SCHEMA public TO dbeaver_ro;
ALTER DEFAULT PRIVILEGES FOR ROLE laporan IN SCHEMA public GRANT SELECT ON TABLES TO dbeaver_ro;
SQL
  echo "✓ User baca-saja dbeaver_ro siap (DBeaver lewat SSH tunnel ke 127.0.0.1:${DB_PORT})"
fi

# Backup harian 02:15, simpan 14 hari.
cat > /etc/cron.d/laporan-bisnis-backup <<CRON
15 2 * * * root mkdir -p /var/backups/laporan-bisnis && docker exec $DB_CONTAINER pg_dump -U laporan -Fc laporan_bisnis > /var/backups/laporan-bisnis/\$(date +\%F).dump && find /var/backups/laporan-bisnis -name '*.dump' -mtime +14 -delete
CRON
