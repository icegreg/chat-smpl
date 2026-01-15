#!/bin/bash
set -e

# Database connection
DB_HOST="${DB_HOST:-postgres}"
DB_PORT="${DB_PORT:-5432}"
DB_USER="${DB_USER:-chatapp}"
DB_PASSWORD="${DB_PASSWORD:-secret}"
DB_NAME="${DB_NAME:-chatapp}"

export PGPASSWORD="$DB_PASSWORD"

echo "Waiting for PostgreSQL to be ready..."
until pg_isready -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" > /dev/null 2>&1; do
  echo "PostgreSQL is not ready yet. Waiting..."
  sleep 2
done

echo "PostgreSQL is ready. Running migrations..."

# Migration directories in order
MIGRATION_DIRS=(
  "/migrations/000_init_schema.sql"
  "/migrations/users"
  "/migrations/chat"
  "/migrations/files"
  "/migrations/voice"
  "/migrations/org"
  "/migrations/health"
)

# Run init schema first if exists
if [ -f "/migrations/000_init_schema.sql" ]; then
  echo "Applying: 000_init_schema.sql"
  psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -f "/migrations/000_init_schema.sql" || true
fi

# Run migrations from each directory
for dir in users chat files voice org health; do
  if [ -d "/migrations/$dir" ]; then
    echo "Applying migrations from: $dir/"
    # Sort files by name to ensure correct order
    for file in $(ls -1 /migrations/$dir/*.sql 2>/dev/null | sort); do
      echo "  - $(basename $file)"
      psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -f "$file" || true
    done
  fi
done

echo "All migrations completed successfully!"
