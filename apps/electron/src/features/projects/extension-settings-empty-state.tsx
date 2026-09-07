import { Layers3 } from "lucide-react";

import {
  Empty,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from "@/components/ui/empty";

export function EmptyState({ label }: { label: string }) {
  return (
    <Empty>
      <EmptyMedia variant="icon">
        <Layers3 />
      </EmptyMedia>
      <EmptyHeader>
        <EmptyTitle>{label}</EmptyTitle>
        <EmptyDescription>调整搜索条件或刷新后重试。</EmptyDescription>
      </EmptyHeader>
    </Empty>
  );
}
