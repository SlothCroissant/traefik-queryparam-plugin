#!/bin/sh
set -eu

destination="${PLUGIN_DESTINATION:-/plugins-local}"

mkdir -p "${destination}"
rm -rf "${destination:?}/"* "${destination:?}/".[!.]* "${destination:?}/"..?*
cp -a /plugin/. "${destination}/"
