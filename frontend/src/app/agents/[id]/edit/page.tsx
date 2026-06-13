"use client"

import { use, useEffect, useMemo, useState } from "react"
import { useRouter } from "next/navigation"

import { useAgent, useUpdateAgent } from "@/lib/hooks"
import type { AgentMetadata, UpdateAgentPayload } from "@/lib/types"
import { cn } from "@/lib/utils"

import { PageHeader } from "@/components/layout/PageHeader"
import { Input } from "@/components/form/Input"
import { Textarea } from "@/components/form/Textarea"
import { SpinnerButton } from "@/components/ui/Spinner"
import { ErrorBlock } from "@/components/ui/ErrorBlock"
import { Skeleton } from "@/components/ui/Skeleton"
import { ProjectPickerGate } from "@/components/agents/ProjectPickerGate"
import { CapabilityMultiSelect } from "@/components/agents/CapabilityMultiSelect"

export default function EditAgentPage({
  params,
}: {
  params: Promise<{ id: string }>
}) {
  return (
    <ProjectPickerGate>
      <EditAgentForm params={params} />
    </ProjectPickerGate>
  )
}

function EditAgentForm({
  params,
}: {
  params: Promise<{ id: string }>
}) {
  const router = useRouter()
  const { id } = use(params)
  const agentQuery = useAgent(id)
  const updateAgent = useUpdateAgent()

  const [role, setRole] = useState("")
  const [capabilities, setCapabilities] = useState<string[]>([])
  const [metadataOpen, setMetadataOpen] = useState(true)
  const [model, setModel] = useState("")
  const [provider, setProvider] = useState("")
  const [type, setType] = useState("")
  const [description, setDescription] = useState("")
  const [hydrated, setHydrated] = useState(false)
  const [currentVersion, setCurrentVersion] = useState<number | undefined>()

  // Seed the form from the loaded agent exactly once. We also stash the
  // loaded version number so the PUT can include it for optimistic
  // concurrency (spec §1.4: 409 VERSION_CONFLICT on stale version).
  useEffect(() => {
    if (agentQuery.data && !hydrated) {
      setRole(agentQuery.data.role ?? "")
      setCapabilities(agentQuery.data.capabilities ?? [])
      const md = (agentQuery.data.metadata ?? {}) as AgentMetadata
      setModel(typeof md.model === "string" ? md.model : "")
      setProvider(typeof md.provider === "string" ? md.provider : "")
      setType(typeof md.type === "string" ? md.type : "")
      setDescription(typeof md.description === "string" ? md.description : "")
      setCurrentVersion(agentQuery.data.version)
      setHydrated(true)
    }
  }, [agentQuery.data, hydrated])

  const metadata = useMemo<AgentMetadata>(() => {
    const m: AgentMetadata = {}
    if (model.trim()) m.model = model.trim()
    if (provider.trim()) m.provider = provider.trim()
    if (type.trim()) m.type = type.trim()
    if (description.trim()) m.description = description.trim()
    return m
  }, [model, provider, type, description])

  const canSubmit = role.trim().length > 0 && capabilities.length > 0

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault()
    if (!canSubmit) return
    const payload: UpdateAgentPayload = {
      role: role.trim(),
      capabilities,
      ...(Object.keys(metadata).length > 0 ? { metadata } : {}),
      ...(currentVersion != null ? { version: currentVersion } : {}),
    }
    const agent = await updateAgent.mutateAsync({ id, payload })
    router.push(`/agents/${agent.id}`)
  }

  if (agentQuery.isError) {
    return <ErrorBlock error={agentQuery.error} onRetry={() => agentQuery.refetch()} />
  }

  if (agentQuery.isLoading || !agentQuery.data) {
    return (
      <div className="mx-auto max-w-2xl space-y-4">
        <Skeleton className="h-8 w-1/3" />
        <Skeleton className="h-10" />
        <Skeleton className="h-10" />
        <Skeleton className="h-10" />
      </div>
    )
  }

  return (
    <div className="mx-auto max-w-2xl space-y-6">
      <PageHeader
        title={`Edit ${agentQuery.data.name}`}
        description="name, project, and status are immutable here. Update role, capabilities, and metadata."
      />

      {updateAgent.isError ? (
        <ErrorBlock
          error={updateAgent.error}
          message={
            // Spec §1.4: 409 VERSION_CONFLICT on stale version.
            (updateAgent.error as Error & { status?: number })?.status === 409
              ? "This agent was modified by someone else. Refresh and try again."
              : undefined
          }
        />
      ) : null}

      <form onSubmit={onSubmit} className="space-y-5">
        <ReadOnlyRow label="Name">{agentQuery.data.name}</ReadOnlyRow>
        <ReadOnlyRow label="Status">
          <span className="text-sm text-slate-700 dark:text-slate-200">
            {agentQuery.data.status}
          </span>
        </ReadOnlyRow>

        <FormRow label="Role" required>
          <Input
            value={role}
            onChange={(e) => setRole(e.target.value)}
            placeholder="e.g. architect"
            required
          />
        </FormRow>

        <FormRow label="Capabilities" required>
          <CapabilityMultiSelect
            value={capabilities}
            onChange={setCapabilities}
          />
        </FormRow>

        <div className="rounded-md border border-slate-200 dark:border-slate-800">
          <button
            type="button"
            onClick={() => setMetadataOpen((o) => !o)}
            className="flex w-full items-center justify-between px-4 py-2 text-left text-sm font-medium text-slate-700 dark:text-slate-200"
            aria-expanded={metadataOpen}
          >
            <span>Metadata</span>
            <span aria-hidden>{metadataOpen ? "▴" : "▾"}</span>
          </button>
          {metadataOpen ? (
            <div className="space-y-4 border-t border-slate-200 p-4 dark:border-slate-800">
              <FormRow label="Model">
                <Input value={model} onChange={(e) => setModel(e.target.value)} placeholder="model name" />
              </FormRow>
              <FormRow label="Provider">
                <Input value={provider} onChange={(e) => setProvider(e.target.value)} placeholder="provider" />
              </FormRow>
              <FormRow label="Type">
                <Input value={type} onChange={(e) => setType(e.target.value)} placeholder="type" />
              </FormRow>
              <FormRow label="Description">
                <Textarea
                  value={description}
                  onChange={(e) => setDescription(e.target.value)}
                  rows={3}
                />
              </FormRow>
            </div>
          ) : null}
        </div>

        <div className="flex items-center justify-end gap-2 border-t border-slate-200 pt-4 dark:border-slate-800">
          <button
            type="button"
            onClick={() => router.back()}
            className="rounded-md border border-slate-300 px-3 py-1.5 text-sm dark:border-slate-700"
          >
            Cancel
          </button>
          <SpinnerButton
            type="submit"
            loading={updateAgent.isPending}
            disabled={!canSubmit}
          >
            Save changes
          </SpinnerButton>
        </div>
      </form>
    </div>
  )
}

function FormRow({
  label,
  hint,
  required,
  children,
}: {
  label: string
  hint?: string
  required?: boolean
  children: React.ReactNode
}) {
  return (
    <div className="space-y-1">
      <label className="block text-sm font-medium text-slate-700 dark:text-slate-200">
        {label}
        {required ? <span className="ml-0.5 text-rose-500">*</span> : null}
      </label>
      {children}
      {hint ? (
        <p className="text-xs text-slate-500 dark:text-slate-400">{hint}</p>
      ) : null}
    </div>
  )
}

function ReadOnlyRow({
  label,
  children,
}: {
  label: string
  children: React.ReactNode
}) {
  return (
    <div className="space-y-1">
      <label className="block text-sm font-medium text-slate-500 dark:text-slate-400">
        {label}
      </label>
      <div
        className={cn(
          "rounded-md border border-dashed border-slate-300 bg-slate-50 px-3 py-1.5 text-sm text-slate-700",
          "dark:border-slate-700 dark:bg-slate-800/50 dark:text-slate-200",
        )}
      >
        {children}
      </div>
    </div>
  )
}
