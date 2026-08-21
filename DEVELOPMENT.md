# Development

## Requirements

- [Go](https://go.dev/) — `brew install go`
- [Task](https://taskfile.dev/) — `brew install go-task`
- [staticcheck](https://staticcheck.dev/) — `go install honnef.co/go/tools/cmd/staticcheck@latest`
- [gh](https://cli.github.com/) — `brew install gh` (for releases)

## Commands

- `task artifacts` — Build release artifacts to `.release/`
- `task build` — Build binary
- `task clean` — Remove build artifacts
- `task install` — Install to `$GOPATH/bin/` and fish completion
- `task lint` — Lint (staticcheck, tidy, vet)
- `task release TAG=v0.1.0` — Create GitHub release with artifacts
- `task run` — Run from source
- `task sha` — Generate SHA256 checksums
- `task tag TAG=v0.1.0` — Create and push git tag
- `task test` — Run tests
- `task uninstall` — Remove from `$GOPATH/bin/` and fish completion
- `task updates` — Check for dependency updates
- `task version` — Show current version info from git

## Release

1. Ensure all changes are committed and pushed
2. `task release TAG=v0.1.0`
3. `task sha` to get hashes
4. Update [homebrew-made](https://github.com/oschrenk/homebrew-made) with new version and SHA
5. `brew update && brew upgrade arto`
