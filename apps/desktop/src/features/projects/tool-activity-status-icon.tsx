import {
  CheckCircle2,
  CircleAlert,
  Clock,
  Loader2,
} from "lucide-react";

import type { ToolActivityTab } from "@/features/projects/tool-activity-model";
import { cn } from "@/lib/utils";

export function ToolActivityStatusIcon({
  className,
  status,
}: {
  className?: string;
  status: ToolActivityTab["status"];
}) {
  switch (status) {
    case "success":
      return <CheckCircle2 className={cn("text-emerald-500", className)} />;
    case "failed":
      return <CircleAlert className={cn("text-destructive", className)} />;
    case "pending_approval":
      return <Clock className={cn("text-amber-500", className)} />;
    default:
      return (
        <Loader2
          className={cn("animate-spin text-muted-foreground", className)}
        />
      );
  }
}
