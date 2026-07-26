#!/bin/sh
set -eu

destination="${PLUGIN_DESTINATION:-/plugins-local/src/github.com/SlothCroissant/traefik-queryparam-plugin}"

rm -rf "${destination}"
mkdir -p "$(dirname "${destination}")"
cp -a /plugin "${destination}"
