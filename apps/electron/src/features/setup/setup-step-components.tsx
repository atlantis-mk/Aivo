import { Cancel01Icon } from "@hugeicons/core-free-icons";
import { HugeiconsIcon } from "@hugeicons/react";

import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { ScrollArea } from "@/components/ui/scroll-area";
import { ProviderIcon } from "@/features/providers/provider-icon";
import type { ProviderChoice } from "@/features/providers/provider-types";
import {
  otherProviderChoices,
  providerChoices,
  welcomeCapabilities,
} from "@/features/setup/setup-provider-options";
import { SetupStepNavigation } from "@/features/setup/setup-step-navigation";
import { cn } from "@/lib/utils";
import { hasCodexDesktopBridge } from "@/lib/app-config";

const volcengineProviderIds = new Set([
  "volcengine-agent-plan",
  "volcengine-coding-plan",
  "volcengine",
]);

export function WelcomeStep({ onNext }: { onNext: () => void }) {
  return (
    <section className="relative flex min-h-dvh flex-col overflow-hidden bg-background">
      <div
        aria-hidden="true"
        className="pointer-events-none absolute inset-x-0 -top-12 mx-auto h-64 w-[32rem] rounded-full bg-[radial-gradient(closest-side,var(--color-muted),transparent)] opacity-90 blur-2xl"
      />
      <div
        aria-hidden="true"
        className="pointer-events-none absolute -left-12 top-24 size-44 rounded-full bg-[radial-gradient(closest-side,color-mix(in_oklch,var(--color-primary)_12%,transparent),transparent)] blur-3xl"
      />
      <div
        aria-hidden="true"
        className="pointer-events-none absolute -right-16 top-28 size-48 rounded-full bg-[radial-gradient(closest-side,color-mix(in_oklch,var(--color-foreground)_8%,transparent),transparent)] blur-3xl"
      />

      <div className="relative flex flex-1 items-center justify-center px-4 py-8 sm:px-8">
        <div className="flex w-full max-w-[680px] flex-col items-center text-center">
          <h1 className="text-xl font-bold tracking-tight text-foreground sm:text-2xl">
            你好，我是 Aivo
          </h1>
          <p className="mt-2 max-w-sm text-sm text-balance text-muted-foreground">
            随时待命，帮你把事情推进
          </p>

          <div className="mt-6 flex w-full flex-col items-center gap-3">
            <h2 className="text-xs font-medium tracking-wide text-muted-foreground">
              我可以帮你完成这些事情
            </h2>
            <ul className="grid w-full max-w-[560px] grid-cols-1 gap-2.5 sm:grid-cols-6">
              {welcomeCapabilities.map((capability, index) => (
                <li
                  className={cn(
                    "group flex min-h-10 items-center justify-center gap-2 rounded-xl border border-border/70 bg-card/70 px-3 py-1.5 text-sm font-medium text-foreground shadow-xs backdrop-blur-sm transition-all duration-200 hover:-translate-y-0.5 hover:border-border hover:bg-card hover:shadow-sm sm:col-span-2",
                    index === 3 && "sm:col-start-2",
                  )}
                  key={capability.label}
                >
                  <span className="flex size-6 shrink-0 items-center justify-center rounded-lg bg-muted/70 text-muted-foreground transition-colors duration-200 group-hover:bg-primary/10 group-hover:text-foreground">
                    <HugeiconsIcon
                      aria-hidden="true"
                      className="size-3.5"
                      icon={capability.icon}
                      strokeWidth={1.8}
                    />
                  </span>
                  <span>{capability.label}</span>
                </li>
              ))}
            </ul>
          </div>
        </div>
      </div>

      <SetupStepNavigation
        currentStep={1}
        helperText="敏感操作会先征得你的确认"
        onPrimary={onNext}
        primaryContent="开始设置"
        totalSteps={4}
      />
    </section>
  );
}

