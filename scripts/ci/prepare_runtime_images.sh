#!/usr/bin/env bash

set -euo pipefail

echo "Pulling compose runtime images..."
docker pull postgres:15
docker pull redis:7-alpine
docker pull nginx:1.25-alpine

echo "Runtime images are ready."
