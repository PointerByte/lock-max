#!/usr/bin/env python3
"""Initialize the dragon-cmk PostgreSQL database from SQL files.

This script intentionally uses the local `psql` client instead of a Python
database driver, so it does not require third-party Python dependencies or a
virtual environment.
"""

from __future__ import annotations

import argparse
import os
import shutil
import subprocess
import sys
from pathlib import Path


DB_DIR = Path(__file__).resolve().parent
PROJECT_ROOT = DB_DIR.parent


def parse_env_file(path: Path) -> dict[str, str]:
    values: dict[str, str] = {}
    if not path.exists():
        return values

    for line_number, raw_line in enumerate(path.read_text(encoding="utf-8").splitlines(), start=1):
        line = raw_line.strip()
        if not line or line.startswith("#"):
            continue
        if line.startswith("export "):
            line = line[len("export ") :].strip()
        if "=" not in line:
            raise ValueError(f"{path}:{line_number}: expected KEY=VALUE")

        key, value = line.split("=", 1)
        key = key.strip()
        value = value.strip()
        if not key:
            raise ValueError(f"{path}:{line_number}: empty key")
        if (value.startswith('"') and value.endswith('"')) or (
            value.startswith("'") and value.endswith("'")
        ):
            value = value[1:-1]
        values[key] = value

    return values


def load_config(env_files: list[Path]) -> dict[str, str]:
    config: dict[str, str] = {}
    for env_file in env_files:
        config.update(parse_env_file(env_file))

    # Real process environment has the highest priority.
    config.update(os.environ)
    return config


def sql_files_to_run() -> list[Path]:
    files: list[Path] = []
    seen: set[Path] = set()

    def add(path: Path) -> None:
        resolved = path.resolve()
        if path.exists() and resolved not in seen:
            files.append(path)
            seen.add(resolved)

    add(DB_DIR / "squema.sql")
    add(DB_DIR / "tables.sql")

    for path in sorted((DB_DIR / "views").rglob("*.sql")):
        add(path)
    for path in sorted((DB_DIR / "storeProrcedures").rglob("*.sql")):
        add(path)

    for path in sorted(DB_DIR.rglob("*.sql")):
        add(path)

    return files


def require_config(config: dict[str, str], key: str) -> str:
    value = config.get(key, "").strip()
    if not value:
        raise ValueError(f"missing required database config: {key}")
    return value


def connection_env(config: dict[str, str]) -> dict[str, str]:
    env = os.environ.copy()
    password = config.get("PGPASSWORD", "")
    if password:
        env["PGPASSWORD"] = password
    return env


def run_psql(args: argparse.Namespace, config: dict[str, str], sql_file: Path) -> None:
    command = [
        args.psql,
        "-v",
        "ON_ERROR_STOP=1",
        "--host",
        require_config(config, "PGHOST"),
        "--port",
        require_config(config, "PGPORT"),
        "--username",
        require_config(config, "PGUSER"),
        "--dbname",
        require_config(config, "PGDATABASE"),
        "--file",
        str(sql_file),
    ]

    print(f"==> {sql_file.relative_to(PROJECT_ROOT)}")
    if args.dry_run:
        print("    " + " ".join(command))
        return

    subprocess.run(command, env=connection_env(config), check=True)


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="Execute db/*.sql files against PostgreSQL in dependency order."
    )
    parser.add_argument(
        "--env-file",
        action="append",
        type=Path,
        help="Environment file to load. Can be used more than once.",
    )
    parser.add_argument(
        "--psql",
        default="psql",
        help="Path to the psql executable. Defaults to 'psql'.",
    )
    parser.add_argument(
        "--dry-run",
        action="store_true",
        help="Print the files and psql commands without connecting.",
    )
    return parser.parse_args()


def main() -> int:
    args = parse_args()
    env_files = args.env_file
    if not env_files:
        env_files = [PROJECT_ROOT / ".env", PROJECT_ROOT / ".env.local"]

    if not args.dry_run and shutil.which(args.psql) is None:
        print(f"psql executable not found: {args.psql}", file=sys.stderr)
        return 127

    try:
        config = load_config(env_files)
        files = sql_files_to_run()
        if not files:
            print("No SQL files found.")
            return 0

        for sql_file in files:
            run_psql(args, config, sql_file)
    except subprocess.CalledProcessError as exc:
        print(f"SQL execution failed with exit code {exc.returncode}", file=sys.stderr)
        return exc.returncode
    except Exception as exc:
        print(str(exc), file=sys.stderr)
        return 1

    print("Database initialization completed.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
