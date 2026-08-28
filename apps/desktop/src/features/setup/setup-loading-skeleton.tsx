import { Spinner } from "@/components/ui/spinner";

export function SetupLoadingSkeleton() {
  return (
    <main className="fixed inset-0 grid place-items-center bg-background text-foreground">
      <div className="flex flex-col items-center gap-3">
        <Spinner className="size-6 text-muted-foreground" />
        <p className="text-sm  text-muted-foreground">加载中</p>
      </div>
    </main>
  );
}
