# Internal Packages

All packages here are private to this module. Follow nearest nested guidance when present.

- Keep dependency direction inward: presentation/application/infrastructure may use domain; domain must not import outward.
- Prefer narrow consumer-owned interfaces and explicit constructor injection.
- Keep package responsibilities small; do not create generic utility dumping grounds.
- Place tests beside implementation and match local table-driven/fake patterns.
- Run the changed package tests first, then `go test ./...` and dependency guardrails.
