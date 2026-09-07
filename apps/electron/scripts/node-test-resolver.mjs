export async function resolve(specifier, context, nextResolve) {
  const isAlias = specifier.startsWith("@/");
  const isRelative = specifier.startsWith("./") || specifier.startsWith("../");
  if (!isAlias && !isRelative) {
    return nextResolve(specifier, context);
  }

  const sourceUrl = isAlias
    ? new URL(`../src/${specifier.slice(2)}`, import.meta.url)
    : new URL(specifier, context.parentURL);
  if (!sourceUrl.pathname.endsWith(".ts")) {
    sourceUrl.pathname += ".ts";
  }
  return nextResolve(sourceUrl.href, context);
}
