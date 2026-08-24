import { CheckCircle2, type LucideIcon, TriangleAlert } from "lucide-react";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  applySimpleTemplate,
  templateVariables,
} from "@/features/projects/extension-settings-model";
import type { MCPPromptRecord, MCPResourceRecord } from "@/services/aivo";

export function SectionHeading({
  icon: Icon,
  label,
}: {
  icon: LucideIcon;
  label: string;
}) {
  return (
    <div className="flex items-center gap-2 text-xs font-medium text-foreground">
      <Icon />
      <span>{label}</span>
    </div>
  );
}

export function PromptActionLine({
  prompt,
  inputs,
  loading,
  inserting,
  onInputChange,
  onInsert,
  onRun,
}: {
  prompt: MCPPromptRecord;
  inputs: Record<string, string>;
  loading: boolean;
  inserting?: boolean;
  onInputChange: (name: string, value: string) => void;
  onInsert?: () => void;
  onRun: () => void;
}) {
  const missingRequired = (prompt.arguments ?? []).some(
    (argument) => argument.required && !inputs[argument.name]?.trim(),
  );
  return (
    <div className="grid gap-2 text-xs">
      <CapabilityLine
        label="prompt"
        name={prompt.name}
        detail={prompt.description}
      />
      {prompt.arguments?.length ? (
        <div className="grid gap-2 sm:grid-cols-2">
          {prompt.arguments.map((argument) => (
            <Input
              key={argument.name}
              className="h-8 text-xs"
              placeholder={`${argument.name}${argument.required ? " *" : ""}`}
              title={argument.description || argument.name}
              value={inputs[argument.name] ?? ""}
              onChange={(event) =>
                onInputChange(argument.name, event.target.value)
              }
            />
          ))}
        </div>
      ) : null}
      <div className="flex flex-wrap gap-2">
        <Button
          disabled={loading || missingRequired}
          onClick={onRun}
          size="sm"
          variant="outline"
        >
          {loading ? "读取中" : "预览 prompt"}
        </Button>
        {onInsert ? (
          <Button
            disabled={inserting || missingRequired}
            onClick={onInsert}
            size="sm"
            variant="outline"
          >
            {inserting ? "插入中" : "插入会话"}
          </Button>
        ) : null}
      </div>
    </div>
  );
}

export function ResourceActionLine({
  resource,
  inputs = {},
  loading,
  inserting,
  onInputChange,
  onInsert,
  onRead,
}: {
  resource: MCPResourceRecord;
  inputs?: Record<string, string>;
  loading: boolean;
  inserting?: boolean;
  onInputChange?: (name: string, value: string) => void;
  onInsert?: (uri: string) => void;
  onRead: (uri: string) => void;
}) {
  const template = resource.uriTemplate ?? "";
  const variables = templateVariables(template);
  const resolvedURI = resource.template
    ? applySimpleTemplate(template, inputs)
    : (resource.uri ?? "");
  const missingRequired =
    resource.template && variables.some((name) => !inputs[name]?.trim());
  return (
    <div className="grid gap-2 text-xs">
      <CapabilityLine
        label={resource.template ? "template" : "resource"}
        name={resource.name}
        detail={resource.uri || resource.uriTemplate || resource.description}
      />
      {resource.template && variables.length > 0 ? (
        <div className="grid gap-2 sm:grid-cols-2">
          {variables.map((name) => (
            <Input
              key={name}
              className="h-8 text-xs"
              placeholder={name}
              value={inputs[name] ?? ""}
              onChange={(event) => onInputChange?.(name, event.target.value)}
            />
          ))}
        </div>
      ) : null}
      <div className="flex min-w-0 items-center gap-2">
        <Button
          disabled={loading || !resolvedURI || missingRequired}
          onClick={() => onRead(resolvedURI)}
          size="sm"
          variant="outline"
        >
          {loading ? "读取中" : "读取"}
        </Button>
        {onInsert ? (
          <Button
            disabled={inserting || !resolvedURI || missingRequired}
            onClick={() => onInsert(resolvedURI)}
            size="sm"
            variant="outline"
          >
            {inserting ? "插入中" : "插入会话"}
          </Button>
        ) : null}
        {resource.template ? (
          <span className="min-w-0 truncate text-muted-foreground">
            {resolvedURI}
          </span>
        ) : null}
      </div>
    </div>
  );
}

export function CapabilityLine({
  label,
  name,
  detail,
}: {
  label: string;
  name: string;
  detail?: string;
}) {
  return (
    <div className="flex min-w-0 items-center gap-2">
      <span className="shrink-0 rounded bg-muted px-1.5 py-0.5 text-[10px] uppercase tracking-wide">
        {label}
      </span>
      <span className="min-w-0 truncate text-foreground">{name}</span>
      {detail ? <span className="min-w-0 truncate">{detail}</span> : null}
    </div>
  );
}

export function PreviewBlock({
  label,
  value,
}: {
  label: string;
  value: string;
}) {
  return (
    <div className="mt-3">
      <div className="mb-1 truncate text-xs font-medium text-foreground">
        {label}
      </div>
      <pre className="max-h-48 overflow-auto whitespace-pre-wrap break-words rounded-md bg-muted/50 px-3 py-2 font-mono text-xs leading-relaxed text-muted-foreground">
        {value || "没有可预览内容"}
      </pre>
    </div>
  );
}

export function StatusBadge({ status }: { status?: string }) {
  const ok = status === "ready" || status === "enabled";
  return (
    <Badge variant={ok ? "secondary" : "outline"}>
      {ok ? <CheckCircle2 /> : <TriangleAlert />}
      {status || "unknown"}
    </Badge>
  );
}
