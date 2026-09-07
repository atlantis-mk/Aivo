import { ArrowLeft } from "lucide-react";
import type { ReactNode } from "react";

import { Button } from "@/components/ui/button";

export function SetupStepNavigation({
  currentStep,
  helperText,
  onBack,
  onPrimary,
  primaryContent,
  primaryDisabled = false,
  totalSteps = 3,
}: {
  currentStep: number;
  helperText: ReactNode;
  onBack?: () => void;
  onPrimary: () => void;
  primaryContent: ReactNode;
  primaryDisabled?: boolean;
  totalSteps?: number;
}) {
  return (
    <footer className="relative flex w-full shrink-0 flex-col items-center px-4 pb-9 sm:px-8">
      <div className="flex items-center gap-3">
        <Button
          className="border-border/70 shadow-xs backdrop-blur-sm"
          aria-label="返回"
          disabled={!onBack}
          onClick={onBack}
          size="icon-lg"
          title="返回"
          type="button"
          variant="outline"
        >
          <ArrowLeft aria-hidden="true" />
        </Button>

        <Button
          className="w-[134px] font-semibold shadow-lg shadow-foreground/10"
          disabled={primaryDisabled}
          onClick={onPrimary}
          size="lg"
          type="button"
        >
          {primaryContent}
        </Button>
      </div>

      <p className="mt-4 text-center text-sm text-muted-foreground">
        {helperText}
      </p>

      <p className="mt-3 flex items-center gap-2.5 text-sm text-muted-foreground">
        {Array.from({ length: totalSteps }, (_, index) => index + 1).map((step) => (
          <span
            aria-hidden="true"
            className={
              step === currentStep
                ? "h-1.5 w-5 rounded-full bg-foreground"
                : "size-1.5 rounded-full bg-muted-foreground/25 transition-colors duration-200"
            }
            key={step}
          />
        ))}
        <span className="ml-1 tabular-nums">{currentStep} / {totalSteps}</span>
      </p>
    </footer>
  );
}
