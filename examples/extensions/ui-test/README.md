# Aivo UI test extension

This development-only Manifest v2 extension exercises Aivo's isolated
`tool-detail` Web view. It contributes the `ui_test_show` tool and associates
the tool with an interactive page served by the same supervised local service.
The service binds `127.0.0.1:0` and reports the operating-system-assigned port
through the bounded `aivo-extension-service/1` readiness handshake, so it does
not reserve or collide on a fixed development port.

## Register with a running Aivo core

From the repository root, start Aivo:

```bash
pnpm dev
```

In another terminal, register, trust, and enable the extension:

```bash
node examples/extensions/ui-test/register.mjs
```

Then ask Aivo to call the tool explicitly:

```text
调用 ui_test_show，并把 message 设置为“你好，Aivo 扩展界面”。
```

The tool detail inspector should open the test page automatically. Use its
buttons to exercise same-origin state loading, declared View actions, Host
notifications, and guest-requested close. The page also opens a Chrome-style
`aivoExtension.runtime` Port. Repeated tool calls update the existing page over
the authenticated Host-brokered stream without navigation; text left in the
input remains a visible page-instance continuity check.

If the core uses a non-default URL, pass it explicitly:

```bash
node examples/extensions/ui-test/register.mjs --core-url http://127.0.0.1:43118
```

## Test the extension service

```bash
node --test examples/extensions/ui-test/server.test.mjs
```

The service intentionally requires the Host-provided bearer token. The Web page
does not receive that token, Node integration, Electron IPC, arbitrary Aivo RPC,
tool arguments, or tool results. It receives only the bounded v1 View bridge and
the Manifest v2 `runtime.messaging` capability. One-shot messages and Port
traffic use fixed well-known service endpoints while Electron retains the
backend origin and bearer.
