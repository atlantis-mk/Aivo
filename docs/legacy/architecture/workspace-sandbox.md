# Workspace And Sandbox Architecture

## Workspace Manager

Workspace Manager owns local project boundaries:

- workspace root normalization
- relative path only APIs
- safe path join
- symlink escape prevention
- ignored directories such as `.git`, `node_modules`, `vendor`, `dist`, `build`, `.next`, and `target`
- binary file detection
- sensitive file detection such as `.env`, `.ssh`, `.pem`, `.key`, and private key names
- max file size and output size
- external directory access policy

Phase 1 implements these checks inside read-only tools. Phase 2 should extract them into a dedicated workspace service used by all file tools.

## Sandbox Runner

Sandbox Runner is the execution backend abstraction for future side effects:

- local backend for trusted low-risk commands
- Docker backend for isolated command execution
- SSH backend for remote workspaces
- future cloud backend for long-running or delegated jobs

Each backend must support command timeout, context cancellation, process cleanup, cwd enforcement, env allowlist, network policy, stdout/stderr limits, and audit metadata.

## Policies

Commands should run with a default timeout and hard maximum. Environment variables are deny-by-default except a small allowlist required for build tools. Network should be disabled or explicitly scoped when possible. Processes must be killed on cancellation, timeout, app shutdown, or session cancellation.

Phase 1 intentionally does not implement bash or write tools. Sandbox becomes mandatory before broad shell access.
