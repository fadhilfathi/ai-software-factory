import { forwardRef, type SelectHTMLAttributes } from "react";
import { cn } from "@/lib/utils";
import { FormField } from "./FormField";

type SelectOption = {
  value: string;
  label: string;
};

type SelectProps = SelectHTMLAttributes<HTMLSelectElement> & {
  label?: string;
  error?: string | null;
  hint?: string;
  options: SelectOption[];
  placeholder?: string;
};

export const Select = forwardRef<HTMLSelectElement, SelectProps>(
  ({ label, error, hint, options, placeholder, className, id, ...props }, ref) => {
    const selectId =
      id ?? `select_${label?.replace(/\s+/g, "_").toLowerCase()}`;

    const select = (
      <select
        ref={ref}
        id={selectId}
        className={cn(
          "w-full rounded-lg border bg-gray-950 px-4 py-2 text-sm text-gray-200 focus:outline-none focus:ring-2 focus:ring-emerald-500/50 transition-colors",
          error
            ? "border-red-800 focus:ring-red-500/50"
            : "border-gray-800",
          className,
        )}
        {...props}
      >
        {placeholder && (
          <option value="">{placeholder}</option>
        )}
        {options.map((opt) => (
          <option key={opt.value} value={opt.value}>
            {opt.label}
          </option>
        ))}
      </select>
    );

    if (!label && !error && !hint) return select;

    return (
      <FormField label={label} error={error} hint={hint} htmlFor={selectId}>
        {select}
      </FormField>
    );
  },
);

Select.displayName = "Select";
