import { Sparkles } from "lucide-react";

import { Input } from "@/components/ui/input";
import { SetupStepNavigation } from "@/features/setup/setup-step-navigation";
import {
  canSubmitAppName,
  DEFAULT_APP_NAME,
  MAX_APP_NAME_CHARACTERS,
} from "@/lib/app-identity";

export function SetupNameStep({
  name,
  onBack,
  onChange,
  onNext,
}: {
  name: string;
  onBack: () => void;
  onChange: (name: string) => void;
  onNext: () => void;
}) {
  const normalizedName = name.trim() || DEFAULT_APP_NAME;

  return (
    <section className="relative flex min-h-dvh flex-col overflow-hidden bg-background">
      <div className="flex flex-1 items-center justify-center px-aivo-4 py-aivo-8 sm:px-aivo-8">
        <div className="flex w-full max-w-[500px] flex-col items-center text-center">
          <div className="grid size-12 place-items-center rounded-2xl border border-border/80 bg-muted/60 text-foreground shadow-sm">
            <Sparkles aria-hidden="true" className="size-5" strokeWidth={1.8} />
          </div>

          <h1 className="aivo-type-large-title mt-aivo-6 font-bold tracking-tight text-foreground">
            请为我命名
          </h1>
          <p className="aivo-type-body mt-aivo-2 max-w-[420px] text-muted-foreground">
            取一个你喜欢的名字，之后我会用它陪你一起工作
          </p>

          <div className="mt-aivo-8 w-full rounded-2xl border border-border/80 bg-card p-aivo-6 text-left shadow-sm shadow-foreground/[0.03]">
            <div className="flex items-center justify-between gap-aivo-3">
              <label
                className="aivo-type-body font-semibold text-foreground"
                htmlFor="setup-app-name"
              >
                名称
              </label>
              <span className="aivo-type-footnote tabular-nums text-muted-foreground">
                {Array.from(name).length} / {MAX_APP_NAME_CHARACTERS}
              </span>
            </div>
            <Input
              autoFocus
              className="mt-aivo-2 h-12 rounded-xl bg-background px-aivo-4 text-base font-medium shadow-none"
              id="setup-app-name"
              onChange={(event) => onChange(event.target.value)}
              onKeyDown={(event) => {
                if (event.key === "Enter" && canSubmitAppName(name)) onNext();
              }}
              placeholder="Aivo"
              spellCheck={false}
              value={name}
            />

            <div className="mt-aivo-3 flex min-w-0 items-center gap-aivo-3 rounded-xl bg-muted/55 px-aivo-4 py-aivo-3">
              <span className="aivo-type-footnote shrink-0 text-muted-foreground">
                首页预览
              </span>
              <span className="aivo-type-body min-w-0 truncate font-semibold text-foreground">
                你好，我是 {normalizedName}
              </span>
            </div>
          </div>
        </div>
      </div>

      <SetupStepNavigation
        currentStep={2}
        helperText="名称保存在本机，并显示在首页和助手选择器中"
        onBack={onBack}
        onPrimary={onNext}
        primaryContent="继续"
        primaryDisabled={!canSubmitAppName(name)}
        totalSteps={4}
      />
    </section>
  );
}
