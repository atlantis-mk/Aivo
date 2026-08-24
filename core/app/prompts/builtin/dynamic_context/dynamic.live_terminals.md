---
schema: aivo.prompt/v1
id: dynamic.live_terminals
category: dynamic_context
title: Live Terminals
enabled: true
---

The following PTY processes are alive and persist across agent turns. A tool wait ending or being cancelled does not mean these processes exited.
If the user asks to continue, answer, type into, stop, or exit a previous command, use write_stdin with its existing processRef. Do not call exec_command merely to regain access. Start another instance only when the user explicitly requests one or no suitable live terminal exists.
For normal line input, send plain chars with press_enter=true in the same write_stdin call; never type escaped \r, \n, or \u000a text into the terminal.
{{terminals}}
