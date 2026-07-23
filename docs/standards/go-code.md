# Go Coding Standard

These rules apply to Go code in this repository. Feature behavior belongs in
its specification or design document.

## Packages and Files

- Give each package one coherent purpose visible to callers.
- Use short, lowercase package names without underscores or mixed capitals.
- Avoid catch-all packages such as `common`, `helpers`, `misc`, and `util`.
- Keep helpers beside the behavior that owns them.
- Split files to improve navigation. Files do not define architectural
  boundaries.
- Do not create one package per type or one file per function.
- Use package comments to explain package contracts. Use file comments only for
  whole-file constraints.
- Do not use dot imports. Alias imports only for collisions or misleading
  package names.

## Names

- Use Go initialisms: `ID`, `URL`, `JSON`, `PNG`, and `CLI`.
- Let the package name supply context: `render.Image`, not
  `render.RenderImage`.
- Name getters after their value: `Width()`, not `GetWidth()`.
- Use short names in short, obvious scopes. Use descriptive names at package
  boundaries.
- Keep receiver names short and consistent across a type. Do not use `this` or
  `self`.
- Name interfaces by behavior when possible: `Reader`, `Stringer`, or
  `Flusher`.

## APIs and Types

- Export only what callers need. Exported APIs are compatibility commitments.
- Accept interfaces where behavior is consumed. Prefer concrete return types;
  return an interface when that interface is the intended contract.
- Define small interfaces in the consuming package.
- Introduce an interface for an actual consumer need. Do not add one only for a
  possible implementation or mock.
- Prefer useful zero values.
- Use constructors when creation must enforce an invariant or provide a
  required dependency.
- Add options, builders, or other configuration abstractions only after actual
  call sites show the needed variation.
- Use pointer receivers for mutation, large values, or values unsafe to copy.
  Keep receiver choice consistent for a type.

## Control Flow

- Keep the normal path left-aligned with early returns.
- Use `switch` when it states a closed set of cases clearly.
- Use naked returns only in short functions where they improve clarity.
- Reserve `panic` for violated internal invariants. Return errors for invalid
  input, unavailable resources, and other expected failures.
- Avoid mutable package globals. Constants and immutable lookup data are fine.

## Data Ownership

- Define ownership of slices, maps, pointers, and byte buffers at API
  boundaries.
- Copy mutable caller data when retaining it would create surprising shared
  state.
- Treat `nil` and empty slices as equivalent unless an API or serialization
  contract distinguishes them.
- Check sizes, counts, arithmetic, and conversions before allocation.
- Do not depend on map iteration order. Sort keys when deterministic order is
  part of the contract.

## Errors

- Add context at the layer that knows what operation failed.
- Do not log and return the same error.
- Wrap with `%w` when callers may inspect the cause with `errors.Is` or
  `errors.As`.
- Start error strings with lowercase. Do not end them with punctuation.
- Use sentinel or typed errors only when callers need to branch on them.
- Keep machine-readable error data structured until the presentation boundary.

## Context and Concurrency

- Pass `context.Context` as the first parameter to cancellable or
  request-scoped work.
- Do not store contexts in structs or pass `nil`.
- Use context values only for request-scoped data crossing API boundaries, not
  ordinary function parameters.
- Cancellation has one explicit owner. A function deriving a cancellable
  context calls its cancel function or transfers it to the caller.
- Every goroutine needs an owner, a completion path, and defined error handling.
- Do not add concurrency without independent work and a clear lifetime.

## Resources and Side Effects

- The function that acquires a resource owns closing it unless ownership is
  explicitly transferred.
- Check close and flush errors when they can affect output.
- Make filesystem, environment, network, clock, and process access visible at
  package boundaries.
- Avoid hidden work in `init`. Use it only when package initialization cannot be
  expressed safely through declarations or explicit setup.

## Comments and Documentation

- Comments explain contracts, constraints, or reasons. Do not narrate code.
- Add doc comments to top-level exported declarations. Explain their contract,
  and follow normal Go declaration-name wording.
- Put externally observable behavior in specifications or API documentation,
  not only implementation comments.
- A temporary-work comment needs an issue or concrete removal condition. Avoid
  ownerless `TODO` comments.

## Tests

- Use table-driven tests when cases share setup and assertions.
- Use subtests when their names improve failure location.
- Keep tests independent of map order, wall-clock timing, network services,
  machine paths, and ambient environment variables.
- Use golden files when whole-output review matters. Updates must be explicit.
- Test contracts and edge cases. Do not restate implementation line by line.
