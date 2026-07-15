# Ports

Define stable boundaries used by application logic to access external capabilities.

- Keep interfaces narrow and expressed with domain types.
- Do not import concrete infrastructure or presentation packages.
- Prefer consumer-driven contracts over mirrors of implementation APIs.
- Contract changes require tests for all adapters and consumers.

Verify with: `go test ./...`
