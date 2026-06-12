"use client"

import { type ReactNode, useState } from "react"

import { useProjects } from "@/lib/hooks"
import { useProjectFilters } from "@/hooks/useProjectFilters"
import { cn } from "@/lib/utils"
import { Spinner } from "@/components/ui/Spinner"
import { EmptyState } from "@/components/ui/EmptyState"

/**
 * Gates a page on project selection. If the user has not selected a
 * project, the children are not rendered; instead a "Select a project"
 * prompt is shown (with a list picker if /v1/projects works, or a
 * text-input fallback if it does not).
 *
 * TASK-407 pages are project-scoped per Lead (2026-06-12). The /v1/agents/*
 * routes assume X-Project-ID is present. This gate guarantees that.
 */
export function ProjectPickerGate({
  children,
  title = "Select a project",
  description = "Agent management is project-scoped. Choose a project to continue.",
}: {
  children: ReactNode
  title?: string
  description?: string
}) {
  const { projectId, setProjectId } = useProjectFilters()
  const projectsQuery = useProjects()
  const [manualId, setManualId] = useState("")

  if (projectId) {
    return <>{children}</>
  }

  function pick(id: string) {
    if (!id) return
    setProjectId(id)
  }

  return (
    <div className="mx-auto flex min-h-[60vh] max-w-2xl flex-col items-center justify-center px-4">
      <div
        className={cn(
          "w-full rounded-lg border border-slate-200 bg-white p-8 shadow-sm",
          "dark:border-slate-800 dark:bg-slate-900",
        )}
      >
        <h2 className="text-lg font-semibold text-slate-900 dark:text-slate-100">
          {title}
        </h2>
        <p className="mt-1 text-sm text-slate-500 dark:text-slate-400">
          {description}
        </p>

        {projectsQuery.isLoading ? (
          <div className="mt-6 flex items-center gap-2 text-sm text-slate-500">
            <Spinner size="sm" /> Loading available projects…
          </div>
        ) : projectsQuery.isError ? (
          <div className="mt-6 space-y-3">
            <p className="text-sm text-amber-700 dark:text-amber-400">
              Could not load the project catalog. Enter a project ID
              (UUID) to continue.
            </p>
            <form
              onSubmit={(e) => {
                e.preventDefault()
                pick(manualId.trim())
              }}
              className="flex gap-2"
            >
              <input
                value={manualId}
                onChange={(e) => setManualId(e.target.value)}
                placeholder="e.g. 8a4c8d6a-…"
                className={cn(
                  "flex-1 rounded-md border border-slate-300 bg-white px-3 py-1.5 text-sm",
                  "dark:border-slate-700 dark:bg-slate-900",
                )}
              />
              <button
                type="submit"
                className="rounded-md bg-sky-600 px-3 py-1.5 text-sm font-medium text-white hover:bg-sky-700 disabled:opacity-50"
                disabled={!manualId.trim()}
              >
                Use project
              </button>
            </form>
          </div>
        ) : (projectsQuery.data ?? []).length === 0 ? (
          <div className="mt-6">
            <EmptyState
              title="No projects yet"
              description="You don't have access to any projects. Ask a project admin to add you, or enter a project ID below."
            />
            <form
              onSubmit={(e) => {
                e.preventDefault()
                pick(manualId.trim())
              }}
              className="mt-4 flex gap-2"
            >
              <input
                value={manualId}
                onChange={(e) => setManualId(e.target.value)}
                placeholder="Enter a project ID (UUID)"
                className={cn(
                  "flex-1 rounded-md border border-slate-300 bg-white px-3 py-1.5 text-sm",
                  "dark:border-slate-700 dark:bg-slate-900",
                )}
              />
              <button
                type="submit"
                className="rounded-md bg-sky-600 px-3 py-1.5 text-sm font-medium text-white hover:bg-sky-700 disabled:opacity-50"
                disabled={!manualId.trim()}
              >
                Use project
              </button>
            </form>
          </div>
        ) : (
          <ul className="mt-6 space-y-2">
            {(projectsQuery.data ?? []).map((p) => (
              <li key={p.id}>
                <button
                  type="button"
                  onClick={() => pick(p.id)}
                  className={cn(
                    "flex w-full items-center justify-between rounded-md border border-slate-200 bg-white px-4 py-2.5 text-left text-sm transition",
                    "hover:border-sky-300 hover:bg-sky-50",
                    "dark:border-slate-800 dark:bg-slate-900 dark:hover:border-sky-700 dark:hover:bg-sky-900/20",
                  )}
                >
                  <span>
                    <span className="font-medium text-slate-900 dark:text-slate-100">
                      {p.name}
                    </span>
                    {p.description ? (
                      <span className="ml-2 text-xs text-slate-500">
                        {p.description}
                      </span>
                    ) : null}
                  </span>
                  <span className="font-mono text-xs text-slate-400">{p.id}</span>
                </button>
              </li>
            ))}
          </ul>
        )}
      </div>
    </div>
  )
}
