# Releases

Create one record per formal or prerelease version from `_template.md`, for example `v0.1.0.md` or `v0.2.0-rc.1.md`.

A Release stores only what an agent needs to know about actual delivery, compatibility, and evidence. It may reference only Work already listed in `../changes/archive.json`. A Release does not rewrite Requirements or sealed Work; the same-name Git tag freezes the full historical snapshot.
