# AGENTS.md

Notes for coding agents working in this repository. [README.md](README.md) covers what it does and how to use it.

## Versioning

Plain SemVer: `vX.Y.Z` tags, each a GitHub Release with the `ahv` binaries
for linux and darwin on amd64 and arm64 (built by `release.yml`), per the
[versioning standard](https://github.com/mbologna/.github/blob/main/VERSIONING.md).

- Never create, move or delete tags, and never edit a version number to cut
  a release: auto-release tags from the commit types (`feat` minor; `fix`,
  `perf`, `revert` patch; `!` or a `BREAKING CHANGE:` footer major).
- Write Conventional Commit subjects; they decide the next version.
