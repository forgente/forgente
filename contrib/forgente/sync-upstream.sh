#!/usr/bin/env bash
# Sync Forgente with upstream Gitea.
# Fetches the upstream remote, merges upstream/main into the local main branch,
# and fast-forwards local release/v* branches and tags. Push is left to the
# caller so merge results can be reviewed first.
set -euo pipefail

UPSTREAM=${UPSTREAM:-upstream}
BRANCH=${BRANCH:-main}

if ! git remote get-url "$UPSTREAM" >/dev/null 2>&1; then
  echo "error: remote '$UPSTREAM' not configured. Add it with:" >&2
  echo "  git remote add $UPSTREAM https://github.com/go-gitea/gitea.git" >&2
  exit 1
fi

if [ -n "$(git status --porcelain)" ]; then
  echo "error: working tree not clean, commit or stash first" >&2
  exit 1
fi

echo "==> Fetching $UPSTREAM (branches + tags)"
git fetch "$UPSTREAM" --tags --prune

echo "==> Merging $UPSTREAM/$BRANCH into $BRANCH"
git checkout "$BRANCH"
if git merge --no-edit "$UPSTREAM/$BRANCH"; then
  echo "==> Merge OK"
else
  echo "==> Merge has conflicts; resolve them, then: git merge --continue" >&2
  exit 1
fi

echo "==> Updating release branches from $UPSTREAM"
for ref in $(git for-each-ref --format='%(refname:short)' "refs/remotes/$UPSTREAM/release/*"); do
  branch=${ref#"$UPSTREAM"/}
  if git show-ref --verify --quiet "refs/heads/$branch"; then
    git fetch . "$ref:$branch" 2>/dev/null || echo "  skipped $branch (not fast-forward)"
  else
    git branch --track "$branch" "$ref" >/dev/null
    echo "  created $branch"
  fi
done

echo
echo "Done. Review the result, then push:"
echo "  git push origin $BRANCH --tags"
echo "  git push origin 'refs/heads/release/*:refs/heads/release/*'"
