import { RefreshCw, Search } from "lucide-react";

import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { ScrollArea } from "@/components/ui/scroll-area";
import type { ProviderPickOption } from "@/features/projects/project-provider-picker-model";

export function ProviderPickerDialog({
  filteredProviders,
  onCatalogRefresh,
  onOpenChange,
  onProviderSelect,
  onQueryChange,
  open,
  query,
  refreshMessage,
  refreshing,
}: {
  filteredProviders: ProviderPickOption[];
  onCatalogRefresh: () => void;
  onOpenChange: (open: boolean) => void;
  onProviderSelect: (provider: ProviderPickOption) => void;
  onQueryChange: (query: string) => void;
  open: boolean;
  query: string;
  refreshMessage: string;
  refreshing: boolean;
}) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-lg">
        <div className="flex flex-col gap-4">
          <DialogHeader>
            <div className="flex items-center justify-between gap-3">
              <DialogTitle>选择提供商</DialogTitle>
              <Button
                disabled={refreshing}
                onClick={onCatalogRefresh}
                size="sm"
                type="button"
                variant="outline"
              >
                <RefreshCw className={refreshing ? "animate-spin" : ""} />
                {refreshing ? "刷新中" : "刷新目录"}
              </Button>
            </div>
          </DialogHeader>
          {refreshMessage ? (
            <p
              aria-live="polite"
              className="text-sm text-muted-foreground"
              role="status"
            >
              {refreshMessage}
            </p>
          ) : null}
          <div className="relative">
            <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              aria-label="搜索提供商"
              className="pl-9"
              onChange={(event) => onQueryChange(event.target.value)}
              placeholder="搜索 provider"
              value={query}
            />
          </div>
          <ScrollArea className="max-h-[min(50vh,24rem)] pr-2 [&_[data-slot=scroll-area-viewport]]:!h-auto [&_[data-slot=scroll-area-viewport]]:max-h-[min(50vh,24rem)]">
            <div className="grid grid-cols-1 gap-2 sm:grid-cols-2">
              {filteredProviders.map((provider) => (
                <button
                  className="flex items-center gap-2 rounded-lg border bg-background p-2 text-left transition-colors hover:bg-muted"
                  key={provider.id}
                  onClick={() => onProviderSelect(provider)}
                  type="button"
                >
                  <ProviderPickIcon provider={provider} />
                  <span className="min-w-0 truncate">{provider.name}</span>
                </button>
              ))}
            </div>
            {filteredProviders.length === 0 ? (
              <div className="px-2 py-8 text-center text-sm text-muted-foreground">
                没有可添加的 provider
              </div>
            ) : null}
          </ScrollArea>
        </div>
      </DialogContent>
    </Dialog>
  );
}

function ProviderPickIcon({ provider }: { provider: ProviderPickOption }) {
  if (provider.iconSrc) {
    return <img alt="" className="size-7 shrink-0" src={provider.iconSrc} />;
  }
  return (
    <span className="grid size-7 shrink-0 place-items-center rounded-full bg-muted text-sm font-semibold text-muted-foreground">
      {provider.name.slice(0, 1)}
    </span>
  );
}
