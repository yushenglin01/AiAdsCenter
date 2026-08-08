#!/usr/bin/env sh
set -eu
docker compose up -d --build
docker compose ps
printf '\nAdNova · 星曜智投: http://localhost:${WEB_PORT:-5173}\n'
