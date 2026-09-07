export const LARGE_PROMPT_PASTE_CHAR_THRESHOLD = 1000;

export type PendingPromptPaste = {
  id: string;
  name?: string;
  text: string;
  target?: string;
};

export type PromptPasteInput = {
  pastedText: string;
  sourcePath?: string;
};

export type PromptPasteResult = {
  paste: PendingPromptPaste;
};

export function isLargePromptPaste(text: string): boolean {
  return [...text.replace(/\r\n?/g, "\n")].length > LARGE_PROMPT_PASTE_CHAR_THRESHOLD;
}

export function createPromptPaste(
  input: PromptPasteInput,
): PromptPasteResult | null {
  const pastedText = input.pastedText.replace(/\r\n?/g, "\n");
  const target = input.sourcePath || promptPasteTarget(pastedText);
  const isLarge = [...pastedText].length > LARGE_PROMPT_PASTE_CHAR_THRESHOLD;
  if (!isLarge && !input.sourcePath) {
    return null;
  }
  const shouldKeepPastedText = isLarge || !input.sourcePath;

  return {
    paste: {
      id: crypto.randomUUID(),
      name: target
        ? promptPasteName(target)
        : inferPromptPasteName(pastedText),
      target,
      text: shouldKeepPastedText ? pastedText : input.sourcePath ?? pastedText,
    },
  };
}

export function promptPasteTitle(paste: PendingPromptPaste): string {
  const name = paste.name ?? (paste.target ? promptPasteName(paste.target) : undefined);
  return name ?? `粘贴内容（${[...paste.text].length} 字符）`;
}

export function promptPasteName(target: string): string {
  const encodedName = target.split(/[\\/]/u).at(-1)?.trim();
  if (!encodedName) return "粘贴内容";
  try {
    return decodeURIComponent(encodedName);
  } catch {
    return encodedName;
  }
}

export function inferPromptPasteName(text: string): string | undefined {
  const normalized = text.replace(/\r\n?/g, "\n").trim();
  if (!normalized) return undefined;

  const frontmatterTitle = normalized
    .slice(0, 2000)
    .match(/^---\n([\s\S]*?)\n---/u)?.[1]
    ?.split("\n")
    .map((line) => line.match(/^title\s*:\s*(.+)$/u)?.[1]?.trim())
    .find(Boolean);
  if (frontmatterTitle) {
    return normalizePromptPasteName(frontmatterTitle);
  }

  const htmlTitle = normalized
    .slice(0, 5000)
    .match(/<title[^>]*>([\s\S]*?)<\/title>/iu)?.[1];
  if (htmlTitle) {
    return normalizePromptPasteName(htmlTitle.replace(/<[^>]*>/gu, ""));
  }

  const heading = normalized
    .split("\n")
    .map((line) => line.match(/^#{1,6}\s+(.+?)\s*$/u)?.[1])
    .find(Boolean);
  if (heading) {
    return normalizePromptPasteName(heading, true);
  }

  const firstLine = normalized.split("\n").find(Boolean);
  if (firstLine && [...firstLine].length <= 80 && !/^[>/-]/u.test(firstLine)) {
    return normalizePromptPasteName(firstLine);
  }

  return undefined;
}

function normalizePromptPasteName(
  value: string,
  markdown = false,
): string | undefined {
  const name = value
    .replace(/[*_`]/gu, "")
    .replace(/\s+/gu, " ")
    .trim();
  if (!name) return undefined;
  const capped = [...name].slice(0, 64).join("");
  return markdown && !/\.[A-Za-z0-9]{1,8}$/u.test(capped)
    ? `${capped}.md`
    : capped;
}

export function promptPasteTarget(text: string): string | undefined {
  const unquoted = text.trim().replace(/^["']|["']$/g, "");
  if (!unquoted || /[\r\n]/u.test(unquoted)) return undefined;
  if (isExternalPromptPasteTarget(unquoted) && !/\s/u.test(unquoted)) {
    return unquoted;
  }
  if (unquoted.startsWith("file://")) {
    try {
      return decodeURIComponent(new URL(unquoted).pathname);
    } catch {
      return undefined;
    }
  }
  if (/^(?:\/|~\/|[A-Za-z]:[\\/]|\\\\)/u.test(unquoted)) return unquoted;
  return undefined;
}

export function isExternalPromptPasteTarget(target: string): boolean {
  try {
    const { protocol } = new URL(target);
    return ["http:", "https:", "mailto:"].includes(protocol);
  } catch {
    return false;
  }
}

export function expandPromptPastes(
  prompt: string,
  pendingPastes: PendingPromptPaste[],
): string {
  if (pendingPastes.length === 0) return prompt;
  const promptText = prompt.trim();
  return [
    ...(promptText ? [promptText] : []),
    ...pendingPastes.map((paste) => paste.text),
  ].join("\n\n");
}

export function promptWithPasteSummaries(
  prompt: string,
  pendingPastes: PendingPromptPaste[],
): string {
  if (pendingPastes.length === 0) return prompt;
  const promptText = prompt.trim();
  return [
    ...(promptText ? [promptText] : []),
    ...pendingPastes.map((paste) => `[${promptPasteTitle(paste)}]`),
  ].join("\n\n");
}
