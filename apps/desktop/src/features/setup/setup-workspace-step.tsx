import { Button } from "@/components/ui/button";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Spinner } from "@/components/ui/spinner";
import { SetupStepNavigation } from "@/features/setup/setup-step-navigation";

export function SetupWorkspaceStep({
  error,
  onBack,
  onChoose,
  onComplete,
  path,
  saving,
}: {
  error: string;
  onBack: () => void;
  onChoose: () => Promise<void>;
  onComplete: () => Promise<void>;
  path: string;
  saving: boolean;
}) {
  return (
    <section className="relative flex min-h-dvh flex-col bg-background">
      <div className="flex flex-1 items-center justify-center px-aivo-4 py-aivo-8 sm:px-aivo-8">
        <div className="flex w-full max-w-[800px] flex-col items-center text-center">
          <h1 className="aivo-type-large-title font-bold tracking-tight text-foreground">
            设置初始化工作目录
          </h1>
          <p className="aivo-type-title-3 mt-aivo-3 max-w-[560px] text-muted-foreground">
            临时对话和未选择项目的对话都会在这里工作，并共用同一个目录
          </p>

          <div className="mt-aivo-8 flex w-full max-w-[640px] flex-col gap-aivo-3 rounded-lg border border-border bg-background p-aivo-4 text-left">
            <span className="aivo-type-headline font-semibold text-foreground">
              工作目录
            </span>
            <div className="flex min-w-0 flex-col gap-aivo-3 sm:flex-row sm:items-center">
              <div
                className="aivo-type-body min-h-aivo-control-lg min-w-0 flex-1 truncate rounded-lg bg-muted px-aivo-3 py-aivo-2 text-muted-foreground"
                title={path || undefined}
              >
                {path || "尚未选择目录"}
              </div>
              <Button
                className="h-aivo-control-lg shrink-0 rounded-lg"
                disabled={saving}
                onClick={() => void onChoose()}
                type="button"
                variant="outline"
              >
                选择目录
              </Button>
            </div>
            <p className="aivo-type-footnote text-muted-foreground">
              已提供默认目录，你可以直接使用或更换。如果目录之后被删除，Aivo
              会按原路径重新创建；不会为每个对话生成子目录。
            </p>
          </div>

          {error ? (
            <Alert className="mt-aivo-4 max-w-[640px]" variant="destructive">
              <AlertDescription>{error}</AlertDescription>
            </Alert>
          ) : null}
        </div>
      </div>

      <SetupStepNavigation
        currentStep={4}
        helperText="选择项目后，对话仍会使用对应项目目录"
        onBack={onBack}
        onPrimary={() => void onComplete()}
        primaryContent={
          <>
            {saving ? <Spinner data-icon="inline-start" /> : null}
            {saving ? "正在保存…" : "完成设置"}
          </>
        }
        primaryDisabled={!path || saving}
        totalSteps={4}
      />
    </section>
  );
}
