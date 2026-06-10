"use client";

import { type ReactNode, useState } from "react";
import { cn } from "@/lib/utils";

type FilterSelectProps = {
  value: string;
  onChange: (value: string) => void;
  options: { value: string; label: string }[];
  placeholder?: string;
  className?: string;
};

export function FilterSelect({
  value,
  onChange,
  options,
  placeholder = "All",
  className,
}: FilterSelectProps) {
  return (
    <select
      value={value}
      onChange={(e) => onChange(e.target.value)}
      className={cn(
        "rounded-lg border border-gray-800 bg-gray-950 px-4 py-2 text-sm text-gray-200 focus:outline-none focus:ring-2 focus:ring-emerald-500/50",
        className,
      )}
    >
      <option value="">{placeholder}</option>
      {options.map((opt) => (
        <option key={opt.value} value={opt.value}>
          {opt.label}
        </option>
      ))}
    </select>
  );
}

type SearchInputProps = {
  value: string;
  onChange: (value: string) => void;
  placeholder?: string;
  className?: string;
};

export function SearchInput({
  value,
  onChange,
  placeholder = "Search...",
  className,
}: SearchInputProps) {
  return (
    <div className={cn("relative flex-1", className)}>
      <input
        type="text"
        placeholder={placeholder}
        value={value}
        onChange={(e) => onChange(e.target.value)}
        className="w-full rounded-lg border border-gray-800 bg-gray-950 px-4 py-2 text-sm text-gray-200 placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-emerald-500/50"
      />
      {value && (
        <button
          onClick={() => onChange("")}
          className="absolute right-3 top-1/2 -translate-y-1/2 text-gray-500 hover:text-gray-300"
          type="button"
        >
          &times;
        </button>
      )}
    </div>
  );
}

type FilterBarWrapperProps = {
  children: ReactNode;
  className?: string;
};

export function FilterBar({ children, className }: FilterBarWrapperProps) {
  return (
    <div className={cn("mb-6 flex flex-wrap gap-4", className)}>
      {children}
    </div>
  );
}

FilterBar.Search = SearchInput;
FilterBar.Select = FilterSelect;
