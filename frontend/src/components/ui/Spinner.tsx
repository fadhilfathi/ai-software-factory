import { cn } from "@/lib/utils";

type SpinnerProps = {
  size?: "sm" | "md" | "lg";
  className?: string;
};

const SIZE_CLASSES = {
  sm: "h-4 w-4",
  md: "h-5 w-5",
  lg: "h-8 w-8",
};

export function Spinner({ size = "md", className }: SpinnerProps) {
  return (
    <svg
      className={cn("animate-spin", SIZE_CLASSES[size], className)}
      viewBox="0 0 24 24"
      fill="none"
    >
      <circle
        className="opacity-25"
        cx="12"
        cy="12"
        r="10"
        stroke="currentColor"
        strokeWidth="4"
      />
      <path
        className="opacity-75"
        fill="currentColor"
        d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"
      />
    </svg>
  );
}

type SpinnerButtonProps = {
  children: React.ReactNode;
  loading?: boolean;
  loadingText?: string;
} & React.ButtonHTMLAttributes<HTMLButtonElement>;

export function SpinnerButton({
  children,
  loading = false,
  loadingText,
  disabled,
  className,
  ...props
}: SpinnerButtonProps) {
  return (
    <button
      disabled={disabled || loading}
      className={cn(
        "inline-flex items-center gap-2 rounded-lg bg-emerald-500 px-4 py-2 text-sm font-medium text-white hover:bg-emerald-600 transition-colors disabled:opacity-50 disabled:cursor-not-allowed",
        className,
      )}
      type="button"
      {...props}
    >
      {loading && <Spinner size="sm" />}
      {loading ? loadingText ?? children : children}
    </button>
  );
}
