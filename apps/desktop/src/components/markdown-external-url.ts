const EXTERNAL_LINK_PROTOCOLS = new Set(["http:", "https:", "mailto:"]);

export function isExternalMarkdownUrl(value: string): boolean {
  try {
    return EXTERNAL_LINK_PROTOCOLS.has(new URL(value).protocol);
  } catch {
    return false;
  }
}
