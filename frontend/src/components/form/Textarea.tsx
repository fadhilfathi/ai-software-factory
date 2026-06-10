import { forwardRef, type TextareaHTMLAttributes } from "react";
import { cn } from "@/lib/utils";
import { FormField } from "./FormField";

type TextareaProps = TextareaHTMLAttributes<HTMLTextAreaElement> & {
  label?: string;
  error?: string | null;
  hint?: string;
};

export const Textarea = forwardRef<HTMLTextAreaElement, TextareaProps>(
  ({ label, error, hint, className, id, ...props }, ref) => {
    const textareaId =
      id ?? `textarea_${label?.replace(/\s+/g, "_").toLowerCase()}`;

    const textarea = (
      <textarea
        ref={ref}
        id={textareaId}
        className={cn(
          "w-full rounded-lg border bg-gray-950 px-4 py-3 text-sm text-gray-200 placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-emerald-500/50 transition-colors",
          error
            ? "border-red-800 focus:ring-red-500/50"
            : "border-gray-800",
          className,
        )}
        {...props}
      />
    );

    if (!label && !error && !hint) return textarea;

    return (
      <FormField label={label} error={error} hint={hint} htmlFor={textareaId}>
        {textarea}
      </FormField>
    );
  },
);

Textarea.displayName = "Textarea";
