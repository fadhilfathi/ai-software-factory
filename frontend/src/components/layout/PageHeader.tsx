"use client";

import type { ReactNode } from "react";

type PageHeaderProps = {
  title: string;
  subtitle?: string | ReactNode;
  /**
   * Alias of `subtitle` for the Sprint 4 page styles that prefer the
   * "description" naming. If both are given, `subtitle` wins.
   */
  description?: string | ReactNode;
  actions?: ReactNode;
  /**
   * Optional content rendered between the title and the actions row.
   * Used by the agent detail page to show the status badge + role
   * inline with the heading. When `children` is present the page
   * header uses a stacked layout (title on top, children below, then
   * actions at the very bottom).
   */
  children?: ReactNode;
};

export function PageHeader({ title, subtitle, description, actions, children }: PageHeaderProps) {
  const text = subtitle ?? description;
  return (
    <div className="mb-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-100">{title}</h1>
          {text && <p className="mt-1 text-sm text-gray-400">{text}</p>}
        </div>
        {actions && <div className="flex items-center gap-3">{actions}</div>}
      </div>
      {children ? <div className="mt-3">{children}</div> : null}
    </div>
  );
}
