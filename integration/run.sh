#!/usr/bin/env bash
set -Eeuo pipefail

cd "$(dirname "${BASH_SOURCE[0]}")"

port="${TRAEFIK_TEST_PORT:-18080}"
base_url="http://127.0.0.1:${port}"
compose=(docker compose --project-name traefik-queryparam-test)

cleanup() {
  "${compose[@]}" down --volumes --remove-orphans >/dev/null 2>&1 || true
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
    "${base_url}/matched?keep=original&replace=client" 2>/dev/null)"; then
    break
  fi
  sleep 1
done

if [[ -z "${matched_response}" ]]; then
  fail "Traefik did not return a response within 30 seconds"
fi

expected_request='GET /matched?added=plugin&keep=original&replace=configured&special=space+%26+symbols HTTP/1.1'
if ! grep --fixed-strings --quiet "${expected_request}" <<<"${matched_response}"; then
  fail "the matched router did not add, preserve, replace, and encode query parameters"
fi

unmatched_response="$(curl --fail --silent --show-error "${base_url}/unmatched?keep=original")"
if ! grep --fixed-strings --quiet 'GET /unmatched?keep=original HTTP/1.1' <<<"${unmatched_response}"; then
  fail "the unmatched router did not preserve its original query"
fi
if grep --fixed-strings --quiet 'added=plugin' <<<"${unmatched_response}"; then
  fail "the middleware changed a request on an unmatched router"
fi

printf 'Traefik integration test passed.\n'
