import type { FormEvent, Ref } from "react";
import {
  ArrowLeft,
  ArrowRight,
  Globe,
  Loader2,
  RefreshCw,
  Search,
  X,
} from "lucide-react";

import { Button } from "@/components/ui/button";
import {
  InputGroup,
  InputGroupAddon,
  InputGroupButton,
  InputGroupInput,
} from "@/components/ui/input-group";
import type { BuiltinBrowserAction } from "@/features/projects/builtin-browser-model";
import { cn } from "@/lib/utils";

type BuiltinBrowserToolbarProps = {
  address: string;
  inputRef: Ref<HTMLInputElement>;
  onAddressChange: (address: string) => void;
  onClose: () => void;
  onRunAction: (action: BuiltinBrowserAction) => void;
  onSubmitAddress: (event?: FormEvent<HTMLFormElement>) => void;
  state: AivoBrowserState;
  webviewReady: boolean;
};

export function BuiltinBrowserToolbar({
  address,
  inputRef,
  onAddressChange,
  onClose,
  onRunAction,
  onSubmitAddress,
  state,
  webviewReady,
}: BuiltinBrowserToolbarProps) {
  return (
    <div className="flex shrink-0 flex-col gap-2 border-b border-border/70 p-2">
      <div className="flex min-w-0 items-center gap-1.5">
        <Button
          aria-label="后退"
          disabled={!webviewReady || !state.canGoBack}
          onClick={() => onRunAction("back")}
          size="icon-sm"
          type="button"
          variant="ghost"
        >
          <ArrowLeft />
        </Button>
        <Button
          aria-label="前进"
          disabled={!webviewReady || !state.canGoForward}
          onClick={() => onRunAction("forward")}
          size="icon-sm"
          type="button"
          variant="ghost"
        >
          <ArrowRight />
        </Button>
        <Button
          aria-label={state.loading ? "停止加载" : "刷新"}
          disabled={!webviewReady || !state.url}
          onClick={() => onRunAction(state.loading ? "stop" : "reload")}
          size="icon-sm"
          type="button"
          variant="ghost"
        >
          {state.loading ? (
            <X />
          ) : (
            <RefreshCw className={cn(state.loading && "animate-spin")} />
          )}
        </Button>
        <form className="min-w-0 flex-1" onSubmit={onSubmitAddress}>
          <InputGroup>
            <InputGroupAddon align="inline-start">
              {state.loading ? <Loader2 className="animate-spin" /> : <Globe />}
            </InputGroupAddon>
            <InputGroupInput
              aria-label="浏览器地址"
              autoCapitalize="none"
              autoCorrect="off"
              onChange={(event) => onAddressChange(event.target.value)}
              placeholder="输入网址"
              ref={inputRef}
              spellCheck={false}
              value={address}
            />
            <InputGroupAddon align="inline-end">
              <InputGroupButton aria-label="打开地址" type="submit">
                <Search />
              </InputGroupButton>
            </InputGroupAddon>
          </InputGroup>
        </form>
        <Button
          aria-label="关闭内置浏览器"
          onClick={onClose}
          size="icon-sm"
          type="button"
          variant="ghost"
        >
          <X />
        </Button>
      </div>
      {state.error ? (
        <div className="min-w-0 px-1 text-xs leading-5">
          <span className="block truncate text-destructive">{state.error}</span>
        </div>
      ) : null}
    </div>
  );
}

export function BuiltinBrowserEmptyState() {
  return (
    <div className="flex h-full min-h-0 items-center justify-center px-5 py-8">
      <div className="flex max-w-72 flex-col items-center gap-3 text-center">
        <div className="flex size-10 items-center justify-center rounded-md bg-muted text-muted-foreground">
          <Globe />
        </div>
        <div className="text-sm font-medium text-foreground">内置浏览器</div>
      </div>
    </div>
  );
}
