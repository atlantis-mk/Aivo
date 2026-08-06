const elements = {
  actionCount: document.querySelector("#action-count"),
  bridgeStatus: document.querySelector("#bridge-status"),
  close: document.querySelector("#close"),
  feedback: document.querySelector("#feedback"),
  interactionLabel: document.querySelector("#interaction-label"),
  locale: document.querySelector("#locale"),
  message: document.querySelector("#message"),
  notify: document.querySelector("#notify"),
  operationId: document.querySelector("#operation-id"),
  receivedAt: document.querySelector("#received-at"),
  record: document.querySelector("#record"),
  refresh: document.querySelector("#refresh"),
  sessionId: document.querySelector("#session-id"),
  surface: document.querySelector("#surface"),
  toolName: document.querySelector("#tool-name"),
};

let bridgeContext = null;
let refreshSequence = 0;
let runtimePort = null;

async function initialize() {
  if (!window.aivoExtension || window.aivoExtension.version !== 1) {
    showFailure("Aivo View bridge 不可用");
    return;
  }

  try {
    runtimePort = window.aivoExtension.runtime.connect({ name: "state-stream" });
    runtimePort.onMessage.addListener((message) => {
      if (
        message?.type === "tool-state" &&
        message.state?.operationId === bridgeContext?.context?.operationId
      ) {
        renderState(message.state, "Port 实时消息");
      } else if (message?.type === "connected") {
        elements.feedback.textContent = "Runtime Port 已连接，等待实时状态。";
      }
    });
    runtimePort.onDisconnect.addListener((_port, event) => {
      if (event?.reason !== "guest") showFailure(`Runtime Port 已断开：${event?.reason || "unknown"}`);
    });
    await window.aivoExtension.runtime.sendMessage({ type: "view.ready" });
    window.aivoExtension.onContextChanged((nextContext) => {
      void applyBridgeContext(nextContext, true).catch((error) =>
        showFailure(error instanceof Error ? error.message : String(error)),
      );
    });
    await applyBridgeContext(await window.aivoExtension.getContext(), false);
    elements.bridgeStatus.textContent = "Bridge 已连接";
    elements.bridgeStatus.classList.add("is-ready");
  } catch (error) {
    showFailure(error instanceof Error ? error.message : String(error));
  }
}

async function applyBridgeContext(nextContext, changed) {
  if (
    bridgeContext &&
    Number(nextContext?.revision || 0) < Number(bridgeContext.revision || 0)
  ) {
    return;
  }
  bridgeContext = nextContext;
  elements.surface.textContent = bridgeContext?.surface || "—";
  elements.locale.textContent = bridgeContext?.locale || "—";
  elements.toolName.textContent = bridgeContext?.context?.toolName || "—";
  elements.operationId.textContent = bridgeContext?.context?.operationId || "—";
  elements.sessionId.textContent = bridgeContext?.context?.sessionId || "—";
  runtimePort?.postMessage({
    type: "select-operation",
    operationId: bridgeContext?.context?.operationId || "",
  });
  if (changed) elements.feedback.textContent = "工具上下文已更新，页面没有重新加载。";
  await refreshState();
}

async function refreshState() {
  const operationId = bridgeContext?.context?.operationId;
  if (!operationId) throw new Error("Host 未提供 operationId");
  const sequence = ++refreshSequence;

  const response = await fetch(
    `/ui/state?operationId=${encodeURIComponent(operationId)}`,
    { cache: "no-store" },
  );
  const payload = await response.json();
  if (!response.ok || !payload.ok) {
    throw new Error(payload.error || `读取状态失败：HTTP ${response.status}`);
  }
  if (
    sequence !== refreshSequence ||
    operationId !== bridgeContext?.context?.operationId
  ) {
    return;
  }

  renderState(payload.state, `HTTP 刷新，context revision ${bridgeContext.revision ?? 1}`);
}

async function recordInteraction() {
  const operationId = bridgeContext?.context?.operationId;
  elements.record.disabled = true;
  try {
    const result = await window.aivoExtension.invokeAction("test.record", {
      operationId,
      label: elements.interactionLabel.value,
    });
    if (!result?.ok) throw new Error(result?.error || "Action 执行失败");
    renderState(result.state, "Host action");
  } catch (error) {
    showFailure(error instanceof Error ? error.message : String(error));
  } finally {
    elements.record.disabled = false;
  }
}

function renderState(state, source) {
  if (!state) return;
  elements.message.textContent = state.message;
  elements.actionCount.textContent = String(state.actionCount ?? 0);
  elements.receivedAt.textContent = formatDate(state.receivedAt);
  elements.feedback.textContent = state.lastAction
    ? `${source}：${state.lastAction.label}`
    : `${source}：状态已更新，页面实例保持不变。`;
}

function showFailure(message) {
  elements.bridgeStatus.textContent = "测试失败";
  elements.bridgeStatus.classList.add("is-error");
  elements.feedback.textContent = message;
}

function formatDate(value) {
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? "—" : date.toLocaleString();
}

elements.refresh.addEventListener("click", () => {
  elements.feedback.textContent = "正在刷新…";
  void refreshState().catch((error) => showFailure(error.message));
});
elements.record.addEventListener("click", () => void recordInteraction());
elements.notify.addEventListener("click", () => {
  void window.aivoExtension.notify({
    title: "Aivo UI Test",
    body: "扩展 View 通知桥接工作正常。",
  });
});
elements.close.addEventListener("click", () => void window.aivoExtension.close());

void initialize();