export function ProviderChoiceGrid({
  activeProviderId,
  fluid = false,
  onProviderClick,
}: {
  activeProviderId?: string;
  fluid?: boolean;
  onProviderClick: (provider: ProviderChoice) => void;
}) {
  const visibleProviders = hasCodexDesktopBridge()
    ? providerChoices.filter(
        (provider) =>
          provider.id === "openai" || volcengineProviderIds.has(provider.id),
      )
    : providerChoices;

  return (
    <div
      className={cn(
        "grid w-full gap-3",
        fluid
          ? "grid-cols-[repeat(auto-fit,minmax(min(10rem,100%),1fr))]"
          : "max-w-[640px] grid-cols-1 sm:grid-cols-6",
      )}
    >
      {visibleProviders.map((provider, index) => (
        <ProviderChoiceCard
          key={provider.id}
          active={activeProviderId === provider.id}
          centered={index === 3}
          fluid={fluid}
          onClick={() => onProviderClick(provider)}
          provider={provider}
        />
      ))}
    </div>
  );
}

export function OtherProviderPickerDialog({
  onOpenChange,
  onSearchChange,
  onSelect,
  open,
  search,
}: {
  onOpenChange: (open: boolean) => void;
  onSearchChange: (search: string) => void;
  onSelect: (provider: ProviderChoice) => void;
  open: boolean;
  search: string;
}) {
  const normalizedSearch = search.trim().toLowerCase();
  const filteredProviders = normalizedSearch
    ? otherProviderChoices.filter((provider) => {
        return (
          provider.name.toLowerCase().includes(normalizedSearch) ||
          provider.id.toLowerCase().includes(normalizedSearch)
        );
      })
    : otherProviderChoices;

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-lg" showCloseButton={false}>
        <div className="flex flex-col gap-4">
          <div className="flex items-center justify-between gap-3">
            <DialogTitle>选择提供商</DialogTitle>
            <DialogClose asChild>
              <Button aria-label="关闭" size="icon" variant="ghost">
                <HugeiconsIcon icon={Cancel01Icon} strokeWidth={2} />
              </Button>
            </DialogClose>
          </div>

          <Input
            aria-label="搜索提供商"
            onChange={(event) => onSearchChange(event.target.value)}
            placeholder="搜索 provider"
            value={search}
          />

          <ScrollArea className="max-h-[min(52vh,420px)] pr-3">
            <div className="grid grid-cols-1 gap-2 sm:grid-cols-2">
              {filteredProviders.map((provider) => (
                <button
                  className="flex items-center gap-2 rounded-lg border bg-background px-3 py-2 text-left text-sm  transition-colors hover:bg-muted"
                  key={provider.id}
                  onClick={() => onSelect(provider)}
                  type="button"
                >
                  <ProviderIcon provider={provider} size="sm" />
                  <span className="min-w-0 truncate">{provider.name}</span>
                </button>
              ))}
            </div>
          </ScrollArea>

          {filteredProviders.length === 0 ? (
            <div className="rounded-lg border border-dashed py-8 text-center text-sm text-muted-foreground">
              没有匹配的提供商
            </div>
          ) : null}
        </div>
      </DialogContent>
    </Dialog>
  );
}

function ProviderChoiceCard({
  active,
  centered,
  fluid,
  onClick,
  provider,
}: {
  active: boolean;
  centered: boolean;
  fluid: boolean;
  onClick: () => void;
  provider: ProviderChoice;
}) {
  return (
    <button
      aria-pressed={active}
      className={cn(
        "flex min-h-12 min-w-0 items-center justify-center gap-2 rounded-lg border px-4 py-2 font-medium transition-colors",
        "hover:bg-muted/60 focus-visible:border-ring focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/30",
        !fluid && "sm:col-span-2",
        centered && !fluid && "sm:col-start-2",
        active
          ? "border-foreground bg-muted text-foreground"
          : "border-border bg-background text-foreground",
      )}
      onClick={onClick}
      type="button"
    >
      <ProviderIcon provider={provider} size="sm" />
      <span className="min-w-0 truncate">{provider.name}</span>
    </button>
  );
}
