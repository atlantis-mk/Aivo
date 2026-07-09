export const EMPTY_BROWSER_STATE: AivoBrowserState = {
  url: "",
  title: "",
  favicon: "",
  loading: false,
  canGoBack: false,
  canGoForward: false,
  error: "",
};

export type BuiltinBrowserAction = "back" | "forward" | "reload" | "stop";

type BuiltinBrowserBridge = NonNullable<NonNullable<typeof window.aivo>["browser"]>;

export function normalizeBrowserURL(input: string) {
  const raw = input.trim();
  if (!raw) {
    throw new Error("Browser address is required");
  }
  const withScheme = /^[a-zA-Z][a-zA-Z\d+.-]*:/.test(raw)
    ? raw
    : `https://${raw}`;
  const parsed = new URL(withScheme);
  if (parsed.protocol !== "http:" && parsed.protocol !== "https:") {
    throw new Error("Only http(s) addresses can be opened in the built-in browser");
  }
  return parsed.toString();
}

export function navigatingBrowserState(
  currentState: AivoBrowserState,
  nextUrl: string,
): AivoBrowserState {
  return {
    ...currentState,
    error: "",
    favicon: "",
    loading: true,
    url: nextUrl,
  };
}

export async function waitForBuiltinBrowserBounds({
  isDisposed,
  updateBrowserBounds,
}: {
  isDisposed: () => boolean;
  updateBrowserBounds: () => Promise<boolean>;
}) {
  for (let attempt = 0; attempt < 8; attempt += 1) {
    if (isDisposed()) return false;
    if (await updateBrowserBounds().catch(() => false)) {
      return true;
    }
    await new Promise<void>((resolve) => requestAnimationFrame(() => resolve()));
  }
  return false;
}

export async function runBuiltinBrowserAction({
  action,
  browser,
  browserTabId,
}: {
  action: BuiltinBrowserAction;
  browser: BuiltinBrowserBridge;
  browserTabId: string;
}) {
  if (action === "back") {
    return browser.goBack(browserTabId);
  }
  if (action === "forward") {
    return browser.goForward(browserTabId);
  }
  if (action === "stop") {
    return browser.stop(browserTabId);
  }
  return browser.reload(browserTabId);
}
