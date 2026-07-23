#!/bin/bash
set -e

# This script runs automatically on the first boot of the Postgres container
# and securely locks down the public schema, aligning with our dynamic .env configuration.

psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" <<-EOSQL
    REVOKE ALL ON SCHEMA public FROM PUBLIC;
    GRANT ALL ON SCHEMA public TO "$POSTGRES_USER";
    GRANT ALL PRIVILEGES ON DATABASE "$POSTGRES_DB" TO "$POSTGRES_USER";
EOSQL

echo "✅ Postgres dynamically initialized via init.sh using environment configuration."
