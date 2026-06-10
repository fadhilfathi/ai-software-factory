import { cn } from "@/lib/utils";

type ToggleProps = {
  checked: boolean;
  onChange: (checked: boolean) => void;
  label?: string;
  description?: string;
  disabled?: boolean;
  id?: string;
};

export function Toggle({
  checked,
  onChange,
  label,
  description,
  disabled = false,
  id,
}: ToggleProps) {
  const toggleId = id ?? `toggle_${label?.replace(/\s+/g, "_").toLowerCase()}`;

  const content = (
    <button
      id={toggleId}
      type="button"
      role="switch"
      aria-checked={checked}
      disabled={disabled}
      onClick={() => onChange(!checked)}
      className={cn(
        "relative inline-flex h-6 w-11 shrink-0 cursor-pointer items-center rounded-full transition-colors focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-emerald-500",
        checked ? "bg-emerald-500" : "bg-gray-700",
        disabled && "opacity-50 cursor-not-allowed",
      )}
    >
      <span
        className={cn(
          "inline-block h-4 w-4 rounded-full bg-white transition-transform",
          checked ? "translate-x-[1.375rem]" : "translate-x-[3px]",
        )}
      />
    </button>
  );

  if (!label && !description) return content;

  return (
    <label
      htmlFor={toggleId}
      className="flex items-center justify-between gap-4 cursor-pointer"
    >
      <div>
        {label && <p className="text-sm text-gray-200">{label}</p>}
        {description && (
          <p className="text-xs text-gray-500">{description}</p>
        )}
      </div>
      {content}
    </label>
  );
}
