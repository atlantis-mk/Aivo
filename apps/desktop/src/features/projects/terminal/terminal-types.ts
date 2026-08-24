export type TerminalRuntimeStatus =
  | "connecting"
  | "running"
  | "reconnecting"
  | "exited"
  | "failed";

export type PersistedTerminalStatus = Exclude<
  TerminalRuntimeStatus,
  "reconnecting"
>;
