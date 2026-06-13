"use client"

import { type FormEvent, use, useEffect, useState } from "react"
import Link from "next/link"
import { useRouter } from "next/navigation"

import { ProjectPickerGate } from "@/components/agents/ProjectPickerGate"
import { AgentPicker } from "@/components/agents/AgentPicker"
import { CapabilityChip } from "@/components/agents/CapabilityChip"
import { ErrorBlock } from "@/components/ui/ErrorBlock"
import { Spinner } from "@/components/ui/Spinner"
import { useAssignTask, useTask, useUpdateTaskStatus } from "@/lib/hooks"
import { cn } from "@/lib/utils"

type Params = Promise<{ id: string }>

/**
 * /tasks/:id/assign — focused single-task assignment UI.
 *
 * Three-step form:
 *   1. Pick the target status (defaults to the task's current status;
 *      if the task is in backlog/ready/blocked, the picker offers the
 *      documented transitions per docs/api-spec.md).
 *   2. Pick the agent (capability-filtered by the task's
 *      `capabilities_required` so the picker is meaningful).
 *   3. Optional notes and a per-task `capabilities_required` override.
 *
 * On submit, the page calls POST /v1/tasks/:id/assign (per Lead's
 * brief: body shape `{agent_id, capabilities_required?, notes?}` —
 * which differs from sprint4 spec §3.1 `{agent_id?, strategy?, reason?}`).
 * On success, the user is redirected to /tasks/:id/ownership so the
 * new event is immediately visible in the timeline.
 */
export default function AssignTaskPage({ params }: { params: Params }) {
  return (
    <ProjectPickerGate
      title="Select a project"
      description="Task assignment is project-scoped. Choose a project to continue."
    >
      <AssignTaskPageInner params={params} />
    </ProjectPickerGate>
  )
}

