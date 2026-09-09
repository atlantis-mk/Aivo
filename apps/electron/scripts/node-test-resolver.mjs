export async function resolve(specifier, context, nextResolve) {
  const isAlias = specifier.startsWith("@/");
  const isRelative = specifier.startsWith("./") || specifier.startsWith("../");
  if (!isAlias && !isRelative) {
    return nextResolve(specifier, context);
  }
  // Dependencies resolve their own .js/.mjs files; only application source
  // imports participate in the extensionless TypeScript mapping.
  if (!isAlias && context.parentURL?.includes("/node_modules/")) {
    return nextResolve(specifier, context);
  }

  const sourceUrl = isAlias
    ? new URL(`../src/${specifier.slice(2)}`, import.meta.url)
    : new URL(specifier, context.parentURL);
  if (!/\.[cm]?[jt]sx?$/.test(sourceUrl.pathname)) {
    sourceUrl.pathname += ".ts";
  }
  return nextResolve(sourceUrl.href, context);
}
