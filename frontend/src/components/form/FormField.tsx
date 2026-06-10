import type { ReactNode } from "react";
import { cn } from "@/lib/utils";

type FormFieldProps = {
  label?: string;
  error?: string | null;
  hint?: string;
  children: ReactNode;
  className?: string;
  required?: boolean;
  htmlFor?: string;
};

export function FormField({
  label,
  error,
  hint,
  children,
  className,
  required = false,
}: FormFieldProps) {
  return (
    <div className={cn("space-y-1", className)}>
      {label && (
        <label className="block text-sm font-medium text-gray-300">
          {label}
          {required && <span className="ml-0.5 text-red-400">*</span>}
        </label>
      )}
      {children}
      {error && <p className="text-xs text-red-400">{error}</p>}
      {hint && !error && (
        <p className="text-xs text-gray-500">{hint}</p>
      )}
    </div>
  );
}
