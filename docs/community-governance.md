# Governance

Forgente is currently a single-maintainer project. This page states how it is
run today, plainly — it will grow when the community does.

## Decisions

The maintainer decides scope, direction, and releases, guided by
[ROADMAP.md](../ROADMAP.md). Technical conventions contributors are expected
to follow live in [CONTRIBUTING.md](../CONTRIBUTING.md) and
[AGENTS.md](../AGENTS.md); operational procedure lives in
[FORGENTE.md](../FORGENTE.md). There is no committee, no voting process, and
no compensation program — descriptions of such structures found in upstream
Gitea's governance documents do not apply to Forgente.

## Contributing and escalation

- Bugs and feature requests: GitHub issues on
  [forgente/forgente](https://github.com/forgente/forgente/issues).
- Code: pull requests per [CONTRIBUTING.md](../CONTRIBUTING.md). All changes
  land through reviewed PRs — nothing is committed directly to `main`.
- Security: privately to `security@forgente.com` per
  [SECURITY.md](../SECURITY.md) — never a public issue.

## Relationship to Gitea

Forgente hard-forked from [Gitea](https://github.com/go-gitea/gitea) in July
2026 and is developed independently. Upstream security advisories are watched
daily and relevant fixes are cherry-picked (see FORGENTE.md). Forgente
maintains API compatibility with Gitea's ecosystem tooling on purpose.
Gitea's name and logo are trademarks of their respective owners; Forgente
uses its own name and mark.
