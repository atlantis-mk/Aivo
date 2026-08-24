import assert from "node:assert/strict";
import test from "node:test";

import {
  desktopUpdateAction,
  desktopUpdateStatusLabel,
  initialDesktopUpdateState,
} from "../src/features/updates/desktop-update-model.ts";

test("desktop update actions follow the trusted check-download-handoff flow", () => {
  assert.deepEqual(desktopUpdateAction(initialDesktopUpdateState, "darwin"), {
    action: "check",
    label: "检查更新",
  });
  assert.deepEqual(
    desktopUpdateAction({ ...initialDesktopUpdateState, phase: "available" }, "win32"),
    { action: "download", label: "下载更新" },
  );
  assert.deepEqual(
    desktopUpdateAction({ ...initialDesktopUpdateState, phase: "downloading" }, "darwin"),
    { action: "cancel", label: "取消下载" },
  );
  assert.deepEqual(
    desktopUpdateAction({ ...initialDesktopUpdateState, phase: "ready" }, "linux"),
    { action: "install", label: "显示更新包" },
  );
  assert.equal(
    desktopUpdateAction({ ...initialDesktopUpdateState, phase: "checking" }, "darwin"),
    null,
  );
  assert.equal(desktopUpdateStatusLabel("error"), "检查失败");
});
