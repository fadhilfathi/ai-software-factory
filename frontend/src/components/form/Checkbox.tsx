import { type InputHTMLAttributes } from "react";
import { cn } from "@/lib/utils";

type CheckboxProps = Omit<InputHTMLAttributes<HTMLInputElement>, "type"> & {
  label?: string;
};

export function Checkbox({
  label,
  className,
  id,
  checked,
  ...props
}: CheckboxProps) {
  const checkboxId =
    id ?? `checkbox_${label?.replace(/\s+/g, "_").toLowerCase()}`;

  const input = (
    <input
      id={checkboxId}
      type="checkbox"
      checked={checked}
      className={cn(
        "h-4 w-4 rounded border-gray-700 bg-gray-800 text-emerald-500 focus:ring-emerald-500/50",
        className,
      )}
      {...props}
    />
  );

  if (!label) return input;

  return (
    <label
      htmlFor={checkboxId}
      className={cn(
        "flex items-center gap-3 rounded-lg border px-4 py-2.5 cursor-pointer transition-colors",
        checked
          ? "border-emerald-500/50 bg-emerald-500/5"
          : "border-gray-800 bg-gray-950",
      )}
    >
      {input}
      <span className="text-sm text-gray-300">{label}</span>
    </label>
  );
}
