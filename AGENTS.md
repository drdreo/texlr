# Agent instructions

- Keep the CLI dependency-free unless a new dependency materially simplifies correctness.
- Preserve the command contract documented in `README.md`.
- Run `gofmt`, `go test ./...`, and `nix flake check` before submitting changes.
- Treat `internal/texlr/templates/texlr.cls` as CC BY 4.0 adapted material; retain its attribution and modification notice.
- Do not add HagenbergThesis/FH Upper Austria logos, backgrounds, example photographs, or institutional branding.
- New Go code is MIT licensed.
- Build failures must remain actionable in both human and JSON output.
