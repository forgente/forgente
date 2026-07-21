#!/usr/bin/env bash
# Cherry-pick upstream Gitea commits onto Forgente (post-hard-fork sync model).
#
# Since the Phase 2 cutover Forgente no longer merges upstream wholesale;
# security and bug fixes are cherry-picked individually. Upstream commits
# still use the gitea.dev module path internally, so any pick touching
# import lines conflicts mechanically — this script rewrites the module
# path in conflicted files and leaves real conflicts for manual resolution.
#
# Usage: contrib/forgente/pick-upstream.sh <upstream-sha> [more shas...]

set -euo pipefail

UPSTREAM=${UPSTREAM:-upstream}
if ! git remote get-url "$UPSTREAM" >/dev/null 2>&1; then
	echo "error: remote '$UPSTREAM' not configured." >&2
	echo "  git remote add $UPSTREAM https://github.com/go-gitea/gitea.git" >&2
	exit 1
fi
[ $# -ge 1 ] || { echo "usage: $0 <upstream-sha> [sha...]" >&2; exit 1; }

if [ "$(git rev-parse --abbrev-ref HEAD)" = "main" ]; then
	echo "error: do not cherry-pick onto main directly; create a branch and open a PR." >&2
	exit 1
fi

git fetch -q "$UPSTREAM"

# Rewrite the upstream module path, excluding the external modules that
# legitimately keep the gitea.dev prefix (sdk, actions-proto-go).
rewrite_module_path() {
	xargs -r perl -pi -e 's{gitea\.dev/(?!sdk\b|actions-proto-go\b)}{forgente.com/}g'
}

for sha in "$@"; do
	echo "==> cherry-picking $sha"
	if git cherry-pick "$sha"; then
		# clean pick may still carry gitea.dev imports in touched files
		git show --name-only --format= HEAD | rewrite_module_path
		if ! git diff --quiet; then
			git add -A && git commit -q --no-edit --amend
			echo "==> rewrote module paths in cleanly-picked files"
		fi
	else
		echo "==> conflicts; rewriting module paths in conflicted files" >&2
		git diff --name-only --diff-filter=U | rewrite_module_path
		echo "==> resolve remaining conflicts, then: git add -A && git cherry-pick --continue" >&2
		echo "==> afterwards run: go build ./... && make lint-go" >&2
		exit 1
	fi
done

echo "Done. Run 'go build ./...' and open a PR as usual."
