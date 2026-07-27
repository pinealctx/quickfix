# Pinealctx fork maintenance

This repository is a deliberately small, independently maintained fork of
[`quickfixgo/quickfix`](https://github.com/quickfixgo/quickfix). It is published
as `github.com/pinealctx/quickfix`, so consumers import the fork directly without
a Go module `replace` directive. Go package names remain `quickfix`.

## Maintained differences

- The public Go module path is `github.com/pinealctx/quickfix` while package
  names remain `quickfix`.
- `settings`: a strict, typed JSON configuration package that converts to the
  engine's native `quickfix.Settings` type.
- `ValidateFieldsHaveValues=N` is honored during data-dictionary validation.

Fork-specific changes should remain focused, documented, and covered by tests.
Business-specific generators, logging adapters, credentials, and application
wrappers belong in their consuming projects.

## Synchronizing upstream

The upstream remote is named `github.com` in the maintainer checkout. Fetch it
and merge its main branch into this fork, resolving conflicts in favor of the
tested fork behavior only where a maintained difference is involved:

```text
git fetch github.com main
git switch main
git merge github.com/main
go test ./...
```

After merging upstream, retain the `github.com/pinealctx/quickfix` module and
import paths throughout the source tree and generator templates.

Do not copy or reintroduce application-specific code into the engine solely to
avoid maintaining it in the application that owns the behavior.
