## Product Positioning

Aivo is a local-first desktop AI productivity assistant for independent developers and creators, helping them complete real deliverables across code, data, documents, and video.

## Target Users

Primary users are independent developers, independent creators, and technical founders who need one assistant to carry context across project work and produce reviewable outputs.

Secondary users may include small teams or collaborators who benefit from shared outputs, but the first version is optimized for a single power user.

Explicit non-target users are enterprise teams that need complex organization permissions, audit compliance, large-scale multi-user administration, or centralized IT governance from the first release.

## Core Problem

Independent developers and creators often move between coding tools, browsers, data tools, document editors, and media tools while repeatedly re-explaining project context. This fragments the work chain and makes it harder to turn an idea or task into a complete, validated deliverable.

## Product Vision

Aivo should become a desktop AI workbench that understands local projects, coordinates tool use, and helps users ship concrete outputs. Coding is the first end-to-end workflow, while data analysis, document generation, and video creation expand the same deliverable-oriented model.

## Reference Products / Benchmarks

- [anomalyco/opencode](https://github.com/anomalyco/opencode): Reference for open-source coding agent workflows, developer entry points, and coverage across terminal, desktop, IDE, and GitHub integrations. Aivo should learn from its coding-agent ergonomics, but should not be limited to coding.
- [NousResearch/hermes-agent](https://github.com/NousResearch/hermes-agent): Reference for agent ergonomics around skill capture, cross-session continuity, and flexible desktop or remote execution. Aivo should treat these as directional product ideas, not first-version implementation commitments.
- Differentiation: Aivo is a local-first desktop productivity workbench focused on multi-artifact deliverables and user-controlled execution, rather than a pure terminal coding agent or a cloud-first always-running agent.

## Core Workflows

1. Code to delivery: understand a repository, plan changes, edit code, run verification, and generate delivery notes or documentation.
2. Research to report: collect data, analyze findings, create charts or summaries, and produce a structured document.
3. Content to video: draft scripts, gather or create assets, plan edits, and produce video-ready materials.
4. Local context management: index or reference files, repositories, notes, and prior outputs while keeping user control over sensitive data.
5. Review and export: make AI outputs inspectable, reproducible, and ready to hand off.

## Product Principles

- Local-first control: prefer desktop workflows and local project access, with cloud models or services used as optional capability providers.
- End-to-end deliverables: optimize for completed work products, not isolated chat answers.
- User-confirmed sensitive actions: require explicit confirmation before using credentials, publishing externally, deleting or overwriting important files, making purchases, or sending data outside the local environment.
- Reviewable autonomy: allow the agent to work autonomously within ordinary task boundaries while preserving clear plans, diffs, logs, and outputs for review.
- Skill and context accumulation: preserve reusable project knowledge and task patterns over time without promising first-version self-evolution.

## Roadmap / Planned Capabilities

### Now

- Desktop App foundation.
- First-run welcome and provider initialization.
- Session Runtime for durable, resumable coding work.
- Code-to-delivery workflow.
- Desktop code-development replacement gap closure against opencode for repository understanding, editing, verification, session recovery, permissions, plugin/MCP, and review surfaces.
- Local file and repository context.
- Autonomous task execution for ordinary work.
- Confirmation gates for sensitive actions.

### Next

- Data collection and analysis.
- Document generation and export.
- Reusable task templates and workflow patterns.
- Capability gap analysis against hermes-agent.

### Later

- Video creation pipeline.
- Multimodal asset management.
- Optional remote execution.
- Optional collaboration and sharing.
- Plugin or skill ecosystem.

## Experience Direction

Aivo should feel like a focused desktop workbench for serious project execution. The interface should support long-running tasks, file and artifact review, diffs, logs, comparisons, and iterative refinement. It should avoid a marketing-style landing page as the primary experience and should not treat chat as the whole product.

## Global Non-Goals

- Do not make enterprise permissions, audit compliance, or organization administration a first-version requirement.
- Do not build a general-purpose social community or marketplace as an early product pillar.
- Do not make pure chat the core experience when a structured workflow or artifact surface is more useful.
- Do not allow high-risk external actions or destructive local actions without explicit user confirmation.
- Do not treat CLI, TUI, SDK, GitHub Action, enterprise collaboration, or non-code opencode parity as part of the first desktop code-development replacement target.
- Do not commit the first version to reproducing every capability of opencode or hermes-agent.

## Product Success Criteria

- Users can complete real code-to-delivery tasks inside Aivo, including code changes, validation, and delivery documentation.
- Users can resume, search, summarize, checkpoint, and fork prior assistant sessions without losing the underlying event history.
- Users can review what the assistant did through plans, diffs, logs, and generated artifacts.
- The product reduces context switching across coding, data, document, and media work over time.
- The assistant preserves user control over sensitive data and high-risk actions.
- Future OpenSpec changes can clearly reference this product baseline when making scope and UX decisions.

## Assumptions / Needs Confirmation

- The working product name is Aivo.
- "Desktop App" means a local-first desktop application, not a pure Web SaaS or only a CLI/IDE plugin.
- Benchmark products are references for product direction and capability comparison, not mandatory feature parity for the first version.
- The first version prioritizes code to delivery; data, documents, and video remain part of the product vision and roadmap.
