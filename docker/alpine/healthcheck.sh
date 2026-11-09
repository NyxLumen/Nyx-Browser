#!/bin/sh

set -e

PORT=${NYX_PORT:-${FB_PORT:-${FILEBROWSER_PORT:-$(cat /config/settings.json | sh /JSON.sh | grep '\["port"\]' | awk '{print $2}')}}}
ADDRESS=${NYX_ADDRESS:-${FB_ADDRESS:-${FILEBROWSER_ADDRESS:-$(cat /config/settings.json | sh /JSON.sh | grep '\["address"\]' | awk '{print $2}' | sed 's/"//g')}}}
ADDRESS=${ADDRESS:-localhost}

wget -q --spider http://$ADDRESS:$PORT/health || exit 1
