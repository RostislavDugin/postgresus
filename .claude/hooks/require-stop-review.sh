#!/usr/bin/env bash
# Stop hook: blocks once per distinct working-tree state until the review round named by $1 has run.
#
# usage: require-stop-review.sh <marker-prefix> <reason>
#
# Blocking is guarded by a marker file keyed on marker prefix + session id + a hash of the diff.
# Claude Code does not document a `stop_hook_active` field, so the marker is the only thing standing
# between this hook and an infinite stop loop. Any git failure exits 0 — never block on a broken repo.
#
# Several rounds can share this script: each gets its own marker prefix, so they block together on
# the same tree state and clear together, costing one stop round rather than one per round.

set -uo pipefail

REVIEWED_SOURCE_DIRS=(backend agent frontend)

marker_prefix=${1:-}
reason=${2:-}
[ -n "$marker_prefix" ] || exit 0
[ -n "$reason" ] || exit 0

hook_input=$(cat)

project_dir=${CLAUDE_PROJECT_DIR:-$(git rev-parse --show-toplevel 2>/dev/null)} || exit 0
[ -n "$project_dir" ] || exit 0
cd "$project_dir" 2>/dev/null || exit 0

git rev-parse --verify HEAD >/dev/null 2>&1 || exit 0

changed_paths=$(git status --porcelain -- "${REVIEWED_SOURCE_DIRS[@]}" 2>/dev/null) || exit 0
[ -n "$changed_paths" ] || exit 0

# `git diff` omits untracked files, so hash their contents separately. Without this an edit to a
# newly created file would not change the hash, and its review round would be silently skipped.
tree_hash=$(
  {
    git diff HEAD -- "${REVIEWED_SOURCE_DIRS[@]}" 2>/dev/null
    git ls-files --others --exclude-standard -z -- "${REVIEWED_SOURCE_DIRS[@]}" 2>/dev/null |
      xargs -0 -r sha256sum 2>/dev/null
  } | sha256sum | cut -d' ' -f1
) || exit 0
[ -n "$tree_hash" ] || exit 0

session_id=$(printf '%s' "$hook_input" | jq -r '.session_id // "unknown"' 2>/dev/null)
session_id=${session_id//[^a-zA-Z0-9_-]/}
[ -n "$session_id" ] || session_id=unknown

marker_dir="${TMPDIR:-/tmp}/databasus-claude-review"
mkdir -p "$marker_dir" 2>/dev/null || exit 0
marker="$marker_dir/$marker_prefix-$session_id-$tree_hash"

[ -e "$marker" ] && exit 0
: >"$marker" 2>/dev/null || exit 0

jq -n --arg reason "$reason" '{decision: "block", reason: $reason}'
