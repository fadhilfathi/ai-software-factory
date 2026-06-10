import { cn } from "@/lib/utils";

type StepIndicatorProps = {
  steps: { num: number; label: string }[];
  currentStep: number;
  className?: string;
};

export function StepIndicator({
  steps,
  currentStep,
  className,
}: StepIndicatorProps) {
  return (
    <div
      className={cn(
        "mb-8 flex items-center justify-center gap-2",
        className,
      )}
    >
      {steps.map((s, i) => (
        <div key={s.num} className="flex items-center gap-2">
          <div
            className={cn(
              "flex h-8 w-8 items-center justify-center rounded-full text-sm font-bold transition-colors",
              s.num <= currentStep
                ? "bg-emerald-500 text-white"
                : "bg-gray-800 text-gray-500",
            )}
          >
            {s.num < currentStep ? "✓" : s.num}
          </div>
          <span
            className={cn(
              "hidden text-xs sm:inline",
              s.num <= currentStep ? "text-gray-300" : "text-gray-600",
            )}
          >
            {s.label}
          </span>
          {i < steps.length - 1 && (
            <div
              className={cn(
                "h-0.5 w-8 sm:w-12 transition-colors",
                s.num < currentStep ? "bg-emerald-500" : "bg-gray-800",
              )}
            />
          )}
        </div>
      ))}
    </div>
  );
}
