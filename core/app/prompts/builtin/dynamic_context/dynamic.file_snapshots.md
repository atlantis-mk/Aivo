---
schema: aivo.prompt/v1
id: dynamic.file_snapshots
category: dynamic_context
title: File Snapshots
enabled: true
---

Use these sha256 values as expectedHash for edit_file/write_file when editing the same content without rereading. If a stale write is reported, read the file again before retrying.
{{snapshots}}
