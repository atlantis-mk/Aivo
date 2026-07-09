import { useLayoutEffect, useState } from "react";

import type { AutoTextareaHeightRef } from "@/features/projects/project-prompt-composer-types";

export function useAutoTextareaHeight(
  value: string,
  minHeight: number,
  maxHeight: number,
  textareaRef: AutoTextareaHeightRef,
) {
  const [height, setHeight] = useState({
    content: 0,
    viewport: 0,
  });

  useLayoutEffect(() => {
    const textarea = textareaRef.current;
    if (!textarea) return;

    textarea.style.height = "";
    const baseHeight = textarea.offsetHeight;
    textarea.style.height = "0px";
    const measuredHeight = Math.max(textarea.scrollHeight, minHeight);
    const contentHeight = measuredHeight > baseHeight ? measuredHeight : 0;
    const viewportHeight =
      contentHeight > 0 ? Math.min(contentHeight, maxHeight) : 0;

    textarea.style.height = contentHeight > 0 ? `${contentHeight}px` : "";
    setHeight((current) => {
      if (
        current.content === contentHeight &&
        current.viewport === viewportHeight
      )
        return current;
      return {
        content: contentHeight,
        viewport: viewportHeight,
      };
    });
  }, [maxHeight, minHeight, textareaRef, value]);

  return height;
}
