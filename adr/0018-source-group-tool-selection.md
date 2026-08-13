# ADR-0018: Select MCP and extension capabilities by source group

- Status: Accepted
- Date: 2026-08-11
- Related Work: `CHG-2026-034-source-group-tool-selection`
- Supersedes decisions in: `ADR-0016`
- Closes OPEN: none

## Context

MCP servers and extensions intentionally package related concrete tools behind one user-recognizable capability source. Sending every member summary to the auxiliary model increases selection context and asks it to reproduce Host-owned membership. Selecting one concrete member also fails to express the user's intent to make that source's complete capability available. The Host already owns source readiness, registration identities, per-tool visibility, Provider serialization, and immutable Tool Snapshots.

## Decision

- Each enabled ready MCP or executable extension MUST contribute at most one stable globally unique auxiliary group name and one bounded optional functional description; a missing description MUST remain an empty value rather than suppressing the group or synthesizing content.
- The auxiliary tool-selection request MUST contain only the user intent and single-line `name：description` group entries. Descriptions MUST be treated as untrusted data and MUST NOT carry authority or instructions.
- The auxiliary response MUST be a bare strict JSON array of unique exact candidate group names and MUST contain no object wrapper, reason, Markdown, or unknown value.
- The Host MUST validate each selected group and expand it to all currently registered, mode-eligible, globally visible concrete tools from that source. A globally hidden or otherwise ineligible member MUST remain excluded.
- The Host MUST persist and snapshot the expanded concrete canonical tool names and registration identities. Later source catalog changes MUST NOT silently change an existing conversation's automatic set.
- Provider adapters MAY serialize the selected concrete set as a namespace when supported, but a namespace MUST contain only members of selected groups that survived Host validation.
- Source-group selection MUST NOT install, trust, enable, authenticate, authorize, or execute a source. Concrete call-time permissions and stale-registration refusal remain authoritative.
- MCP creation MAY capture a bounded functional description. An omitted description MUST persist as empty, remain editable, and still allow the source to participate in automatic selection.

## Rationale

The auxiliary model is best used to choose the capability source named by the user's request, while the local Registry is the only reliable owner of current group membership. A minimal line catalog and strict name array reduce tokens and parsing ambiguity. Persisting the expanded concrete set preserves the stable conversation surface and immutable snapshot guarantees established by ADR-0016.

## Consequences

One selected source may expose several concrete schemas to the primary model, so Host-owned member-count and schema-size bounds remain necessary. Source descriptions remain optional bounded hints and blank descriptions reduce semantic routing quality without changing eligibility. Auxiliary selection becomes source-granular, while global visibility, manual activation, execution history, and permission enforcement remain concrete-tool-granular.

## Rejected alternatives

- Select every concrete tool independently: repeats Host-owned source membership in the prompt and does not satisfy source-level user intent.
- Let the auxiliary model return concrete membership for a selected source: permits stale or fabricated expansion and duplicates Registry authority.
- Persist only the selected group ID and expand on every request: silently changes an existing conversation when a source catalog changes.
- Inject every source regardless of intent: recreates the unbounded default tool surface.

## Verification

`AT-TOOL-001` verifies the minimal request and strict array parser. `AT-EXTENSION-001` verifies one candidate per MCP/extension, complete eligible expansion, functional descriptions, and Provider namespace membership. `AT-SESSION-001` verifies stable expanded automatic state and Tool Snapshots. `CT-SECURITY-001` verifies untrusted description handling, unknown-group refusal, hidden-member exclusion, and unchanged call-time authorization.
