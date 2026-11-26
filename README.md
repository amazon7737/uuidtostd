# uuidtostd

Go 1.27 adds the [`uuid`](https://pkg.go.dev/uuid) package to the standard library.

`uuidtostd` reports packages that can replace `github.com/google/uuid` with it, and provides a fix.

## Install

```console
go install github.com/amazon7737/uuidtostd/cmd/uuidtostd@latest
```

## Usage

```console
uuidtostd ./...
uuidtostd -fix ./...
```

## What is replaced

| `github.com/google/uuid`   | `uuid`                |
|----------------------------|-----------------------|
| `uuid.UUID`                | `uuid.UUID`           |
| `uuid.New()`               | `uuid.NewV4()`        |
| `uuid.NewString()`         | `uuid.NewV4().String()` |
| `uuid.Must(uuid.NewRandom())` | `uuid.NewV4()`     |
| `uuid.Must(uuid.NewV7())`  | `uuid.NewV7()`        |
| `uuid.Must(uuid.Parse(s))` | `uuid.MustParse(s)`   |
| `uuid.Parse(s)`            | `uuid.Parse(s)`       |
| `uuid.MustParse(s)`        | `uuid.MustParse(s)`   |
| `uuid.Nil`, `uuid.Max`     | `uuid.Nil()`, `uuid.Max()` |
| `String`, `MarshalText`, `UnmarshalText` methods | same methods |

`uuid.New()` becomes `uuid.NewV4()`, not `uuid.New()`: the standard `New` is only "at this time" equivalent to `NewV4`.

`uuid.Parse`, `uuid.MustParse`, and `UnmarshalText` of the standard library are stricter.
Every input accepted by the standard library is accepted by google/uuid with the same value,
but google/uuid also accepts inputs like `(f81d4fae-7dec-11d0-a765-00a0c91e6bf6)` or `URN:UUID:...`.
The report says so when the package uses them.

## When a package is skipped

The standard `uuid.UUID` and `github.com/google/uuid.UUID` are different types,
so the package is the unit of migration: a package is reported only when every use of google/uuid in it can be replaced.

A package is skipped when:

- it uses an API without standard equivalent: UUID versions 1, 2, 3, 5, 6, `NewSHA1`, `NewMD5`, `NewHash`, `NameSpace*`, `NullUUID`, `SetRand`, the rand pool, `Version()`, `Variant()`, `Time()`, `URN()`, `MarshalBinary`, ...
- it uses `uuid.NewRandom()` or `uuid.NewV7()` outside `uuid.Must()` (the standard functions don't return an error).
- its exported API mentions a google/uuid type (except for `main` packages): changing it would break its importers.
- it imports `encoding/gob`: gob encodes google/uuid with `MarshalBinary`, and the standard `UUID` does not implement it.
- it takes the address of `uuid.Nil` or `uuid.Max`, or assigns them.
- the file Go version is lower than 1.27.

## Known limitations

- The fixed import stays in its original import group, run `goimports` or `gci` after the fix.
- Database drivers with their own type handling (pgx, ...) were not tested. `database/sql` works.
