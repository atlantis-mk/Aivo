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

export function WelcomeStep({ onNext }: { onNext: () => void }) {
  return (
    <section className="flex min-h-dvh flex-col bg-background">
      <div className="flex flex-1 items-center justify-center px-aivo-4 py-aivo-8 sm:px-aivo-8">
        <div className="flex w-full max-w-[800px] flex-col items-center text-center">
          <h1 className="aivo-type-large-title font-bold tracking-tight text-foreground">
            你好，我是 Aivo
          </h1>
          <p className="aivo-type-title-3 mt-aivo-3 text-muted-foreground">
            随时待命，帮你把事情推进
          </p>

          <div className="mt-aivo-8 flex w-full flex-col items-center gap-aivo-4">
            <h2 className="aivo-type-headline font-semibold text-foreground">
              我可以帮你完成这些事情
            </h2>
            <ul className="grid w-full max-w-[640px] grid-cols-1 gap-aivo-3 sm:grid-cols-6">
              {welcomeCapabilities.map((capability, index) => (
                <li
                  className={cn(
                    "aivo-type-body flex min-h-aivo-control-lg items-center justify-center gap-aivo-2 rounded-lg border border-border bg-background px-aivo-4 py-aivo-2 font-medium text-foreground sm:col-span-2",
                    index === 3 && "sm:col-start-2",
                  )}
                  key={capability.label}
                >
                  <HugeiconsIcon
                    aria-hidden="true"
                    className="size-4 shrink-0"
                    icon={capability.icon}
                    strokeWidth={1.8}
                  />
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
  return (
    <div
      className={cn(
        "grid w-full gap-aivo-3",
        fluid
          ? "grid-cols-[repeat(auto-fit,minmax(min(10rem,100%),1fr))]"
          : "max-w-[640px] grid-cols-1 sm:grid-cols-6",
      )}
    >
      {providerChoices.map((provider, index) => (
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
        "aivo-type-body flex min-h-aivo-control-lg min-w-0 items-center justify-center gap-aivo-2 rounded-lg border px-aivo-4 py-aivo-2 font-medium transition-colors",
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
