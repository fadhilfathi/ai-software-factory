"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { Fragment } from "react";

// Route → breadcrumb mapping
const routeNames: Record<string, string> = {
  dashboard: "Dashboard",
  projects: "Projects",
  tasks: "Tasks",
  agents: "Agents",
  settings: "Settings",
  new: "New",
};

export function Breadcrumb() {
  const pathname = usePathname();
  const segments = pathname.split("/").filter(Boolean);

  if (segments.length === 0) return null;

  const crumbs = segments.map((segment, i) => {
    const href = "/" + segments.slice(0, i + 1).join("/");
    const label = routeNames[segment] || decodeURIComponent(segment);
    const isLast = i === segments.length - 1;
    return { href, label, isLast };
  });

  return (
    <nav aria-label="Breadcrumb" className="mb-4">
      <ol className="flex items-center gap-2 text-sm text-gray-400">
        <li>
          <Link href="/dashboard" className="hover:text-gray-200 transition-colors">
            Home
          </Link>
        </li>
        {crumbs.map((crumb, i) => (
          <Fragment key={crumb.href}>
            <span className="text-gray-600">/</span>
            <li>
              {crumb.isLast ? (
                <span className="text-gray-200 font-medium" aria-current="page">
                  {crumb.label}
                </span>
              ) : (
                <Link href={crumb.href} className="hover:text-gray-200 transition-colors">
                  {crumb.label}
                </Link>
              )}
            </li>
          </Fragment>
        ))}
      </ol>
    </nav>
  );
}
