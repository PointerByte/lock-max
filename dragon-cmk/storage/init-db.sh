#!/usr/bin/env sh
# Copyright 2026 PointerByte Contributors
# SPDX-License-Identifier: Apache-2.0

set -eu

SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
STORE_PROCEDURES_DIR="${SCRIPT_DIR}/storeProcedures"
VIEWS_DIR="${SCRIPT_DIR}/views"

: "${PGHOST:=localhost}"
: "${PGPORT:=5432}"
: "${PGDATABASE:=lock_max_db}"
: "${PGADMIN_USER:=lock_max_user}"
: "${PGADMIN_PASSWORD:?PGADMIN_PASSWORD is required}"
: "${PGADMIN_DATABASE:=postgres}"
: "${PGPASSWORD:?PGPASSWORD is required}"

psql_admin() {
  PGPASSWORD="${PGADMIN_PASSWORD}" \
  psql \
    -v ON_ERROR_STOP=1 \
    --host "$PGHOST" \
    --port "$PGPORT" \
    --username "$PGADMIN_USER" \
    "$@"
}

run_file() {
  file=$1
  echo "Running ${file}"
  psql_admin \
    --dbname "$PGDATABASE" \
    --set "dragon_cmk_user_password=${PGPASSWORD}" \
    --file "$file"
}

ensure_database() {
  echo "Ensuring database ${PGDATABASE}"
  psql_admin \
    --dbname "$PGADMIN_DATABASE" \
    --set "database_name=${PGDATABASE}" <<'SQL'
SELECT format('CREATE DATABASE %I', :'database_name')
WHERE NOT EXISTS (
  SELECT 1
  FROM pg_database
  WHERE datname = :'database_name'
)\gexec
SQL
}

run_sql_dir() {
  dir=$1

  if [ ! -d "$dir" ]; then
    echo "Directory not found: ${dir}" >&2
    exit 1
  fi

  find "$dir" -type f -name '*.sql' | sort | while IFS= read -r file; do
    run_file "$file"
  done
}

ensure_database
run_file "${SCRIPT_DIR}/squema.sql"
run_file "${SCRIPT_DIR}/tables.sql"
run_sql_dir "$STORE_PROCEDURES_DIR"
run_sql_dir "$VIEWS_DIR"

echo "Database initialization completed"