function AssignTaskPageInner({ params }: { params: Params }) {
  const { id } = use(params)
  const router = useRouter()
  const taskQuery = useTask(id)
  const assignTask = useAssignTask()
  const updateStatus = useUpdateTaskStatus()

  // Form state
  const [agentId, setAgentId] = useState<string | null>(null)
  const [notes, setNotes] = useState("")
  const [overrideCapabilities, setOverrideCapabilities] = useState<string[]>([])
  const [capabilityInput, setCapabilityInput] = useState("")
  const [submitError, setSubmitError] = useState<string | null>(null)

  // Pre-fill the override list with the task's existing required_capabilities
  // (so the user can either keep them or strip a few). This seeds the
  // form the first time the task data arrives. The Type falls back to
  // `capabilities_required` (Sprint 4 spec) to stay compatible with
  // whichever backend is wired up.
  const taskCapabilities =
    taskQuery.data?.required_capabilities ??
    (taskQuery.data as { capabilities_required?: string[] } | undefined)
      ?.capabilities_required ??
    []
  useEffect(() => {
    setOverrideCapabilities((prev) =>
      prev.length === 0 && taskCapabilities.length > 0
        ? [...taskCapabilities]
        : prev,
    )
  }, [taskCapabilities])

  // Auto-promote backlog/ready tasks to "in_progress" once an agent is picked
  // (matches the documented state machine: backlog/ready → in_progress requires
  // an assignment). The mutation is debounced — we don't want to fire it
  // before the user clicks the main submit button. We surface the status
  // change as a non-blocking toast/state.
  const currentStatus = taskQuery.data?.status ?? "ready"
  const needsStatusBump = currentStatus === "backlog" || currentStatus === "ready"

  function addCapability(raw: string) {
    const trimmed = raw.trim()
    if (!trimmed) return
    setOverrideCapabilities((prev) =>
      prev.includes(trimmed) ? prev : [...prev, trimmed],
    )
    setCapabilityInput("")
  }

  function removeCapability(cap: string) {
    setOverrideCapabilities((prev) => prev.filter((c) => c !== cap))
  }

  async function onSubmit(e: FormEvent) {
    e.preventDefault()
    setSubmitError(null)
    if (!agentId) {
      setSubmitError("Pick an agent before assigning.")
      return
    }
    try {
      // If the task is sitting in backlog/ready, bump it to in_progress
      // first so the assignment is consistent with the state machine.
      if (needsStatusBump) {
        await updateStatus.mutateAsync({ id, status: "in_progress" })
      }
      await assignTask.mutateAsync({
        taskId: id,
        agent_id: agentId,
        capabilities_required:
          // Only send the override if it differs from the task default.
          // (Equal lists are sent to keep the request explicit; the
          // backend is permissive about redundant fields.)
          JSON.stringify([...overrideCapabilities].sort()) !==
          JSON.stringify([...taskCapabilities].sort())
            ? overrideCapabilities
            : undefined,
        notes: notes.trim() || undefined,
      })
      router.push(`/tasks/${id}/ownership`)
    } catch (err) {
      setSubmitError(
        err instanceof Error ? err.message : "Assignment failed. Please retry.",
      )
    }
  }

  if (taskQuery.isLoading) {
    return (
      <div className="flex items-center gap-2 px-6 py-12 text-sm text-slate-500">
        <Spinner size="sm" /> Loading task…
      </div>
    )
  }
  if (taskQuery.isError || !taskQuery.data) {
    return (
      <div className="mx-auto max-w-3xl px-6 py-12">
        <ErrorBlock
          title="Task not found"
          message={
            (taskQuery.error as Error | undefined)?.message ??
            "Could not load the task. It may have been deleted or you may not have access."
          }
          actions={
            <Link
              href="/tasks"
              className="rounded-md border border-slate-300 px-3 py-1.5 text-sm hover:bg-slate-100 dark:border-slate-700 dark:hover:bg-slate-800"
            >
              Back to tasks
            </Link>
          }
        />
      </div>
    )
  }

  const task = taskQuery.data
  const submitting = assignTask.isPending || updateStatus.isPending

  return (
    <div className="mx-auto max-w-3xl px-6 py-8">
      <nav className="mb-2 flex items-center gap-2 text-sm text-slate-500">
        <Link href="/tasks" className="hover:text-slate-700 dark:hover:text-slate-300">
          Tasks
        </Link>
        <span>/</span>
        <Link
          href={`/tasks/${task.id}`}
          className="hover:text-slate-700 dark:hover:text-slate-300"
        >
          {task.id.slice(0, 8)}…
        </Link>
        <span>/</span>
        <span className="text-slate-700 dark:text-slate-300">Assign</span>
      </nav>

      <header className="mb-6">
        <h1 className="text-2xl font-semibold text-slate-900 dark:text-slate-100">
          Assign task
        </h1>
        <p className="mt-1 text-sm text-slate-500 dark:text-slate-400">
          {task.title}
        </p>
        <div className="mt-3 flex flex-wrap gap-2 text-xs text-slate-500">
          <span className="rounded-md bg-slate-100 px-2 py-0.5 dark:bg-slate-800">
            {task.status}
          </span>
          <span className="rounded-md bg-slate-100 px-2 py-0.5 dark:bg-slate-800">
            {task.priority}
          </span>
          {((task as { due_date?: string | null }).due_date) ? (
            <span className="rounded-md bg-slate-100 px-2 py-0.5 dark:bg-slate-800">
              due {new Date((task as { due_date?: string }).due_date!).toLocaleDateString()}
            </span>
          ) : null}
        </div>
      </header>

      <form onSubmit={onSubmit} className="space-y-6">
        {/* Step 1: status bump notice */}
        {needsStatusBump ? (
          <div
            className={cn(
              "rounded-md border border-sky-200 bg-sky-50 p-3 text-sm text-sky-800",
              "dark:border-sky-800 dark:bg-sky-900/30 dark:text-sky-200",
            )}
          >
            This task is currently <strong>{currentStatus}</strong>. Assigning an
            agent will also move it to <strong>in_progress</strong> per the
            documented state machine.
          </div>
        ) : null}

        {/* Step 2: agent picker */}
        <section>
          <label className="mb-2 block text-sm font-medium text-slate-700 dark:text-slate-300">
            Agent
          </label>
          <AgentPicker
            value={agentId}
            onChange={setAgentId}
            requiredCapabilities={taskCapabilities}
            emptyHint="No idle agents match the required capabilities."
            statusFilter={["idle", "initializing"]}
          />
          <p className="mt-1.5 text-xs text-slate-500">
            Picker is pre-filtered by the task&apos;s required capabilities. You
            can override the list below.
          </p>
        </section>

        {/* Step 3: capabilities_required override */}
        <section>
          <label className="mb-2 block text-sm font-medium text-slate-700 dark:text-slate-300">
            Required capabilities (override)
          </label>
          {overrideCapabilities.length > 0 ? (
            <div className="mb-2 flex flex-wrap gap-1.5">
              {overrideCapabilities.map((c) => (
                <span
                  key={c}
                  className="inline-flex items-center gap-1 rounded-md bg-slate-100 pl-2 pr-1 py-0.5 text-xs dark:bg-slate-800"
                >
                  <CapabilityChip name={c} />
                  <button
                    type="button"
                    onClick={() => removeCapability(c)}
                    aria-label={`Remove ${c}`}
                    className="rounded p-0.5 text-slate-400 hover:bg-slate-200 hover:text-slate-700 dark:hover:bg-slate-700 dark:hover:text-slate-200"
                  >
                    ✕
                  </button>
                </span>
              ))}
            </div>
          ) : null}
          <div className="flex gap-2">
            <input
              value={capabilityInput}
              onChange={(e) => setCapabilityInput(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === "Enter" || e.key === ",") {
                  e.preventDefault()
                  addCapability(capabilityInput)
                }
              }}
              placeholder="Type a capability and press Enter…"
              className={cn(
                "flex-1 rounded-md border border-slate-300 bg-white px-3 py-1.5 text-sm",
                "dark:border-slate-700 dark:bg-slate-900",
              )}
            />
            <button
              type="button"
              onClick={() => addCapability(capabilityInput)}
              className="rounded-md border border-slate-300 px-3 py-1.5 text-sm hover:bg-slate-100 dark:border-slate-700 dark:hover:bg-slate-800"
            >
              Add
            </button>
          </div>
          {overrideCapabilities.length === 0 ? (
            <p className="mt-1.5 text-xs text-slate-500">
              No required capabilities — the agent picker will show any idle
              agent.
            </p>
          ) : null}
        </section>

        {/* Step 4: notes */}
        <section>
          <label
            htmlFor="notes"
            className="mb-2 block text-sm font-medium text-slate-700 dark:text-slate-300"
          >
            Notes <span className="text-slate-400">(optional)</span>
          </label>
          <textarea
            id="notes"
            value={notes}
            onChange={(e) => setNotes(e.target.value)}
            rows={3}
            placeholder="Why this agent? Any context for the audit log…"
            className={cn(
              "w-full rounded-md border border-slate-300 bg-white px-3 py-2 text-sm",
              "dark:border-slate-700 dark:bg-slate-900",
            )}
          />
        </section>

        {submitError ? <ErrorBlock message={submitError} /> : null}

        <div className="flex items-center gap-3">
          <button
            type="submit"
            disabled={!agentId || submitting}
            className="rounded-md bg-sky-600 px-4 py-2 text-sm font-medium text-white hover:bg-sky-700 disabled:cursor-not-allowed disabled:opacity-50"
          >
            {submitting ? <Spinner size="sm" /> : "Assign agent"}
          </button>
          <Link
            href={`/tasks/${task.id}`}
            className="text-sm text-slate-500 hover:text-slate-700 dark:hover:text-slate-300"
          >
            Cancel
          </Link>
        </div>
      </form>
    </div>
  )
}
