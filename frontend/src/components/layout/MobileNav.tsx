"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { cn } from "@/lib/utils";

const mobileTabs = [
  { href: "/dashboard", label: "Dashboard", icon: "▦" },
  { href: "/projects", label: "Projects", icon: "▣" },
  { href: "/tasks", label: "Tasks", icon: "☰" },
  { href: "/agents", label: "Agents", icon: "◎" },
  { href: "/settings", label: "Settings", icon: "⚙" },
];

export function MobileNav() {
  const pathname = usePathname();

  return (
    <nav
      className="fixed bottom-0 left-0 right-0 z-40 flex h-16 items-center justify-around border-t border-gray-800 bg-gray-950 md:hidden"
      role="navigation"
      aria-label="Mobile navigation"
    >
      {mobileTabs.map((tab) => {
        const isActive = pathname.startsWith(tab.href);
        return (
          <Link
            key={tab.href}
            href={tab.href}
            className={cn(
              "flex flex-col items-center gap-0.5 px-3 py-1 text-xs font-medium transition-colors",
              isActive ? "text-emerald-400" : "text-gray-500 hover:text-gray-300",
            )}
            aria-current={isActive ? "page" : undefined}
          >
            <span className="text-lg">{tab.icon}</span>
            <span>{tab.label}</span>
          </Link>
        );
      })}
    </nav>
  );
}
