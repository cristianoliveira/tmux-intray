# Release Notes

## Unreleased

### SQLite gradual opt-in rollout

- SQLite storage is available as a beta opt-in path; default backend remains TSV.
- Added phased rollout guidance for `tsv` -> `dual` -> `sqlite`, including rollback steps.
- Documented safeguards: unknown/failed backend initialization falls back to TSV.
- Added migration and troubleshooting documentation with sqlc workflow references.
- Added a dedicated GitHub issue template for SQLite opt-in feedback.
