---
name: ship-release
description: Ship a tagged Forgente release (vX.Y.Z-N) end to end — cut the forgente/vX.Y branch, rc dry run, annotated release tag, and verify the full fan-out (GitHub release, docker tags, dl binaries, version.json, brew, helm). Use when shipping or verifying a release.
---

# Shipping a tagged Forgente release

Authoritative background: FORGENTE.md "Shipping a tagged release". This skill adds
the exact commands and the verification checklist. Versions are
`v<upstream>-<forgente release>` (e.g. `v1.26.4-1`); plain `vX.Y.Z` names are
mirrored upstream tags — never reuse them.

## Preconditions

- Decide the upstream tag to ship. Note forgente.com runs main-nightly and can
  only pin to a release whose DB migration level ≥ the instance's current
  level; check before promising an instance pin.
- Repo secrets are already configured (see FORGENTE.md); no secret work needed
  unless a workflow adds a new consumer. IAM user `forgente-release-ci` is at
  its 2-access-key max — new consumers need their own scoped IAM user.

## Procedure

1. **Cut the release branch** `forgente/vX.Y` from the upstream tag:
   `git branch forgente/vX.Y vX.Y.Z && git push origin forgente/vX.Y`.
   NOT `release/v*` — those are pure upstream mirrors owned by the sync routine.
2. **Cherry-pick the rebrand commits** onto it. The conflict-file list at the end
   of FORGENTE.md "Rebranding state" is the checklist. Also cherry-pick the docs
   PRs (README etc.) so the tag/tarball carries Forgente's README (missed in
   v1.26.4-1). On 1.26-era branches expect structural differences: Dockerfiles
   use `/go/src/code.gitea.io/gitea/<binary>`, no cosign steps, unpinned action
   refs — keep the release line's structure, apply substitutions only.
3. **rc dry run**: push annotated tag `vX.Y.Z-N-rc1` on the release branch.
   `release-tag-rc` builds binaries, rc images, and a *draft* GitHub release.
   Verify (see checklist below scaled to rc), then delete the draft release.
4. **Release tag**: push an **annotated** tag `vX.Y.Z-N` whose tag message IS the
   release notes ("based on Gitea X.Y.Z" + upstream changelog link + Forgente
   delta) — `release-tag-version` runs `gh release create --notes-from-tag`.
5. **Fan out** (each is its own repo/PR):
   - `forgente/deployment`: bump `version.json` (update checker). CloudFront
     caches it — invalidate after any out-of-band upload.
   - `homebrew-forgente`: versioned formula bump.
   - `helm-forgente`: chart appVersion pin; its workflow must stage the packaged
     chart OUTSIDE `charts/` (helm dependency build vendors subcharts there).
   - Snap: promote to stable once the store listing is public.
6. **Housekeeping**: pushing near release branches can queue upstream-workflow
   runs targeting private runners — they hang forever; cancel them
   (`gh run list --status queued`, then `gh run cancel`).

## Verification checklist (validate against REAL external state, not workflow logs)

- GitHub release exists with the full asset matrix and `.asc` signatures.
- Binaries: download one from https://dl.forgente.com and `gpg --verify` against
  release key `67129BAD57A2C8D2186032489D6FD2FD6E0B9BA5`.
- Docker: `docker manifest inspect forgente/forgente:<tag>` for each of
  `latest`, `<major>`, `<major.minor>`, `<upstream version>`, `<full version>`
  and the `-rootless` variants, on both Docker Hub and ghcr.io.
- Update checker: `curl https://dl.forgente.com/forgente/version.json` shows the
  new version.
- Helm: `curl https://dl.forgente.com/charts/index.yaml` lists the new chart.
- Brew: formula installs (or at least `brew audit` passes) with the new URL/sha.
- Known issue until fixed: container `forgente --version` reports the version
  via `git describe` as `X.Y.Z+N` instead of `X.Y.Z-N`.
