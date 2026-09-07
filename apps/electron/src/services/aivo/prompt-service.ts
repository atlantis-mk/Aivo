import { invoke } from "@/services/aivo/invoke";

export type PromptCategory =
  | "agent"
  | "protocol"
  | "auxiliary"
  | "task"
  | "dynamic_context"
  | "quick_prompt";

export type PromptDiagnostic = {
  code: string;
  message: string;
  line?: number;
  column?: number;
};

export type PromptDocument = {
  id: string;
  category: PromptCategory;
  title: string;
  body: string;
  enabled: boolean;
  origin: "builtin" | "override" | string;
  required: boolean;
  disableable: boolean;
  deletable: boolean;
  variables?: string[];
  requiredVariables?: string[];
  maxLength: number;
  workingRevision?: string;
  activeRevision?: string;
  status: "valid" | "invalid" | "disabled" | string;
  fallback?: boolean;
  diagnostics?: PromptDiagnostic[];
};

export type PromptDocumentInput = Pick<
  PromptDocument,
  "id" | "category" | "title" | "body" | "enabled"
>;

export type PromptToolDescription = {
  name: string;
  description: string;
  category?: string;
  source?: string;
};

export const listPromptDocuments = () =>
  invoke<PromptDocument[]>("ListPromptDocuments");
export const getPromptDocument = (id: string) =>
  invoke<PromptDocument>("GetPromptDocument", id);
export const validatePromptDraft = (input: PromptDocumentInput) =>
  invoke<{ valid: boolean; revision?: string; diagnostics?: PromptDiagnostic[] }>(
    "ValidatePromptDraft",
    input,
  );
export const savePromptDocument = (input: PromptDocumentInput) =>
  invoke<PromptDocument>("SavePromptDocument", input);
export const resetPromptDocument = (id: string) =>
  invoke<PromptDocument>("ResetPromptDocument", { id });
export const setPromptDocumentEnabled = (id: string, enabled: boolean) =>
  invoke<PromptDocument>("SetPromptDocumentEnabled", { id, enabled });
export const deletePromptDocument = (id: string) =>
  invoke<void>("DeletePromptDocument", { id });
export const reloadPromptCatalog = () =>
  invoke<PromptDocument[]>("ReloadPromptCatalog");
export const getPromptDirectory = () => invoke<string>("PromptDirectory");
export const listPromptToolDescriptions = () =>
  invoke<PromptToolDescription[]>("ListPromptToolDescriptions");
export const createAgentPrompt = (input: {
  id: string;
  title: string;
  body: string;
  description?: string;
  permissionScope?: string;
  mode?: "primary" | "subagent" | "all";
  subagents?: string[];
}) => invoke("CreateAgentPrompt", input);
export const createQuickPrompt = (input: {
  id: string;
  title: string;
  body: string;
}) => invoke<PromptDocument>("CreateQuickPrompt", input);
