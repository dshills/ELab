# Service, compilation and analysis ownership

Every user and agent action enters the same `workbench.Service.Dispatch` command
facade. The service mutex remains the single authority for document generation,
command/revision guards, publication, cancellation and shutdown. The root HTTP host,
stdio, sockets and the 229-command discovery surface share that implementation.
No analysis engine, integration budget or electrical limit changes here.

## Application boundaries

The repository root is the Go web host. `static/` is embedded in standalone
executables; development mode serves that directory directly. Browser profiles
have isolated live workspaces and share a local project/recovery library. HTTP
commands route through the same service and revision/generation guards.

`launcher/macos/` supplies a small web-host launcher with start/open/quit controls.
It does not implement a second circuit UI. The old Vellum client and its SDK are
[archived](native-archive.md), while shared Go models, agents and platform
PDF/PNG/audio/print services remain. Headless agent sessions remain independent. An explicitly enabled browser workspace
accepts MCP through the installed host executable and a private, revocable local
descriptor. The bridge pins the host instance and workspace; commands share the
visible service, revision/generation guards, actor history and checkpoint path.
Host filesystem, application lifecycle and printing commands are excluded.

## Compilation boundary

`compilation.go` contains the existing pure `Graph.Compile` topology compiler and
shortcut RC validation. Numerical graph validation and solver laws are unchanged.
A private one-entry `compilationCache` belongs to Service. Its lock-required
entry point keys the complete serialized graph, including models, positions,
nested block metadata and initial conditions. Nonfinite encoding errors bypass
the cache instead of reusing a previous circuit. All source graph fields take
part in identity, even when they do not alter electrical connectivity.

The cache retains private detached data. Every return deep-copies graph/model
pointers, shortcut components, maps, net terminals and labels, preserving nil
versus empty compiler representations. Workers and callers cannot modify the
entry or the service graph through a compiled result. The store retains only
one graph's compilation, so document replacement cannot retain an unbounded
history. Service graph consumers use this boundary; independent verification
initial-condition graphs and sweep trial graphs still call pure Graph.Compile.
The live restart path derives its summary from the same compilation rather than
compiling the graph twice.

`SnapshotIfChanged` still provides the zero-allocation idle path. Active snapshots
can reuse compilation, but still own their presentation and analysis arrays.
Cache microbenchmarks measure only compiler-boundary cost, not visible desktop
latency or overall simulation throughput. Hits still hash and clone the graph;
this trades some copying for reduced topology work without new invalidation
counters spread through editing commands.

## Analysis-job store

`analysisJobs` groups sweep, verification/recording, spectrum and tolerance state,
serials, worker accounting and analysis defaults. Its extracted admission and
sweep-retention methods protect active/kept assets and retire evicted selection
and cursor references together. The store has no mutex, background loop, command
handler or independent revision. The embedding Service holds Service.mu for all
access. Store methods ending Locked require that authority or an isolated owned
store in a test.

Job start, graph capture, cancellation, source/document validation, file
publication and service lifecycle remain on Service. This preserves the existing
publication guards while making retained analysis state and its capacity policy
explicit. The store is private to workbench; it is a focused ownership boundary,
not a package-wide solver rewrite or a second command authority.

## Regression scope

Tests compare fresh and cached compilations across every built-in example,
including nested models, blocks and shortcut paths. Returned references and
source references are independently damaged to check ownership. Capacity,
metadata changes, invalid graphs, signed zero, encoding errors and simultaneous
agent reads/user edits are covered. Separate store controls enforce bounded
admission, active/kept protection, eviction selection cleanup and store isolation.
Existing electrical, capture, persistence, transport, cancellation and HTTP/browser
command tests continue to exercise the unchanged command facade.

## Portable library backups

The web host snapshots the shared library after checkpointing active workspaces
under registry → library → workspace locks. Native project validation is exposed
as a pure workbench function; backup/restore does not create an engine or run a
solve for each entry. The bounded ZIP carries explicit logical records and digests,
never arbitrary filesystem paths or credentials.

Restore stages a complete validated batch and publishes `restores/<new-id>/` by
one rename. Library reads include these batches; ordinary save/open/trash and
recovery controls use the same resolver. Fresh identities preserve existing
projects and current browser workspaces. Restored history is marked retained so
automatic-history rotation cannot delete an explicitly imported backup copy.
