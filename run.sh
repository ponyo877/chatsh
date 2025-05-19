#!/bin/sh

set -e

rm -f /app/chatsh.db
litestream restore -if-replica-exists -config /etc/litestream.yml /app/chatsh.db
litestream replicate -exec /app/chatsh -config /etc/litestream.yml