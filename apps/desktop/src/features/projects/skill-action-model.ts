import type { SkillEntry } from "@/services/aivo";

export const SKILL_ACTION_ACTIVATE = "activate";
export const SKILL_ACTION_SET_ENABLED = "set_enabled";
export const SKILL_ACTION_EDIT = "edit";
export const SKILL_ACTION_DELETE = "delete";

function isSystemSkill(skill: SkillEntry) {
  return (
    skill.source === "aivo-system" ||
    skill.source === "codex-system" ||
    Boolean(skill.metadata?.["aivo.system"])
  );
}

export function skillSupportsAction(skill: SkillEntry, action: string) {
  if (skill.actions) {
    return skill.actions.includes(action);
  }
  if (action === SKILL_ACTION_ACTIVATE) {
    return skill.enabled;
  }
  if (isSystemSkill(skill)) {
    return false;
  }
  if (action === SKILL_ACTION_SET_ENABLED || action === SKILL_ACTION_DELETE) {
    return true;
  }
  return action === SKILL_ACTION_EDIT && skill.source === "aivo";
}

export function skillCanActivate(skill: SkillEntry) {
  return skillSupportsAction(skill, SKILL_ACTION_ACTIVATE);
}
