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
      <div className="flex flex-1 items-center justify-center px-4 py-8 sm:px-8">
        <div className="flex w-full max-w-[520px] flex-col items-center text-center">
          <h1 className="font-bold tracking-tight text-foreground">
            设置初始化工作目录
          </h1>
          <p className="mt-3 max-w-[560px] text-muted-foreground">
            临时对话和未选择项目的对话都会在这里工作，并共用同一个目录
          </p>

          <div className="mt-8 flex w-full flex-col gap-3 rounded-lg border border-border bg-background p-4 text-left">
            <span className="font-semibold text-foreground">工作目录</span>
            <div className="flex min-w-0 flex-col gap-3 sm:flex-row sm:items-center">
              <div
                className="min-w-0 flex-1 truncate rounded-lg bg-muted px-3 py-2 text-sm text-muted-foreground"
                title={path || undefined}
              >
                {path || "尚未选择目录"}
              </div>
              <Button
                className="shrink-0"
                disabled={saving}
                onClick={() => void onChoose()}
                size="sm"
                type="button"
                variant="outline"
              >
                选择目录
              </Button>
            </div>
            <p className="text-muted-foreground">
              已提供默认目录，你可以直接使用或更换。如果目录之后被删除，Aivo
              会按原路径重新创建；不会为每个对话生成子目录。
            </p>
          </div>

          {error ? (
            <Alert className="mt-4 w-full" variant="destructive">
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
