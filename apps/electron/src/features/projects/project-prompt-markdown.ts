const markdownPatterns = [
  /^(?: {0,3})#{1,6}\s+\S/m,
  /^(?: {0,3})(?:[-*+])\s+\S/m,
  /^(?: {0,3})\d+[.)]\s+\S/m,
  /^(?: {0,3})>\s+\S/m,
  /^(?: {0,3})(?:```|~~~)[\s\S]*?(?:```|~~~)\s*$/m,
  /^\|.+\|\s*\n\|(?:\s*:?-{3,}:?\s*\|)+\s*$/m,
  /!?(?:\[[^\]\n]+\]\([^\)\n]+\))/, // links and images
  /(?<!\*)\*\*[^*\n]+\*\*(?!\*)|(?<!_)__[^_\n]+__(?!_)/,
  /(?<!`)`[^`\n]+`(?!`)/,
];

/**
 * A plain sentence is valid Markdown too, but rendering every sentence as a
 * preview would make the composer hard to edit. Only switch once the text has
 * a complete, visible Markdown construct.
 */
export function hasRenderableMarkdown(value: string) {
  const content = value.trim();
  return (
    content.length > 0 &&
    markdownPatterns.some((pattern) => pattern.test(content))
  );
}
