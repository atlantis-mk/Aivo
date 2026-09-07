import { Plus, Trash2 } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { emptyCustomRow } from "@/features/providers/custom-provider-form";
import type {
  CustomProviderForm,
  CustomProviderProtocol,
  CustomProviderRow,
} from "@/features/providers/provider-types";

const customProviderProtocols: Array<{
  id: CustomProviderProtocol;
  label: string;
}> = [
  { id: "openai", label: "OpenAI Responses" },
  { id: "responses", label: "Responses API" },
  { id: "openai-compatible", label: "OpenAI Compatible" },
  { id: "anthropic", label: "Anthropic Messages" },
  { id: "google", label: "Google Gemini" },
  { id: "openrouter", label: "OpenRouter" },
];

export function CustomProviderFormFields({
  form,
  onChange,
}: {
  form: CustomProviderForm;
  onChange: (form: CustomProviderForm) => void;
}) {
  function updateField(
    field: keyof Omit<CustomProviderForm, "models" | "headers">,
    value: string,
  ) {
    onChange({ ...form, [field]: value });
  }

  function updateRow(
    section: "models" | "headers",
    id: string,
    field: "name" | "value",
    value: string,
  ) {
    onChange({
      ...form,
      [section]: form[section].map((row) =>
        row.id === id ? { ...row, [field]: value } : row,
      ),
    });
  }

  function addRow(section: "models" | "headers") {
    onChange({ ...form, [section]: [...form[section], emptyCustomRow()] });
  }

  function removeRow(section: "models" | "headers", id: string) {
    const nextRows = form[section].filter((row) => row.id !== id);
    onChange({
      ...form,
      [section]: nextRows.length > 0 ? nextRows : [emptyCustomRow()],
    });
  }

  return (
    <div className="flex flex-col gap-4 text-left">
      <CustomProviderInput
        description="使用小写字母、数字、连字符或下划线"
        label="提供商 ID"
        onChange={(value) => updateField("providerId", value)}
        placeholder="myprovider"
        value={form.providerId}
      />
      <CustomProviderInput
        label="显示名称"
        onChange={(value) => updateField("displayName", value)}
        placeholder="我的 AI 提供商"
        value={form.displayName}
      />
      <div className="flex flex-col gap-1.5">
        <label className="text-sm ">协议</label>
        <Select
          onValueChange={(value: string) =>
            updateField("protocol", value as CustomProviderProtocol)
          }
          value={form.protocol}
        >
          <SelectTrigger className="h-9  px-3 text-sm">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {customProviderProtocols.map((protocol) => (
              <SelectItem key={protocol.id} value={protocol.id}>
                {protocol.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>
      <CustomProviderInput
        label="基础 URL"
        onChange={(value) => updateField("baseUrl", value)}
        placeholder={customProviderBaseURLPlaceholder(form.protocol)}
        value={form.baseUrl}
      />
      <CustomProviderInput
        description="可选。如果你通过请求头管理认证，可留空。"
        label="API 密钥"
        onChange={(value) => updateField("apiKey", value)}
        placeholder="API 密钥"
        value={form.apiKey}
      />

      <CustomRows
        addLabel="添加模型"
        leftPlaceholder="model-id"
        onAdd={() => addRow("models")}
        onRemove={(id) => removeRow("models", id)}
        onUpdate={(id, field, value) => updateRow("models", id, field, value)}
        removeLabel="移除模型"
        rightPlaceholder="显示名称"
        rows={form.models}
        title="模型"
      />

      <CustomRows
        addLabel="添加请求头"
        leftPlaceholder="Header-Name"
        onAdd={() => addRow("headers")}
        onRemove={(id) => removeRow("headers", id)}
        onUpdate={(id, field, value) => updateRow("headers", id, field, value)}
        removeLabel="移除请求头"
        rightPlaceholder="value"
        rows={form.headers}
        title="请求头（可选）"
      />
    </div>
  );
}

function CustomProviderInput({
  description,
  label,
  onChange,
  placeholder,
  value,
}: {
  description?: string;
  label: string;
  onChange: (value: string) => void;
  placeholder: string;
  value: string;
}) {
  return (
    <label className="flex flex-col gap-1.5 text-sm ">
      {label}
      <Input
        onChange={(event) => onChange(event.target.value)}
        placeholder={placeholder}
        value={value}
      />
      {description ? (
        <span className="text-xs font-normal text-muted-foreground">
          {description}
        </span>
      ) : null}
    </label>
  );
}

function CustomRows({
  addLabel,
  leftPlaceholder,
  onAdd,
  onRemove,
  onUpdate,
  removeLabel,
  rightPlaceholder,
  rows,
  title,
}: {
  addLabel: string;
  leftPlaceholder: string;
  onAdd: () => void;
  onRemove: (id: string) => void;
  onUpdate: (id: string, field: "name" | "value", value: string) => void;
  removeLabel: string;
  rightPlaceholder: string;
  rows: CustomProviderRow[];
  title: string;
}) {
  return (
    <div className="flex flex-col gap-2">
      <div className="text-sm  text-muted-foreground">{title}</div>
      {rows.map((row) => (
        <div
          className="grid grid-cols-[minmax(0,1fr)_minmax(0,1fr)_auto] items-center gap-2 max-[420px]:grid-cols-[minmax(0,1fr)_auto]"
          key={row.id}
        >
          <Input
            aria-label={`${title} ID`}
            onChange={(event) => onUpdate(row.id, "name", event.target.value)}
            placeholder={leftPlaceholder}
            value={row.name}
          />
          <Input
            aria-label={`${title} 值`}
            className="max-[420px]:col-span-2 max-[420px]:col-start-1"
            onChange={(event) => onUpdate(row.id, "value", event.target.value)}
            placeholder={rightPlaceholder}
            value={row.value}
          />
          <Button
            aria-label={removeLabel}
            className="max-[420px]:col-start-2 max-[420px]:row-start-1"
            disabled={rows.length === 1}
            onClick={() => onRemove(row.id)}
            size="icon"
            type="button"
            variant="ghost"
          >
            <Trash2 />
          </Button>
        </div>
      ))}
      <Button className="self-start" onClick={onAdd} type="button" variant="ghost">
        <Plus />
        {addLabel}
      </Button>
    </div>
  );
}

function customProviderBaseURLPlaceholder(protocol: CustomProviderProtocol) {
  if (protocol === "openai") return "https://api.openai.com/v1";
  if (protocol === "anthropic") return "https://api.anthropic.com/v1";
  if (protocol === "google") {
    return "https://generativelanguage.googleapis.com/v1beta";
  }
  if (protocol === "openrouter") return "https://openrouter.ai/api/v1";
  return "https://api.myprovider.com/v1";
}
