#!/usr/bin/env bash
set -euo pipefail

timeout_seconds=240
run_test="TestNeo4jProductionTracer"
image="neo4j:5.26-community"

usage() {
  printf 'usage: %s [--timeout-seconds N] [--run TestName]\n' "$0" >&2
}

while [ "$#" -gt 0 ]; do
  case "$1" in
    --timeout-seconds)
      [ "$#" -ge 2 ] || { usage; exit 2; }
      timeout_seconds=$2
      shift 2
      ;;
    --run)
      [ "$#" -ge 2 ] || { usage; exit 2; }
      run_test=$2
      shift 2
      ;;
    *)
      usage
      exit 2
      ;;
  esac
done

case "$timeout_seconds" in
  ''|*[!0-9]*) printf 'timeout must be a positive integer\n' >&2; exit 2 ;;
  0) printf 'timeout must be a positive integer\n' >&2; exit 2 ;;
esac
if [[ ! "$run_test" =~ ^Test[A-Za-z0-9_]*$ ]]; then
  printf 'invalid exact Go test identifier\n' >&2
  exit 2
fi

go_cmd=${GORTEX_NEO4J_TEST_GO:-go}
docker_cmd=${GORTEX_NEO4J_TEST_DOCKER:-docker}
package=./internal/neo4jprojection
selector="^${run_test}$"

listed=$(GOWORK=off "$go_cmd" test "$package" -list "$selector")
printf '%s\n' "$listed" | grep -qx "$run_test" || {
  printf 'exact integration test not found: %s\n' "$run_test" >&2
  exit 3
}

nonce="${$}-${RANDOM}-${RANDOM}"
container="gortex-neo4j-${nonce}"
network="gortex-neo4j-${nonce}"
password=$(printf '%s' "${nonce}-${RANDOM}" | shasum -a 256 | cut -c1-32)
container_id=""

cleanup() {
  if [ -n "$container_id" ]; then
    "$docker_cmd" rm -f "$container" >/dev/null 2>&1 || true
  fi
  "$docker_cmd" network rm "$network" >/dev/null 2>&1 || true
}
trap cleanup EXIT HUP INT TERM

"$docker_cmd" network create "$network" >/dev/null
container_id=$("$docker_cmd" run -d --name "$container" --network "$network" -p 127.0.0.1::7687 \
  -e "NEO4J_AUTH=neo4j/${password}" "$image")

deadline=$((SECONDS + timeout_seconds))
until "$docker_cmd" exec "$container" cypher-shell -u neo4j -p "$password" \
  'RETURN 1' >/dev/null 2>&1; do
  if [ "$SECONDS" -ge "$deadline" ]; then
    printf 'Neo4j readiness timed out\n' >&2
    exit 4
  fi
  sleep 1
done

remaining=$((deadline - SECONDS))
if [ "$remaining" -le 0 ]; then
  printf 'Neo4j test deadline expired before execution\n' >&2
  exit 5
fi

host_port=$("$docker_cmd" port "$container" 7687/tcp | sed 's/.*://')
[ -n "$host_port" ] || { printf 'Neo4j Bolt port was not published\n' >&2; exit 4; }

GORTEX_NEO4J_INTEGRATION=1 \
GORTEX_NEO4J_REQUIRE_SCENARIOS="${GORTEX_NEO4J_REQUIRE_SCENARIOS:-1}" \
GORTEX_NEO4J_URI="bolt://127.0.0.1:${host_port}" \
GORTEX_NEO4J_USERNAME=neo4j \
GORTEX_NEO4J_PASSWORD="$password" \
GOWORK=off "$go_cmd" test "$package" -run "$selector" -count=1 &
test_pid=$!
while kill -0 "$test_pid" 2>/dev/null; do
  if [ "$SECONDS" -ge "$deadline" ]; then
    kill "$test_pid" 2>/dev/null || true
    wait "$test_pid" 2>/dev/null || true
    printf 'Neo4j integration test timed out\n' >&2
    exit 5
  fi
  sleep 1
done
wait "$test_pid"
