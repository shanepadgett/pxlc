# Go Architecture Standard

Go has no universal application architecture or official standard project
tree. This document records durable conventions supported by Go's language,
tooling, and community guidance. Feature-specific architecture belongs in a
design document when its requirements are known.

## Modules and Repositories

- A module is a set of packages versioned and distributed together. Its root
  contains `go.mod`.
- Prefer one module at the repository root.
- Add another module only when code needs independent versioning, distribution,
  ownership, or dependency management.
- Keep `go.mod` and any generated `go.sum` in version control.
- Start with the simplest directory layout that fits the packages and commands.
- Do not impose a generic `pkg`, `app`, `domain`, `service`, `repository`, or
  `adapters` tree. None is a standard Go layout.

## Packages

- Organize packages around cohesive capabilities visible to their callers.
- One directory contains one package, excluding external test packages.
- Package names are short, lowercase, and describe what the package provides.
- Avoid package trees that merely mirror conceptual layers or data types.
- Avoid catch-all packages such as `common`, `util`, `types`, `interfaces`, and
  `models`.
- Keep implementation details unexported. Put unsupported external APIs under
  `internal` when the import restriction matches the intended boundary.
- Do not create forwarding packages or wrappers without added behavior or a
  real compatibility boundary.

The `internal` import rule is enforced by the Go command.

## Dependency Graph

- Package imports form an acyclic graph. Go rejects direct and indirect import
  cycles.
- Resolve a cycle by reconsidering ownership and package responsibilities.
- Do not add global registries, callback indirection, or empty interfaces only
  to hide a cycle.
- Keep dependencies explicit through function arguments, constructors, and
  concrete fields.
- Define interfaces in the package that consumes the behavior.
- Introduce abstractions after a caller needs substitution or a stable contract.
  Do not design them for hypothetical implementations.
- Prefer the standard library when it adequately solves the problem.
- Evaluate third-party modules for API fit, maintenance, documentation,
  licensing, release stability, transitive cost, and security posture.

## Commands

- Executable packages use `package main` and define `main.main`.
- Give each independently installable command its own package directory.
- `cmd/<name>` is a convention for repositories with multiple commands or a mix
  of commands and importable packages. It is not mandatory.
- Keep process concerns such as arguments, environment, startup, shutdown, and
  exit status at the command boundary.
- Command-only logic may remain in `package main`. Move code into another
  package when reuse, cohesion, or API boundaries justify it. No universal rule
  requires a tiny `main`.

## Public APIs

- Exported packages and identifiers create compatibility obligations.
- Keep application implementation under `internal` unless external consumers
  are deliberately supported.
- Return concrete types when practical. Accept narrow consumer-owned interfaces
  for required behavior.
- Keep transport, storage, command, and serialization types from becoming a
  shared model by accident. Convert at a boundary when their contracts differ.
- Avoid exposing dependency types unless they are intentionally part of the
  public contract.

## State, Lifetime, and Concurrency

- Avoid mutable global state. Construct state explicitly and give it a clear
  owner.
- The creator of a goroutine, worker, or background loop owns its cancellation
  and completion.
- Pass `context.Context` through cancellable call chains. The caller controls
  cancellation.
- Do not make concurrency part of a package API when synchronous behavior is
  sufficient.
- Define shutdown, error propagation, and resource ownership wherever
  background work exists.

## Decisions Outside This Standard

Community guidance does not establish one correct choice for:

- clean, hexagonal, onion, MVC, or other application layering;
- dependency-injection frameworks or generators;
- storage, caching, queues, background workers, or transaction boundaries;
- plugin systems and extension loading;
- network protocols and service boundaries;
- monorepo versus multirepo organization;
- vendoring dependencies;
- one module versus multiple modules when independent lifecycle needs exist; or
- functional options versus option structs.

Do not encode these choices as standing rules before a feature needs them.
Discuss the concrete requirements, choose the smallest suitable design, and
record that decision with the feature. Promote it to a repository standard only
after it becomes a stable cross-cutting convention.
