import { Skeleton } from "@/components/ui/skeleton";

export function ToolActivationDialogSkeleton() {
  return (
    <div className="flex flex-col gap-4">
      {Array.from({ length: 3 }, (_, groupIndex) => (
        <section className="flex flex-col gap-2" key={groupIndex}>
          <div className="flex items-center justify-between gap-2">
            <Skeleton className="h-4 w-36" />
            <Skeleton className="h-5 w-10 rounded-full" />
          </div>
          <div className="overflow-hidden rounded-md border">
            {Array.from({ length: 3 }, (_, rowIndex) => (
              <div
                className="flex items-start gap-3 border-b p-3 last:border-b-0"
                key={rowIndex}
              >
                <Skeleton className="h-[14px] w-6 rounded-full" />
                <div className="flex flex-1 flex-col gap-2">
                  <Skeleton className="h-4 w-44" />
                  <Skeleton className="h-3 w-full" />
                  <Skeleton className="h-3 w-2/3" />
                </div>
              </div>
            ))}
          </div>
        </section>
      ))}
    </div>
  );
}
