export function invokeBridge<T>(
  method: string,
  ...args: unknown[]
): Promise<T> {
  return window.aivo.invoke<T>(method, ...args);
}
