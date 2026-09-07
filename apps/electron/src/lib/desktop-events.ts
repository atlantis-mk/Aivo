export function onDesktopEvent<T>(
  eventName: string,
  callback: (payload: T) => void,
): () => void {
  return (
    window.aivoDesktop?.codex.onRuntimeEvent((event) => {
      if (event.method !== eventName) return;
      callback(event.params as T);
    }) ?? (() => undefined)
  );
}

export async function openExternalURL(url: string): Promise<void> {
  if (window.aivoDesktop?.file) {
    await window.aivoDesktop.file.openExternal(url);
    return;
  }
  window.open(url, "_blank", "noopener,noreferrer");
}
