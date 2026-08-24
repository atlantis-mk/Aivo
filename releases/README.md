# Releases

Create one record per formal stable version from `_template.md`, for example `v0.1.0.md`.

A Release stores only what an agent needs to know about actual delivery, compatibility, and evidence. It may reference schema-v2 Work in `Done` state or legacy Work listed in `../changes/archive.json`. A Release does not rewrite Requirements or historical Work; the same-name Git tag freezes the full snapshot.
