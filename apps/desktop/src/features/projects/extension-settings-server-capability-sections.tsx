import { FileText, MessageSquareText } from "lucide-react";

import {
  PromptActionLine,
  ResourceActionLine,
  SectionHeading,
} from "@/features/projects/extension-settings-components";
import type {
  MCPPromptRecord,
  MCPResourceRecord,
} from "@/services/aivo";

export function McpPromptSection({
  insertingPromptId,
  loadingPromptId,
  onInputChange,
  onInsert,
  onRun,
  promptInputs,
  prompts,
  sessionId,
}: {
  insertingPromptId: string;
  loadingPromptId: string;
  onInputChange: (promptId: string, name: string, value: string) => void;
  onInsert: (prompt: MCPPromptRecord) => void;
  onRun: (prompt: MCPPromptRecord) => void;
  promptInputs: Record<string, Record<string, string>>;
  prompts: MCPPromptRecord[];
  sessionId?: string;
}) {
  if (prompts.length === 0) {
    return null;
  }
  return (
    <div className="grid gap-3 rounded-md border p-3">
      <SectionHeading icon={MessageSquareText} label="Prompts" />
      <div className="grid gap-2">
        {prompts.map((prompt) => (
          <PromptActionLine
            key={prompt.id}
            prompt={prompt}
            inputs={promptInputs[prompt.id] ?? {}}
            loading={loadingPromptId === prompt.id}
            inserting={insertingPromptId === prompt.id}
            onInputChange={(name, value) =>
              onInputChange(prompt.id, name, value)
            }
            onInsert={sessionId ? () => onInsert(prompt) : undefined}
            onRun={() => onRun(prompt)}
          />
        ))}
      </div>
    </div>
  );
}

export function McpResourceSection({
  insertingResourceId,
  loadingResourceId,
  onInsert,
  onRead,
  onTemplateInputChange,
  resources,
  sessionId,
  templateInputs,
  templates,
}: {
  insertingResourceId: string;
  loadingResourceId: string;
  onInsert: (resource: MCPResourceRecord, uri: string) => void;
  onRead: (resource: MCPResourceRecord, uri: string) => void;
  onTemplateInputChange: (
    templateId: string,
    name: string,
    value: string,
  ) => void;
  resources: MCPResourceRecord[];
  sessionId?: string;
  templateInputs: Record<string, Record<string, string>>;
  templates: MCPResourceRecord[];
}) {
  if (resources.length + templates.length === 0) {
    return null;
  }
  return (
    <div className="grid gap-3 rounded-md border p-3">
      <SectionHeading icon={FileText} label="Resources" />
      <div className="grid gap-2">
        {resources.map((resource) => (
          <ResourceActionLine
            key={resource.id}
            resource={resource}
            loading={loadingResourceId === resource.id}
            inserting={insertingResourceId === resource.id}
            onInsert={
              sessionId ? (uri) => onInsert(resource, uri) : undefined
            }
            onRead={(uri) => onRead(resource, uri)}
          />
        ))}
        {templates.map((template) => (
          <ResourceActionLine
            key={template.id}
            resource={template}
            inputs={templateInputs[template.id] ?? {}}
            loading={loadingResourceId === template.id}
            inserting={insertingResourceId === template.id}
            onInputChange={(name, value) =>
              onTemplateInputChange(template.id, name, value)
            }
            onInsert={sessionId ? (uri) => onInsert(template, uri) : undefined}
            onRead={(uri) => onRead(template, uri)}
          />
        ))}
      </div>
    </div>
  );
}
