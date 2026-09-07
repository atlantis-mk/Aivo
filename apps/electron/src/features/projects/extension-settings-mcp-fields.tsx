import type { ReactNode } from "react";
import { Plus, Trash2 } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import type { KeyValueRow } from "@/features/projects/extension-settings-model";

export function McpField({
  action,
  children,
  label,
}: {
  action?: ReactNode;
  children: ReactNode;
  label: string;
}) {
  return (
    <div className="grid gap-2">
      <div className="flex min-w-0 items-center justify-between gap-2">
        <Label>{label}</Label>
        {action}
      </div>
      {children}
    </div>
  );
}

export function McpStringRows({
  addLabel,
  label,
  onChange,
  placeholder,
  rows,
}: {
  addLabel: string;
  label: string;
  onChange: (rows: string[]) => void;
  placeholder: string;
  rows: string[];
}) {
  return (
    <div className="grid gap-2">
      <Label>{label}</Label>
      {rows.map((row, index) => (
        <div
          className="grid grid-cols-[minmax(0,1fr)_auto] items-center gap-2"
          key={index}
        >
          <Input
            onChange={(event) =>
              onChange(
                rows.map((item, itemIndex) =>
                  itemIndex === index ? event.target.value : item,
                ),
              )
            }
            placeholder={placeholder}
            value={row}
          />
          <Button
            aria-label={`删除${label}`}
            disabled={rows.length === 1}
            onClick={() =>
              onChange(rows.filter((_item, itemIndex) => itemIndex !== index))
            }
            size="icon"
            type="button"
            variant="ghost"
          >
            <Trash2 />
          </Button>
        </div>
      ))}
      <Button
        onClick={() => onChange([...rows, ""])}
        type="button"
        variant="secondary"
      >
        <Plus />
        {addLabel}
      </Button>
    </div>
  );
}

export function McpKeyValueRows({
  addLabel,
  label,
  leftPlaceholder,
  onChange,
  rightPlaceholder,
  rows,
}: {
  addLabel: string;
  label: string;
  leftPlaceholder: string;
  onChange: (rows: KeyValueRow[]) => void;
  rightPlaceholder: string;
  rows: KeyValueRow[];
}) {
  return (
    <div className="grid gap-2">
      <Label>{label}</Label>
      {rows.map((row, index) => (
        <div
          className="grid grid-cols-[minmax(0,1fr)_minmax(0,1fr)_auto] items-center gap-2"
          key={index}
        >
          <Input
            onChange={(event) =>
              onChange(
                rows.map((item, itemIndex) =>
                  itemIndex === index
                    ? { ...item, key: event.target.value }
                    : item,
                ),
              )
            }
            placeholder={leftPlaceholder}
            value={row.key}
          />
          <Input
            onChange={(event) =>
              onChange(
                rows.map((item, itemIndex) =>
                  itemIndex === index
                    ? { ...item, value: event.target.value }
                    : item,
                ),
              )
            }
            placeholder={rightPlaceholder}
            value={row.value}
          />
          <Button
            aria-label={`删除${label}`}
            disabled={rows.length === 1}
            onClick={() =>
              onChange(rows.filter((_item, itemIndex) => itemIndex !== index))
            }
            size="icon"
            type="button"
            variant="ghost"
          >
            <Trash2 />
          </Button>
        </div>
      ))}
      <Button
        onClick={() => onChange([...rows, { key: "", value: "" }])}
        type="button"
        variant="secondary"
      >
        <Plus />
        {addLabel}
      </Button>
    </div>
  );
}
