#!/bin/sh

set -e

echo "Starting Litestream restore process..."

# Remove existing database files
rm -f /app/chatsh.db /app/chatsh.db-shm /app/chatsh.db-wal

# Restore database with timeout and verbose logging
timeout 300 litestream restore -if-replica-exists -config /etc/litestream.yml /app/chatsh.db || {
    echo "Litestream restore failed or timed out, starting with empty database"
    # If restore fails, we'll let the application create a new database
}

echo "Starting application with Litestream replication..."
litestream replicate -exec /app/chatsh -config /etc/litestream.yml