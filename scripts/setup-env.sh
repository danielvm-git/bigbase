#!/usr/bin/env bash
# Setup environment script for BigBase development and testing.
# Sets up common variables and paths.
# Idempotent: safe to run multiple times.

set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
export ROOT
export PATH="$ROOT/bin:$PATH" # Prepend bin to PATH

# Common ports
PORT="${PORT:-9999}" # Use external PORT if set, else default to 9999
export PORT
PORT_CLI="${PORT_CLI:-$(($PORT + 1))}"
export PORT_CLI
PORT_DEBUG="${PORT_DEBUG:-$(($PORT + 2))}"
export PORT_DEBUG

# Default paths
DB_PATH="${DB_PATH:-$ROOT/data/bigbase.db}"
export DB_PATH
BUILD_DIR="${BUILD_DIR:-$ROOT/data/builds}"
export BUILD_DIR
LOG_DIR="${LOG_DIR:-$ROOT/data/logs}"
export LOG_DIR
CONFIG_DIR="${CONFIG_DIR:-$ROOT/config}"
export CONFIG_DIR

# Default domain
SITES_DOMAIN="${SITES_DOMAIN:-localhost.localdomain}"
export SITES_DOMAIN

# Google OAuth
GOOGLE_CLIENT_ID="${GOOGLE_CLIENT_ID:-}"
export GOOGLE_CLIENT_ID
GOOGLE_CLIENT_SECRET="${GOOGLE_CLIENT_SECRET:-}"
export GOOGLE_CLIENT_SECRET

echo "Setup complete"
echo "ROOT=$ROOT"
echo "PORT=$PORT"
echo "DB_PATH=$DB_PATH"
echo "BUILD_DIR=$BUILD_DIR"
echo "LOG_DIR=$LOG_DIR"
echo "CONFIG_DIR=$CONFIG_DIR"
echo "SITES_DOMAIN=$SITES_DOMAIN"