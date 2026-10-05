# Contributing

## Dev setup

Requires Go 1.24+ (see `go.mod`). `sshpass` is needed at runtime for
password-auth connections, but not for building/testing.

```sh
git clone https://github.com/emirhangumus/sshmanager
cd sshmanager
make build
```

## Workflow

```sh
make test            # go test -v ./...
make test-coverage    # race + coverage, writes coverage.html
make lint             # golangci-lint run
make fmt              # go fmt ./...
make vet              # go vet ./...
```

Before opening a PR:

- `make vet` and `make lint` must be clean.
- New behavior should come with table-driven tests alongside the existing
  ones in the relevant `_test.go` file.
- CI enforces a minimum total coverage threshold (see `COVERAGE_MIN` in
  `.github/workflows/ci.yml`) — don't drop it.

## Security-sensitive changes

This tool stores and uses SSH credentials. Read the [security policy and threat
model](SECURITY.md), [storage format and recovery](docs/storage-format.md), and
[execution/backup boundaries](docs/security-boundaries.md) before changing
`internal/crypto`, `internal/storage`, `internal/store`, or SSH/SCP command
construction in `internal/cli/commands`. See [architecture](docs/architecture.md)
for package responsibilities. If you're changing how secrets are stored,
transmitted to subprocesses, or validated, call that out explicitly in your PR
description.
