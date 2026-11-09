#!/bin/sh

set -e

PORT=${NYX_PORT:-${FB_PORT:-${FILEBROWSER_PORT:-$(jq -r .port /config/settings.json)}}}
ADDRESS=${NYX_ADDRESS:-${FB_ADDRESS:-${FILEBROWSER_ADDRESS:-$(jq -r .address /config/settings.json)}}}
ADDRESS=${ADDRESS:-localhost}

wget -q --spider http://$ADDRESS:$PORT/health || exit 1
