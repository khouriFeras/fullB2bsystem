#!/bin/bash
# Create GetDeliveryStatus/.env from .env.example if it doesn't exist.
# Run from repo root: ./deploy/setup-getdeliverystatus-env.sh

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
ENV_FILE="$REPO_ROOT/GetDeliveryStatus/.env"
EXAMPLE="$REPO_ROOT/GetDeliveryStatus/.env.example"

if [ -f "$ENV_FILE" ]; then
    echo "GetDeliveryStatus/.env already exists."
    exit 0
fi

if [ ! -f "$EXAMPLE" ]; then
    echo "Error: GetDeliveryStatus/.env.example not found."
    exit 1
fi

cp "$EXAMPLE" "$ENV_FILE"
echo "Created GetDeliveryStatus/.env from .env.example"
echo "Edit GetDeliveryStatus/.env to add your Wassel API credentials for delivery lookup."
echo "Until then, the service will start but shipment lookups will fail."
