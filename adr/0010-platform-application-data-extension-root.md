# ADR-0010: Store managed extensions in platform application data

- Status: Accepted
- Date: 2026-08-06
- Related Work: `CHG-2026-024-platform-extension-storage`
- Supersedes: ADR-0009 only for selection of the managed root location
- Closes OPEN: none

## Context

ADR-0009 established Host-managed immutable extension generations but left the physical root as an implementation detail. The initial implementation used the database sibling `~/.aivo/extensions`. The user selected a Chrome-style application-data hierarchy to reduce accidental discovery and modification during ordinary file browsing.

## Decision

- The default desktop/Core store MUST use the platform configuration/application-data base plus `Aivo/Default/Extensions`.
- On macOS the resulting root is `~/Library/Application Support/Aivo/Default/Extensions`.
- Explicit non-default database stores MAY retain a database-sibling `extensions` directory to preserve isolation for tests and embedded runtimes.
- Existing exact packages beneath the former default database-sibling root MUST migrate through the same bounded copy, verification, hardening, and atomic publication path used for installation.
- Persistence MUST switch to the new managed path before the old owned directory is removed. A failed copy or save MUST leave the old package recoverable.
- Runtime, Views, tools, updates, and uninstall MUST use the persisted new package and the exact current managed root after migration.
- Directory obscurity MUST NOT be described as protection from the owning OS user; private permissions, read-only modes, integrity verification, and scoped deletion remain authoritative controls.

## Rationale

Platform application-data locations match desktop conventions and Chrome's profile-oriented layout. Keeping `Default` in the hierarchy permits a future profile decision without requiring one now. Reusing the existing package publication mechanism avoids introducing a second trust path.

## Consequences

The database and managed packages no longer share a parent in default installations. Startup gains a one-time filesystem migration. Older development binaries may require package reinstall after downgrade.

## Rejected alternatives

- Rename `~/.aivo/extensions` to another dot-directory: still exposes an application-specific home-folder entry and does not follow platform desktop conventions.
- Treat the deeper path as tamper prevention: the owning user can still find and change it.
- Move packages without revalidation: could persist content different from the trusted generation.
- Move the entire database and credential lifecycle in this Work: expands risk and migration scope without being required for extension storage.

## Verification

`AT-EXTENSION-004` covers default-root selection and installation. `CT-SECURITY-001` covers exact legacy/current containment and deletion scope. `CT-RELIABILITY-001` covers verified copy-before-switch, cleanup order, failure recovery, and isolated test stores.
