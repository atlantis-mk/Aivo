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
    <footer className="flex w-full shrink-0 flex-col items-center gap-aivo-3 px-aivo-4 pb-aivo-8 sm:px-aivo-8">
      <div className="flex items-center gap-aivo-3">
        <Button
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
          className="aivo-type-body w-[134px] rounded-lg"
          disabled={primaryDisabled}
          onClick={onPrimary}
          size="lg"
          type="button"
        >
          {primaryContent}
        </Button>
      </div>

      <p className="aivo-type-footnote text-center text-muted-foreground">
        {helperText}
      </p>

      <p className="aivo-type-footnote mt-aivo-3 flex items-center gap-aivo-2 text-muted-foreground">
        {Array.from({ length: totalSteps }, (_, index) => index + 1).map((step) => (
          <span
            aria-hidden="true"
            className={
              step === currentStep
                ? "size-1.5 rounded-full bg-foreground"
                : "size-1.5 rounded-full bg-muted"
            }
            key={step}
          />
        ))}
        <span className="ml-aivo-2">{currentStep} / {totalSteps}</span>
      </p>
    </footer>
  );
}
