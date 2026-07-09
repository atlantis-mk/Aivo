import { cn } from "@/lib/utils";
import type { ProviderChoice } from "@/features/providers/provider-types";

export function ProviderIcon({
  provider,
  size,
}: {
  provider: ProviderChoice;
  size: "sm" | "lg";
}) {
  const wrapperClassName = size === "lg" ? "size-11" : "size-5 shrink-0";
  const imageClassName = size === "lg" ? "size-7" : "size-4";

  if (provider.iconSrc) {
    return (
      <span
        className={cn(
          "grid place-items-center rounded-full bg-card text-foreground",
          wrapperClassName,
        )}
      >
        <img alt="" className={imageClassName} src={provider.iconSrc} />
      </span>
    );
  }

  return (
    <span
      className={cn(
        "rounded-full bg-primary",
        wrapperClassName,
        provider.iconClassName,
      )}
    />
  );
}
