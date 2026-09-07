export async function invoke<T>(..._args: unknown[]): Promise<T> {
  throw new Error("Legacy desktop bridge service is unavailable.");
}
