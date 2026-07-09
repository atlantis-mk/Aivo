import { X } from "lucide-react";

import { Badge } from "@/components/ui/badge";
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
  capabilityPills,
  otherProviderChoices,
  providerChoices,
} from "@/features/setup/setup-provider-options";
import { cn } from "@/lib/utils";

export function WelcomeStep({ onNext }: { onNext: () => void }) {
  return (
    <section className="flex min-h-dvh items-center justify-center bg-background px-5 py-16">
      <div className="flex w-full max-w-[1100px] flex-col items-center text-center">
        <h1 className="text-3xl font-extrabold leading-9 tracking-normal text-foreground sm:text-4xl sm:leading-10">
          你好，我是 Aivo
        </h1>
        <p className="mt-6 text-xl leading-7 text-foreground">
          为你 24 小时随时在线
        </p>

        <div className="mt-12 flex flex-wrap items-center justify-center gap-x-5 gap-y-3 sm:mt-16 min-[1180px]:flex-nowrap">
          {capabilityPills.map((pill) => (
            <Badge key={pill} className="h-9 px-5 text-sm" variant="secondary">
              {pill}
            </Badge>
          ))}
        </div>

        <Button
          className="mt-16 h-12 rounded-full px-8 text-base sm:mt-24"
          onClick={onNext}
          size="lg"
        >
          下一步
        </Button>
      </div>
    </section>
  );
}

export function ProviderChoiceGrid({
  activeProviderId,
  onProviderClick,
}: {
  activeProviderId?: string;
  onProviderClick: (provider: ProviderChoice) => void;
}) {
  return (
    <div className="grid w-full max-w-[880px] grid-cols-2 gap-3 sm:grid-cols-3 sm:gap-4 min-[980px]:grid-cols-5">
      {providerChoices.map((provider) => (
        <ProviderChoiceCard
          key={provider.id}
          active={activeProviderId === provider.id}
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
                <X />
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
  onClick,
  provider,
}: {
  active: boolean;
  onClick: () => void;
  provider: ProviderChoice;
}) {
  return (
    <button
      className={cn(
        "flex h-28 min-w-0 flex-col items-center justify-center gap-3 rounded-2xl border p-4 text-center",
        active ? "border-primary bg-accent" : "border-border bg-card",
      )}
      onClick={onClick}
      type="button"
    >
      <ProviderIcon provider={provider} size="lg" />
      <span className="w-full min-w-0 truncate text-sm font-bold leading-4 text-foreground">
        {provider.name}
      </span>
    </button>
  );
}
