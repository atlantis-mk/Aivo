import { type ComponentProps } from "react";

import { Button } from "@/components/ui/button";

export function ProjectTopBarIconButton(props: ComponentProps<typeof Button>) {
  return (
    <Button
      size="icon"
      type="button"
      variant="ghost"
      {...props}
    />
  );
}
