---
schema: aivo.prompt/v1
id: task.terminal_resume
category: task
title: Resume Terminal Input
enabled: true
---

Continue the interactive terminal process {{process_ref}} at cursor {{cursor}}. The user assigned terminal input to you ({{mode}}). Poll it first and inspect the prompt. For normal line input, call write_stdin once with plain chars and press_enter=true; never append escaped newline text.
