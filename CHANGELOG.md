# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

This project has not been tagged yet, so everything below is unreleased. Entries
predating the commands being feature-complete are omitted as initial development.

### Changed

- **Breaking: `do-cleanup` now clears artifact deploy locations as well as asset
  ones.** It previously read only the `assets` section of `artifacts.json`, while
  `do-deploys` extracts into the `deploy_location` of the `artifacts` section.
  Nothing ever cleared those directories, so a file present in an older archive
  but absent from the current one survived into the next deployment.

  If you run `slarty do-cleanup` in a pipeline today, it will now empty artifact
  deploy directories that it previously left untouched. **Pass `--assets-only` to
  keep the old behavior.** ([#39](https://github.com/dstockto/slarty/issues/39))

- **Breaking: `do-cleanup` refuses any `deploy_location` outside the project
  root.** Cleaning is confined to directories strictly inside `root_directory`.
  A location that resolves to the root, to a directory containing the root, or
  outside it via `..` is reported and skipped rather than cleaned. Symlinks are
  resolved before this check. `deploy_location` has always been relative to
  `root_directory`, and an absolute value is folded back underneath it, so
  deploying outside the project root is not a supported configuration.

- `validate` now reports a `deploy_location` that escapes the project root with
  `..` as an error, because `do-cleanup` refuses such paths and files there would
  accumulate unnoticed.

### Added

- `do-cleanup` accepts `--artifacts-only` and `--assets-only` to restrict cleanup
  to a single section of `artifacts.json`. The two are mutually exclusive.
  `--filter` and `--exclude` match names in both sections, so an artifact and an
  asset sharing a name are selected together. A `deploy_location` shared by more
  than one entry is cleaned once.
- `validate` command, which inspects `artifacts.json` for duplicate names,
  missing directories, empty or out-of-root deploy locations, and repository
  adapter problems, exiting non-zero when any errors are found.
- `--json` output for the `artifact-names`, `hash-application`, and
  `should-build` commands, so pipelines no longer have to scrape ASCII tables.
- `--fail-fast` for `do-builds`, which stops after the first failed build. The
  run summary now also names the artifacts that failed rather than only counting
  them.

### Fixed

- `do-cleanup` no longer wipes the project root when a `deploy_location` is empty
  or `"."`.
- Artifact tables are printed in configuration order rather than a
  non-deterministic one.
- Errors from `git` now include its stderr, instead of being reported without the
  underlying reason.
- Documentation and command output use `tar.gz` terminology consistently; the
  archives have never been zip files.
- The S3 adapter no longer forces the `default` AWS profile, allowing instance
  credentials to be used.

### Security

- Archive entries that would escape the destination directory during extraction
  are rejected, closing a path traversal (Zip Slip) in `do-deploys` and
  `deploy-assets`.
- Extracted file modes are masked to `0o777`, so an archive cannot set setuid,
  setgid, or sticky bits on deployed files.
- Per-entry decompression is bounded by a generous, tunable cap, so a
  decompression bomb cannot fill the disk.
- S3 operations run under a timeout rather than hanging indefinitely.
- The shell-execution trust boundary in `do-builds` is documented: the `command`
  field in `artifacts.json` runs on the build server, so anyone who can modify
  that file can run commands there.

[Unreleased]: https://github.com/dstockto/slarty/commits/main
