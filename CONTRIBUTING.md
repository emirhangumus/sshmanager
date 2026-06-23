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

This tool stores and uses SSH credentials. See the **Security invariants**
section in `CLAUDE.md` before touching `internal/crypto`, `internal/storage`,
`internal/store`, or anything that builds `ssh`/`sshpass` invocations in
`internal/cli/commands/connect.go`. If you're changing how secrets are
stored, transmitted to subprocesses, or validated, call that out explicitly
in your PR description.
