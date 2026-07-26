#!/usr/bin/env bash
set -Eeuo pipefail

cd "$(dirname "${BASH_SOURCE[0]}")"

port="${TRAEFIK_TEST_PORT:-18080}"
base_url="http://127.0.0.1:${port}"
project_name="traefik-queryparam-test"
compose_file="docker-compose.yml"
remote_config=""

if [[ -n "${TRAEFIK_REMOTE_PLUGIN_VERSION:-}" ]]; then
  if [[ ! "${TRAEFIK_REMOTE_PLUGIN_VERSION}" =~ ^v[0-9]+\.[0-9]+\.[0-9]+-pr\.[0-9]+\+[0-9a-f]{40}$ ]]; then
    printf 'Invalid prerelease plugin version: %s\n' "${TRAEFIK_REMOTE_PLUGIN_VERSION}" >&2
    exit 1
  fi

  remote_config="$(mktemp)"
  cat >"${remote_config}" <<EOF
entryPoints:
  web:
    address: :80

providers:
  file:
    filename: /etc/traefik/dynamic.yml

experimental:
  plugins:
    QueryParamsPlugin:
      moduleName: github.com/SlothCroissant/traefik-queryparam-plugin
      version: "${TRAEFIK_REMOTE_PLUGIN_VERSION}"

log:
  level: INFO
EOF

  registry_url="https://plugins.traefik.io/public/download/github.com/SlothCroissant/traefik-queryparam-plugin/${TRAEFIK_REMOTE_PLUGIN_VERSION}"
  for _ in $(seq 1 "${TRAEFIK_PLUGIN_REGISTRY_ATTEMPTS:-180}"); do
    if curl --fail --silent --show-error --output /dev/null "${registry_url}" 2>/dev/null; then
      break
    fi
    sleep 10
  done
  if ! curl --fail --silent --show-error --output /dev/null "${registry_url}"; then
    printf 'Traefik Plugin Registry did not publish %s within 30 minutes\n' "${TRAEFIK_REMOTE_PLUGIN_VERSION}" >&2
    exit 1
  fi

  compose_file="docker-compose.remote.yml"
  project_name="traefik-queryparam-remote-plugin-test"
  export TRAEFIK_CONFIG_FILE="${remote_config}"
fi

compose=(docker compose --project-name "${project_name}" -f "${compose_file}")

cleanup() {
  "${compose[@]}" down --volumes --remove-orphans >/dev/null 2>&1 || true
  rm -f "${remote_config}"
}

fail() {
  printf 'Integration test failed: %s\n\n' "$1" >&2
  "${compose[@]}" logs --no-color >&2 || true
  exit 1
}

trap cleanup EXIT
cleanup
"${compose[@]}" up --detach

matched_response=""
for _ in $(seq 1 30); do
  if matched_response="$(curl --fail --silent --show-error \
    "${base_url}/matched?keep=original&replace=client&remove=first&remove=second&remove-all=first&remove-all=second" 2>/dev/null)"; then
    break
  fi
  sleep 1
done

if [[ -z "${matched_response}" ]]; then
  fail "Traefik did not return a response within 30 seconds"
fi

expected_request='GET /matched?added=plugin&keep=original&remove=second&replace=configured&special=space+%26+symbols HTTP/1.1'
if ! grep --fixed-strings --quiet "${expected_request}" <<<"${matched_response}"; then
  fail "the matched router did not add, remove, preserve, replace, and encode query parameters"
fi

unmatched_response="$(curl --fail --silent --show-error "${base_url}/unmatched?keep=original")"
if ! grep --fixed-strings --quiet 'GET /unmatched?keep=original HTTP/1.1' <<<"${unmatched_response}"; then
  fail "the unmatched router did not preserve its original query"
fi
if grep --fixed-strings --quiet 'added=plugin' <<<"${unmatched_response}"; then
  fail "the middleware changed a request on an unmatched router"
fi

if [[ -n "${TRAEFIK_REMOTE_PLUGIN_VERSION:-}" ]]; then
  printf 'Traefik remote plugin integration test passed.\n'
else
  printf 'Traefik local plugin integration test passed.\n'
fi
