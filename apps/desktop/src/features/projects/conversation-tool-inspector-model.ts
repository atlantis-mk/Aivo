export function toolTimelineDescription(
  invocationDescription: string,
  previousInvocationDescription: string,
) {
  const description = invocationDescription.trim();
  if (
    description &&
    description !== previousInvocationDescription.trim()
  ) {
    return description;
  }
  return "";
}
