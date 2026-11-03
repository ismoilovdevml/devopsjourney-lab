#!/usr/bin/env bash
# End-to-end smoke test against a running stack (`make up`).
#
# Usage: scripts/smoke.sh
#
# Environment:
#   BASE_URL        public URL of the stack            (default http://localhost:8088)
#   ADMIN_USERNAME  admin login                        (default: from ./.env, else "admin")
#   ADMIN_PASSWORD  admin password                     (default: from ./.env; required)
#   SOLUTION        reference solution for task 1     (default e2e/fixtures/solutions/nginx-health.sh)
#
# Needs: curl, jq, docker (the solution is applied with `docker exec` into the
# lab container labelled devopslab.session=<session id>).
#
# Every assertion prints PASS/FAIL; the script exits non-zero if any failed.
# It creates one random student account and one lab session; the session is
# deleted at the end (also on failure).
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BASE_URL="${BASE_URL:-http://localhost:8088}"
BASE_URL="${BASE_URL%/}"
SOLUTION="${SOLUTION:-${repo_root}/e2e/fixtures/solutions/nginx-health.sh}"

# Read a single KEY=value from .env without sourcing (values are never executed).
env_file_value() {
  local key="$1" file="${repo_root}/.env"
  [[ -f "${file}" ]] || return 0
  grep -E "^${key}=" "${file}" | tail -n 1 | cut -d= -f2- | sed -e 's/^"//' -e 's/"$//'
}
ADMIN_USERNAME="${ADMIN_USERNAME:-$(env_file_value ADMIN_USERNAME)}"
ADMIN_USERNAME="${ADMIN_USERNAME:-admin}"
ADMIN_PASSWORD="${ADMIN_PASSWORD:-$(env_file_value ADMIN_PASSWORD)}"

for bin in curl jq docker; do
  command -v "${bin}" >/dev/null 2>&1 || { echo "error: ${bin} is required" >&2; exit 2; }
done
[[ -n "${ADMIN_PASSWORD}" ]] || { echo "error: ADMIN_PASSWORD not set (env or .env)" >&2; exit 2; }
[[ -f "${SOLUTION}" ]] || { echo "error: solution script ${SOLUTION} not found" >&2; exit 2; }

work="$(mktemp -d)"
student_jar="${work}/student.jar"
admin_jar="${work}/admin.jar"
body="${work}/body"
headers="${work}/headers"
session_id=""

failures=0
passes=0
pass() { passes=$((passes + 1)); printf 'PASS  %s\n' "$1"; }
fail() { failures=$((failures + 1)); printf 'FAIL  %s\n' "$1"; [[ -n "${2:-}" ]] && printf '      %s\n' "$2"; return 0; }

cleanup() {
  if [[ -n "${session_id}" ]]; then
    curl -sS -o /dev/null -b "${student_jar}" -c "${student_jar}" -H "Origin: ${BASE_URL}" \
      -X DELETE "${BASE_URL}/api/sessions/${session_id}" || true
  fi
  rm -rf "${work}"
}
trap cleanup EXIT

# req JAR METHOD PATH [JSON] [ORIGIN] -> prints the HTTP status; body in $body, headers in $headers.
req() {
  local jar="$1" method="$2" path="$3" data="${4:-}" origin="${5:-${BASE_URL}}"
  local args=(-sS -o "${body}" -D "${headers}" -w '%{http_code}' -X "${method}"
    -b "${jar}" -c "${jar}" -H "Origin: ${origin}" -H 'Accept: application/json')
  if [[ -n "${data}" ]]; then
    args+=(-H 'Content-Type: application/json' --data "${data}")
  fi
  curl "${args[@]}" "${BASE_URL}${path}" || true # -w prints 000 on transport errors
}

# expect_status NAME WANT GOT
expect_status() {
  if [[ "$3" == "$2" ]]; then pass "$1 ($3)"; else fail "$1: got HTTP $3, want $2" "$(head -c 300 "${body}" 2>/dev/null)"; fi
}

# expect_jq NAME FILTER: passes when jq -e FILTER is truthy on the last body.
expect_jq() {
  if jq -e "$2" "${body}" >/dev/null 2>&1; then pass "$1"; else fail "$1" "$(head -c 300 "${body}")"; fi
}

echo "Smoke test against ${BASE_URL}"
: >"${student_jar}"
: >"${admin_jar}"

# --- health and headers -------------------------------------------------------
expect_status "GET /healthz" 200 "$(req "${student_jar}" GET /healthz)"
expect_status "GET /readyz" 200 "$(req "${student_jar}" GET /readyz)"

req "${student_jar}" GET / >/dev/null
for h in "Content-Security-Policy" "X-Frame-Options" "X-Content-Type-Options: nosniff"; do
  if grep -qi "^${h}" "${headers}"; then pass "security header ${h%%:*}"; else fail "security header ${h%%:*} missing"; fi
