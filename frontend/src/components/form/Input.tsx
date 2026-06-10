import { forwardRef, type InputHTMLAttributes } from "react";
import { cn } from "@/lib/utils";
import { FormField } from "./FormField";

type InputProps = InputHTMLAttributes<HTMLInputElement> & {
  label?: string;
  error?: string | null;
  hint?: string;
};

export const Input = forwardRef<HTMLInputElement, InputProps>(
  ({ label, error, hint, className, id, ...props }, ref) => {
    const inputId = id ?? `input_${label?.replace(/\s+/g, "_").toLowerCase()}`;

    const input = (
      <input
        ref={ref}
        id={inputId}
        className={cn(
          "w-full rounded-lg border bg-gray-950 px-4 py-2 text-sm text-gray-200 placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-emerald-500/50 transition-colors",
          error
            ? "border-red-800 focus:ring-red-500/50"
            : "border-gray-800",
          className,
        )}
        {...props}
      />
    );

    if (!label && !error && !hint) return input;

    return (
      <FormField label={label} error={error} hint={hint} htmlFor={inputId}>
        {input}
      </FormField>
    );
  },
);

Input.displayName = "Input";
