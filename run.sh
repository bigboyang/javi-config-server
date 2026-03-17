#!/bin/bash

set -e

APP_NAME="config-server"
BINARY="./$APP_NAME"

echo "Building $APP_NAME..."
go build -o "$APP_NAME" .

echo "Starting $APP_NAME..."
exec "$BINARY"
