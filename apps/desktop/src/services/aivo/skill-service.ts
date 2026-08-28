import type { SessionEvent } from "@/services/aivo/session-event-service";
import { invoke } from "@/services/aivo/invoke";

export type SkillEntry = {
  id: string;
  name: string;
  description: string;
  scope: string;
  source: string;
  rootPath: string;
  skillPath: string;
  contentHash: string;
  enabled: boolean;
  metadata?: Record<string, string>;
  timeCreated: string;
  timeUpdated: string;
};

export type SkillImportCandidate = {
  id: string;
  name: string;
  description: string;
  scope: string;
  source: string;
  rootPath: string;
  skillPath: string;
  contentHash: string;
  status: string;
  conflictId?: string;
  error?: string;
  lastSeenAt: string;
};

export type SkillScanResult = {
  entries?: SkillEntry[];
  candidates?: SkillImportCandidate[];
  scanned: number;
  imported: number;
  conflicts: number;
  errors?: string[];
};

export type SkillListResult = {
  entries: SkillEntry[];
  candidates?: SkillImportCandidate[];
};

export type SkillEditResult = {
  skill: SkillEntry;
  content: string;
};

export type SessionActiveSkillsResult = {
  sessionId: string;
  skillIds: string[];
  skills?: SkillEntry[];
};

export function scanGlobalSkills() {
  return invoke<SkillScanResult>("ScanGlobalSkills");
}

export function scanProjectSkills(workspaceRoot: string) {
  return invoke<SkillScanResult>("ScanProjectSkills", { workspaceRoot });
}

export function listSkills(input: {
  workspaceRoot?: string;
  includeCandidates?: boolean;
  includeDisabled?: boolean;
  includeIgnored?: boolean;
} = {}) {
  return invoke<SkillListResult>("ListSkills", input);
}

export function importSkill(candidateId: string, targetScope?: string) {
  return invoke<SkillEntry>("ImportSkill", { candidateId, targetScope });
}

export function ignoreSkillCandidatesByName(name: string) {
  return invoke<SkillImportCandidate[]>("IgnoreSkillCandidatesByName", { name });
}

export function setSkillEnabled(skillId: string, enabled: boolean) {
  return invoke<SkillEntry>("SetSkillEnabled", { skillId, enabled });
}

export function getManagedSkillForEdit(skillId: string) {
  return invoke<SkillEditResult>("GetManagedSkillForEdit", skillId);
}

export function updateManagedSkill(input: {
  skillId: string;
  description: string;
  content: string;
  expectedContentHash: string;
}) {
  return invoke<SkillEditResult>("UpdateManagedSkill", input);
}

export function deleteManagedSkill(skillId: string) {
  return invoke<{ ok: boolean }>("DeleteManagedSkill", skillId);
}

export function loadSkillIntoSession(input: {
  sessionId: string;
  skillId?: string;
  name?: string;
  scope?: string;
  reason?: string;
  reload?: boolean;
}) {
  return invoke<SessionEvent>("LoadSkillIntoSession", input);
}

export function getSessionActiveSkills(sessionId: string) {
  return invoke<SessionActiveSkillsResult>("GetSessionActiveSkills", sessionId);
}

export function setSessionActiveSkills(sessionId: string, skillIds: string[]) {
  return invoke<SessionActiveSkillsResult>("SetSessionActiveSkills", {
    sessionId,
    skillIds,
  });
}
