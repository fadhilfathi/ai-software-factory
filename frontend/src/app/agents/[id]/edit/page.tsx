"use client";

import { use, useState } from "react";
import { useRouter } from "next/navigation";
import { PageHeader } from "@/components/layout/PageHeader";
import { useAgent, useUpdateAgent } from "@/lib/hooks";
import { Input } from "@/components/form/Input";
import { Select } from "@/components/form/Select";
import { Checkbox } from "@/components/form/Checkbox";
import { SpinnerButton } from "@/components/ui/Spinner";
import { ErrorBlock } from "@/components/ui/ErrorBlock";
import { Skeleton } from "@/components/ui/Skeleton";

const ROLE_OPTIONS = [
  { value: "pm", label: "Project Manager" },
  { value: "architect", label: "Architect" },
  { value: "developer", label: "Developer" },
  { value: "qa", label: "QA" },
  { value: "devops", label: "DevOps" },
  { value: "security", label: "Security" },
  { value: "techwriter", label: "Tech Writer" },
];

const STATUS_OPTIONS = [
  { value: "spawning", label: "Spawning" },
  { value: "idle", label: "Idle" },
  { value: "working", label: "Working" },
  { value: "completed", label: "Completed" },
  { value: "failed", label: "Failed" },
];

const CAPABILITY_OPTIONS = [
  "architecture",
  "coding",
  "testing",
  "security",
  "documentation",
  "devops",
  "project_management",
  "data_engineering",
];

export default function EditAgentPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = use(params);
  const router = useRouter();
  const { data: agent, isLoading, isError } = useAgent(id);
  const updateAgent = useUpdateAgent();

  const [name, setName] = useState("");
  const [role, setRole] = useState("developer");
  const [type, setType] = useState("");
  const [model, setModel] = useState("");
  const [provider, setProvider] = useState("");
  const [status, setStatus] = useState("idle");
  const [capabilities, setCapabilities] = useState<string[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [initialized, setInitialized] = useState(false);

  if (agent && !initialized) {
    setName(agent.name ?? "");
    setRole(agent.role ?? "developer");
    setType(agent.type ?? "");
    setModel(agent.model ?? "");
    setProvider(agent.provider ?? "");
    setStatus(agent.status ?? "idle");
    setCapabilities(agent.capabilities ?? []);
    setInitialized(true);
  }

  const nameError = name.length > 0 && name.trim().length < 2 ? "Name must be at least 2 characters" : null;

  const toggleCapability = (cap: string) => {
    setCapabilities((prev) =>
      prev.includes(cap) ? prev.filter((c) => c !== cap) : [...prev, cap],
    );
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!name.trim()) return;
    setError(null);
    try {
      await updateAgent.mutateAsync({
        id,
        name: name.trim(),
        role,
        type: type || undefined,
        model: model || undefined,
        provider: provider || undefined,
        status: status as any,
        capabilities: capabilities.length > 0 ? capabilities : undefined,
      });
      router.push(`/agents/${id}`);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to update agent");
    }
  };

  if (isLoading) {
    return (
      <div className="mx-auto max-w-2xl">
        <PageHeader title="Edit Agent" />
        <div className="space-y-4">
          <Skeleton className="h-10 w-full" />
          <Skeleton className="h-10 w-full" />
          <Skeleton className="h-10 w-full" />
          <Skeleton className="h-10 w-full" />
          <Skeleton className="h-10 w-full" />
        </div>
      </div>
    );
  }

  if (isError || !agent) {
    return (
      <div className="mx-auto max-w-2xl">
        <PageHeader title="Agent Not Found" />
        <ErrorBlock
          message="Could not load this agent for editing."
          title="Error"
        />
      </div>
    );
  }

  return (
    <div className="mx-auto max-w-2xl">
      <PageHeader
        title="Edit Agent"
        actions={
          <button
            onClick={() => router.push(`/agents/${id}`)}
            className="text-sm text-gray-400 hover:text-gray-200 transition-colors"
            type="button"
          >
            Cancel
          </button>
        }
      />

      {error && <ErrorBlock message={error} className="mb-4" />}

      <form onSubmit={handleSubmit} className="space-y-4">
        <Input
          label="Name"
          value={name}
          onChange={(e) => setName(e.target.value)}
          placeholder="My Agent"
          required
          error={nameError}
        />

        <Select
          label="Role"
          value={role}
          onChange={(e) => setRole(e.target.value)}
          options={ROLE_OPTIONS}
        />

        <Input
          label="Type"
          value={type}
          onChange={(e) => setType(e.target.value)}
          placeholder="e.g. llm, codex"
        />

        <Input
          label="Model"
          value={model}
          onChange={(e) => setModel(e.target.value)}
          placeholder="e.g. gpt-4, claude-3"
        />

        <Input
          label="Provider"
          value={provider}
          onChange={(e) => setProvider(e.target.value)}
          placeholder="e.g. openai, anthropic"
        />

        <Select
          label="Status"
          value={status}
          onChange={(e) => setStatus(e.target.value)}
          options={STATUS_OPTIONS}
        />

        <fieldset>
          <legend className="mb-2 block text-sm font-medium text-gray-300">
            Capabilities
          </legend>
          <div className="grid grid-cols-2 gap-2">
            {CAPABILITY_OPTIONS.map((cap) => (
              <Checkbox
                key={cap}
                label={cap.replace(/_/g, " ")}
                checked={capabilities.includes(cap)}
                onChange={() => toggleCapability(cap)}
              />
            ))}
          </div>
        </fieldset>

        <div className="flex justify-end gap-3 pt-4">
          <button
            type="button"
            onClick={() => router.push(`/agents/${id}`)}
            className="rounded-lg border border-gray-800 px-4 py-2 text-sm font-medium text-gray-300 hover:bg-gray-800 transition-colors"
          >
            Cancel
          </button>
          <SpinnerButton
            type="submit"
            loading={updateAgent.isPending}
            loadingText="Saving..."
            disabled={!name.trim() || !!nameError}
          >
            Save Changes
          </SpinnerButton>
        </div>
      </form>
    </div>
  );
}
