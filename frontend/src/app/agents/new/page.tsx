"use client"

import { useMemo, useState } from "react"
import { useRouter } from "next/navigation"

import { useCreateAgent } from "@/lib/hooks"
import { useProjectFilters } from "@/hooks/useProjectFilters"
import type { AgentMetadata, CreateAgentPayload } from "@/lib/types"
import { cn } from "@/lib/utils"

import { PageHeader } from "@/components/layout/PageHeader"
import { Input } from "@/components/form/Input"
import { Textarea } from "@/components/form/Textarea"
import { SpinnerButton } from "@/components/ui/Spinner"
import { ErrorBlock } from "@/components/ui/ErrorBlock"
import { ProjectPickerGate } from "@/components/agents/ProjectPickerGate"
import { CapabilityMultiSelect } from "@/components/agents/CapabilityMultiSelect"

export default function NewAgentPage() {
  return (
    <ProjectPickerGate>
      <NewAgentForm />
    </ProjectPickerGate>
  )
}

function NewAgentForm() {
  const router = useRouter()
  const { projectId } = useProjectFilters()
  const createAgent = useCreateAgent()

  const [name, setName] = useState("")
  const [role, setRole] = useState("")
  const [capabilities, setCapabilities] = useState<string[]>([])
  const [metadataOpen, setMetadataOpen] = useState(false)
  const [model, setModel] = useState("")
  const [provider, setProvider] = useState("")
  const [type, setType] = useState("")
  const [description, setDescription] = useState("")

  const metadata = useMemo<AgentMetadata>(() => {
    const m: AgentMetadata = {}
    if (model.trim()) m.model = model.trim()
    if (provider.trim()) m.provider = provider.trim()
    if (type.trim()) m.type = type.trim()
    if (description.trim()) m.description = description.trim()
    return m
  }, [model, provider, type, description])

  const canSubmit =
    name.trim().length > 0 &&
    role.trim().length > 0 &&
    capabilities.length > 0 &&
    !!projectId

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault()
    if (!canSubmit || !projectId) return
    const payload: CreateAgentPayload = {
      project_id: projectId,
      name: name.trim(),
      role: role.trim(),
      capabilities,
      ...(Object.keys(metadata).length > 0 ? { metadata } : {}),
    }
    const agent = await createAgent.mutateAsync(payload)
    router.push(`/agents/${agent.id}`)
  }

  return (
    <div className="mx-auto max-w-2xl space-y-6">
      <PageHeader
        title="New agent"
        description="Register a new agent in the current project."
      />

      {createAgent.isError ? (
        <ErrorBlock error={createAgent.error} />
      ) : null}

      <form onSubmit={onSubmit} className="space-y-5">
        <FormRow label="Name" required hint="1-100 characters, must be unique within the project.">
          <Input
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder="e.g. Backend Architect"
            maxLength={100}
            required
          />
        </FormRow>

        <FormRow label="Role" required hint="Free-form role label (e.g. architect, developer, qa).">
          <Input
            value={role}
            onChange={(e) => setRole(e.target.value)}
            placeholder="e.g. architect"
            required
          />
        </FormRow>

        <FormRow
          label="Capabilities"
          required
          hint="At least one. Capabilities must exist in the catalog (useCapabilities)."
        >
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
            <span>Metadata (optional)</span>
            <span aria-hidden>{metadataOpen ? "▴" : "▾"}</span>
          </button>
          {metadataOpen ? (
            <div className="space-y-4 border-t border-slate-200 p-4 dark:border-slate-800">
              <FormRow label="Model" hint="e.g. gpt-4o, claude-3.5-sonnet">
                <Input
                  value={model}
                  onChange={(e) => setModel(e.target.value)}
                  placeholder="model name"
                />
              </FormRow>
              <FormRow label="Provider" hint="e.g. openai, anthropic, internal">
                <Input
                  value={provider}
                  onChange={(e) => setProvider(e.target.value)}
                  placeholder="provider"
                />
              </FormRow>
              <FormRow label="Type" hint="Free-form subtype label">
                <Input
                  value={type}
                  onChange={(e) => setType(e.target.value)}
                  placeholder="e.g. backend, frontend, research"
                />
              </FormRow>
              <FormRow label="Description" hint="Short human-readable description">
                <Textarea
                  value={description}
                  onChange={(e) => setDescription(e.target.value)}
                  rows={3}
                  placeholder="What does this agent do?"
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
            loading={createAgent.isPending}
            disabled={!canSubmit}
          >
            Create agent
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
      <label
        className={cn(
          "block text-sm font-medium text-slate-700 dark:text-slate-200",
        )}
      >
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
