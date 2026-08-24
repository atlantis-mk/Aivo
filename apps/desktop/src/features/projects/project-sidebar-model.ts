import type { domain } from "../../../bridge/go/models";

export type ProjectConversationGroup = {
  conversations: domain.Session[];
  project: domain.AssistantProject;
  projectPath: string;
};

export function upsertRecentProject(
  projects: domain.AssistantProject[],
  project: domain.AssistantProject,
) {
  return [
    project,
    ...projects.filter((item) => item.rootPath !== project.rootPath),
  ].slice(0, 20);
}

export function buildProjectConversationGroups(
  sessions: domain.Session[],
  projects: domain.AssistantProject[],
  selectedProjectPath: string,
): ProjectConversationGroup[] {
  const projectsByPathKey = new Map<string, domain.AssistantProject>();
  const orderedProjectPathKeys: string[] = [];
  const conversationsByPathKey = new Map<string, domain.Session[]>();

  const addProject = (project: domain.AssistantProject) => {
    const pathKey = normalizeProjectPathKey(project.rootPath);
    if (!pathKey || !projectIsUserSelectable(project)) return;
    if (!projectsByPathKey.has(pathKey)) {
      projectsByPathKey.set(pathKey, project);
      orderedProjectPathKeys.push(pathKey);
    }
  };

  if (selectedProjectPath) {
    addProject(assistantProjectFromPath(selectedProjectPath));
  }

  for (const project of projects) {
    addProject(project);
  }

  for (const session of sessions) {
    const projectPath = sessionSidebarProjectPath(session);
    const pathKey = normalizeProjectPathKey(projectPath);
    if (!pathKey || !projectsByPathKey.has(pathKey)) continue;
    conversationsByPathKey.set(pathKey, [
      ...(conversationsByPathKey.get(pathKey) ?? []),
      session,
    ]);
  }

  return orderedProjectPathKeys
    .map((pathKey) => {
      const project = projectsByPathKey.get(pathKey);
      if (!project) return null;
      return {
        conversations: conversationsByPathKey.get(pathKey) ?? [],
        project,
        projectPath: project.rootPath,
      } satisfies ProjectConversationGroup;
    })
    .filter((group): group is ProjectConversationGroup => Boolean(group));
}

export function isSessionGroupedUnderProject(
  session: domain.Session,
  projectGroups: ProjectConversationGroup[],
) {
  const pathKey = normalizeProjectPathKey(sessionSidebarProjectPath(session));
  if (!pathKey) return false;
  return getProjectGroupPathKeySet(projectGroups).has(pathKey);
}

export function filterSessionsOutsideProjectGroups(
  sessions: domain.Session[],
  projectGroups: ProjectConversationGroup[],
) {
  const projectPathKeys = getProjectGroupPathKeySet(projectGroups);
  return sessions.filter((session) => {
    const pathKey = normalizeProjectPathKey(sessionSidebarProjectPath(session));
    return !pathKey || !projectPathKeys.has(pathKey);
  });
}

export function normalizeProjectPathKey(projectPath: string) {
  return projectPath.trim().replace(/[\\/]+$/, "");
}

export function projectPickerLabel(
  project: domain.AssistantProject | null,
  projectPath: string,
) {
  if (project?.name) return project.name;
  if (projectPath) return projectNameFromPath(projectPath);
  return "项目选择";
}

export function projectNameFromPath(rootPath: string) {
  const trimmed = rootPath.trim().replace(/[\\/]+$/, "");
  const parts = trimmed.split(/[\\/]/).filter(Boolean);
  return parts.at(-1) || trimmed || "Project";
}

export function projectIsUserSelectable(project: domain.AssistantProject) {
  return !isManagedWorkspacePath(project.rootPath);
}

export function sessionProjectName(session: domain.Session) {
  const projectPath = sessionSidebarProjectPath(session);
  return projectPath ? projectNameFromPath(projectPath) : "";
}

function sessionSidebarProjectPath(session: domain.Session) {
  const projectPath = session.projectPath?.trim() ?? "";
  if (!projectPath || isManagedWorkspacePath(projectPath)) return "";
  return projectPath;
}

function getProjectGroupPathKeySet(projectGroups: ProjectConversationGroup[]) {
  return new Set(
    projectGroups.map((group) => normalizeProjectPathKey(group.projectPath)),
  );
}

function assistantProjectFromPath(rootPath: string): domain.AssistantProject {
  const now = new Date().toISOString();
  return {
    id: rootPath,
    name: projectNameFromPath(rootPath),
    rootPath,
    gitAvailable: false,
    timeOpened: now,
    timeUpdated: now,
  };
}

function isManagedWorkspacePath(rootPath: string) {
  const parts = rootPath.trim().split(/[\\/]/).filter(Boolean);
  const rootIndex = parts.lastIndexOf("Aivo Workspaces");
  if (rootIndex < 0) return false;
  const datePart = parts[rootIndex + 1] ?? "";
  const workspacePart = parts[rootIndex + 2] ?? "";
  return (
    /^\d{4}-\d{2}-\d{2}$/.test(datePart) &&
    isManagedWorkspaceSlug(workspacePart)
  );
}

function isManagedWorkspaceSlug(value: string) {
  return (
    value === "session" ||
    value.startsWith("session-") ||
    /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}(?:-\d+)?$/.test(
      value,
    )
  );
}
