# Database initialization

Use `init-db.sh` from an environment that has `psql` available.

Required environment:

```bash
export PGHOST=localhost
export PGPORT=5432
export PGDATABASE=lock_max_db
export PGADMIN_USER=lock_max_user
export PGADMIN_PASSWORD='<admin-password>'
export PGPASSWORD='<dragon_cmk_user-password>'
```

The application runtime should connect with the role created by `squema.sql`:

```bash
export PGUSER=dragon_cmk_user
export PGPASSWORD='<dragon_cmk_user-password>'
```

Then run:

```bash
./storage/init-db.sh
```

The script creates the target database if it does not exist, then executes:

1. `storage/squema.sql`
2. `storage/tables.sql`
3. every `*.sql` file under `storage/storeProcedures`, sorted by path
4. every `*.sql` file under `storage/views`, sorted by path
