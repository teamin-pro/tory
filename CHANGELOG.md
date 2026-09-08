# Changelog

Tory is a thin PostgreSQL layer over [pgx](https://github.com/jackc/pgx): SQL lives in `.sql` files
as named queries, Go calls them by name and scans rows into its own types. The
[Readme](https://github.com/botforge-pro/tory#readme) documents the whole API; this file records what
changed between releases.

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and the project follows
[Semantic Versioning](https://semver.org/spec/v2.0.0.html). Entries start at 2.0.0, the first release
of the current API.

## [2.1.0] - 2026-09-08

### Added

- `t.Atomic(ctx, fn)` — a transaction as a method on the connection, next to the query methods that
  already live there.
- `tx.Atomic(ctx, inner)` nests a transaction inside a running one, which PostgreSQL runs as a
  savepoint: a batch rejected by a constraint no longer costs the whole transaction. Both forms take
  the result type from the function they run — `Atomic[R]` with `R` inferred, which is why neither
  call spells a type argument, and a nested block that produces nothing returns `any` and `nil`.

  What a savepoint does and does not hold — a dead connection or a cancelled context is not one of
  the things it holds — and what nesting costs per block is written up under Transactions in the
  [Readme](https://github.com/botforge-pro/tory#readme). Read it before building recovery on this.

  ```go
  report, err := t.Atomic(ctx, func(tx tory.Tx) (*Report, error) {
      _, err := tx.Atomic(ctx, func(tx tory.Tx) (any, error) {
          return nil, tx.Exec(ctx, "import-batch", tory.Args{"rows": batch.Rows})
      })
      ...
  })
  ```

### Changed

- The repository moved, so the import path moved with it. Every file that imports the package
  changes by that one line; calls, types and signatures stay as they are:

  ```go
  import "github.com/teamin-pro/tory/v2"   // was
  import "github.com/botforge-pro/tory/v2" // now
  ```

  Rewrite the imports, then run `go mod tidy`: it picks up the new path from them and drops the old
  requirement. `go get` on its own does not touch import lines.

  As a module, the new path begins at 2.1.0: the `go.mod` inside every earlier tag still declares
  the old one, so `@v2.0.0` under the new path fails with a module path mismatch. Builds pinned to
  the old path keep working — GitHub redirects it after the move, and the proxy already holds its
  tags — but 2.1.0 and everything after it is published under the new path only, so switching the
  import is the way to get them. As a repository, the move carried the tags along, which is why the
  links at the bottom of this file point at the new home for 2.0.0 as well.

  Versions are counted per module, and the host path is not part of that count, which is why moving
  it lands in a minor release with the `/v2` suffix untouched.

### Deprecated

- `tory.Atomic(ctx, t, fn)` in favour of `t.Atomic(ctx, fn)`. The old form still works and goes away
  in the next major release.

## [2.0.0] - 2026-09-07

Every query call in your code changes. Go 1.27 or newer is required: the API is built on methods
with type parameters, which the language gained in that release.

Coming from 0.7.x, the import path gains the major-version suffix as well:

```go
import "github.com/teamin-pro/tory"    // was, up to 0.7.4
import "github.com/teamin-pro/tory/v2" // 2.0.0; see 2.1.0 above for where it lives now
```

### Changed

- Queries became methods and take a `context.Context` first. The connection and the transaction
  carry the same set, except `Query`, which belongs to the transaction alone — as it did before,
  though it changed shape with the rest: `tx.Query(name, args, scan)` is now
  `tx.Query[T](ctx, name, args, scan)`.

  ```go
  users, err := tory.Select[User](t, "list-users", nil) // was
  users, err := t.Select[User](ctx, "list-users", nil)  // now
  ```

- A transaction is no longer bound to one row type. `Tx[T]` became a plain `Tx`, and the type
  parameter moved to the methods that return rows — `Select`, `Get`, `Scalar` and `Query` — so one
  transaction reads as many different types as it needs. `Exec`, `ExecReturning` and `QueryRow` are
  methods on it too, they just have no type parameter to move.

  ```go
  tory.Atomic(t, func(tx tory.Tx[User]) (*User, error) {      // was
      return tx.Get("get-user-by-id", tory.Args{"id": 42})
  })
  tory.Atomic(ctx, t, func(tx tory.Tx) (*User, error) {       // now
      return tx.Get[User](ctx, "get-user-by-id", tory.Args{"id": 42})
  })
  ```

- `ApplyPatches(ctx, t, opts)` and `Atomic(ctx, t, fn)` take a context too, and `Atomic[R, T]`
  became `Atomic[R]`: the row type left with `Tx[T]`, the result type stayed.

### Removed

- The package-level query functions `Select`, `Get`, `Scalar`, `QueryRow`, `Exec` and
  `ExecReturning`. Each is now a method of the same name on the connection and on the transaction,
  so the replacement is mechanical.

[2.1.0]: https://github.com/botforge-pro/tory/compare/v2.0.0...v2.1.0
[2.0.0]: https://github.com/botforge-pro/tory/releases/tag/v2.0.0