done

# --- student account ----------------------------------------------------------
suffix="$(LC_ALL=C tr -dc 'a-z0-9' </dev/urandom | head -c 8 || true)"
student="smoke-${suffix}"
password="Smoke-${suffix}-Pw1"
expect_status "register ${student}" 201 \
  "$(req "${student_jar}" POST /api/auth/register "{\"username\":\"${student}\",\"email\":\"${student}@example.com\",\"password\":\"${password}\"}")"
expect_jq "register body has no token" '(.user.username != null) and (has("token") | not)'

expect_status "GET /api/me" 200 "$(req "${student_jar}" GET /api/me)"
expect_jq "/api/me is the new student" ".username == \"${student}\" and .role == \"student\""

# --- tasks ----------------------------------------------------------------------
expect_status "GET /api/tasks" 200 "$(req "${student_jar}" GET /api/tasks)"
expect_jq "6 tasks" 'length == 6'
if grep -q 'cmd' "${body}"; then fail "tasks list must not contain \"cmd\"" "$(grep -o '.\{40\}cmd.\{40\}' "${body}" | head -n 1)"; else pass "tasks list contains no \"cmd\""; fi
task1="$(jq -r 'sort_by(.order) | .[0].id' "${body}")"
task2="$(jq -r 'sort_by(.order) | .[1].id' "${body}")"
expect_jq "task 1 unlocked, task 2 locked" 'sort_by(.order) | (.[0].unlocked == true and .[1].unlocked == false)'

expect_status "POST /api/sessions (locked ${task2})" 409 "$(req "${student_jar}" POST /api/sessions "{\"taskId\":\"${task2}\"}")"

# --- lab session ----------------------------------------------------------------
status="$(req "${student_jar}" POST /api/sessions "{\"taskId\":\"${task1}\"}")"
expect_status "POST /api/sessions (${task1})" 201 "${status}"
session_id="$(jq -r '.sessionId // empty' "${body}" 2>/dev/null || true)"
if [[ -z "${session_id}" ]]; then
  fail "session id in response"
else
  pass "session ${session_id} created"

  expect_status "check before solving" 200 "$(req "${student_jar}" POST "/api/check/${session_id}")"
  expect_jq "check before solving is not passed" '.passed == false'

  container="$(docker ps -q --filter "label=devopslab.session=${session_id}")"
  if [[ -z "${container}" ]]; then
    fail "lab container labelled devopslab.session=${session_id} is running"
  else
    pass "lab container ${container} is running"
    if docker exec -i "${container}" bash -s <"${SOLUTION}" >"${work}/solution.log" 2>&1; then
      pass "reference solution applied ($(basename "${SOLUTION}"))"
    else
      fail "reference solution failed" "$(tail -n 5 "${work}/solution.log")"
    fi
  fi

  expect_status "check after solving" 200 "$(req "${student_jar}" POST "/api/check/${session_id}")"
  expect_jq "check after solving is passed" '.passed == true and .score == .total'
fi

expect_status "GET /api/progress" 200 "$(req "${student_jar}" GET /api/progress)"
expect_jq "progress shows 1 completed task" '.completed == 1'

expect_status "GET /api/leaderboard" 200 "$(req "${student_jar}" GET /api/leaderboard)"
expect_jq "leaderboard includes ${student}" "any(.entries[]; .username == \"${student}\") or .me.username == \"${student}\""

# --- guards ---------------------------------------------------------------------
expect_status "foreign-origin POST rejected" 403 \
  "$(req "${work}/anon.jar" POST /api/auth/login '{"username":"x","password":"y"}' "https://evil.example")"

# --- admin ----------------------------------------------------------------------
admin_body="$(jq -nc --arg u "${ADMIN_USERNAME}" --arg p "${ADMIN_PASSWORD}" '{username: $u, password: $p}')"
expect_status "admin login" 200 "$(req "${admin_jar}" POST /api/auth/login "${admin_body}")"
expect_status "admin GET /api/admin/stats" 200 "$(req "${admin_jar}" GET /api/admin/stats)"
expect_status "student GET /api/admin/stats" 403 "$(req "${student_jar}" GET /api/admin/stats)"
expect_status "admin logout" 204 "$(req "${admin_jar}" POST /api/auth/logout)"

# --- teardown -------------------------------------------------------------------
if [[ -n "${session_id}" ]]; then
  expect_status "DELETE /api/sessions/${session_id}" 204 "$(req "${student_jar}" DELETE "/api/sessions/${session_id}")"
  session_id=""
fi
expect_status "student logout" 204 "$(req "${student_jar}" POST /api/auth/logout)"
expect_status "GET /api/me after logout" 401 "$(req "${student_jar}" GET /api/me)"

echo
echo "Summary: ${passes} passed, ${failures} failed"
[[ "${failures}" -eq 0 ]]
