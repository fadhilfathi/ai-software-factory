"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { PageHeader } from "@/components/layout/PageHeader";
import { useCreateAgent } from "@/lib/hooks";
import { Input } from "@/components/form/Input";
import { Select } from "@/components/form/Select";
import { Checkbox } from "@/components/form/Checkbox";
import { SpinnerButton } from "@/components/ui/Spinner";
import { ErrorBlock } from "@/components/ui/ErrorBlock";

const ROLE_DETAILS = [
  { value: "pm", label: "Project Manager", description: "Strategic planning and task decomposition.", icon: "📈" },
  { value: "architect", label: "Architect", description: "System design and technical blueprints.", icon: "📐" },
  { value: "developer", label: "Developer", description: "Code implementation and problem solving.", icon: "💻" },
  { value: "reviewer", label: "Reviewer", description: "Quality assurance and security audits.", icon: "🔍" },
  { value: "qa", label: "QA Engineer", description: "Automated testing and bug hunting.", icon: "🧪" },
  { value: "devops", label: "DevOps", description: "Deployment and infrastructure management.", icon: "🚀" },
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

export default function NewAgentPage() {
  const router = useRouter();
  const createAgent = useCreateAgent();

  const [name, setName] = useState("");
  const [role, setRole] = useState("developer");
  const [type, setType] = useState("LLM");
  const [model, setModel] = useState("gpt-4-turbo");
  const [provider, setProvider] = useState("openai");
  const [capabilities, setCapabilities] = useState<string[]>(["coding"]);
  const [error, setError] = useState<string | null>(null);

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
      const result = await createAgent.mutateAsync({
        name: name.trim(),
        role,
        type: type || undefined,
        model: model || undefined,
        provider: provider || undefined,
        capabilities: capabilities.length > 0 ? capabilities : undefined,
      });
      router.push(`/agents/${result.id}`);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to spawn agent");
    }
  };

  return (
    <div className="mx-auto max-w-3xl py-8">
      <div className="mb-8">
        <PageHeader
          title="Spawn New Agent"
          subtitle="Deploy a specialized AI agent to the factory workforce."
          actions={
            <button
              onClick={() => router.push("/agents")}
              className="rounded-lg border border-gray-800 px-4 py-2 text-sm font-medium text-gray-400 hover:bg-gray-800 hover:text-gray-200 transition-colors"
              type="button"
            >
              Cancel
            </button>
          }
        />
      </div>

      <div className="rounded-2xl border border-gray-800 bg-gray-950/50 p-8 shadow-2xl backdrop-blur-sm">
        {error && <ErrorBlock message={error} className="mb-6" />}

        <form onSubmit={handleSubmit} className="space-y-10">
          {/* Section 1: Role Selection */}
          <div className="space-y-6">
            <h3 className="text-sm font-bold uppercase tracking-widest text-emerald-500">1. Select Core Role</h3>
            <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
              {ROLE_DETAILS.map((option) => (
                <label
                  key={option.value}
                  className={`relative flex cursor-pointer flex-col rounded-xl border p-4 transition-all hover:bg-gray-900/40 ${
                    role === option.value
                      ? "border-emerald-500 bg-emerald-500/5 ring-1 ring-emerald-500"
                      : "border-gray-800 bg-gray-900/20"
                  }`}
                >
                  <input
                    type="radio"
                    name="role"
                    value={option.value}
                    checked={role === option.value}
                    onChange={(e) => setRole(e.target.value)}
                    className="sr-only"
                  />
                  <div className="text-xl mb-2">{option.icon}</div>
                  <span className={`text-sm font-bold ${role === option.value ? "text-emerald-400" : "text-gray-200"}`}>
                    {option.label}
                  </span>
                  <span className="mt-1 text-[10px] leading-relaxed text-gray-500">
                    {option.description}
                  </span>
                </label>
              ))}
            </div>
          </div>

          {/* Section 2: Configuration */}
          <div className="space-y-6">
            <h3 className="text-sm font-bold uppercase tracking-widest text-emerald-500">2. Technical Configuration</h3>
            <div className="grid gap-6 sm:grid-cols-2">
              <Input
                label="Agent Display Name"
                value={name}
                onChange={(e) => setName(e.target.value)}
                placeholder="e.g., Senior Gopher"
                required
                error={nameError}
                className="bg-gray-900/50"
              />

              <Input
                label="Model Engine"
                value={model}
                onChange={(e) => setModel(e.target.value)}
                placeholder="e.g., gpt-4-turbo"
                className="bg-gray-900/50"
              />

              <Select
                label="Infrastructure Provider"
                value={provider}
                onChange={(e) => setProvider(e.target.value)}
                options={[
                  { value: "openai", label: "OpenAI" },
                  { value: "anthropic", label: "Anthropic" },
                  { value: "google", label: "Google Vertex" },
                  { value: "local", label: "Local / Self-hosted" },
                ]}
                className="bg-gray-900/50"
              />

              <Input
                label="Agent Type"
                value={type}
                onChange={(e) => setType(e.target.value)}
                placeholder="e.g., LLM, Codex, Agentic"
                className="bg-gray-900/50"
              />
            </div>
          </div>

          {/* Section 3: Capabilities */}
          <div className="space-y-6">
            <h3 className="text-sm font-bold uppercase tracking-widest text-emerald-500">3. Functional Capabilities</h3>
            <div className="grid grid-cols-2 gap-3 sm:grid-cols-4">
              {CAPABILITY_OPTIONS.map((cap) => (
                <button
                  key={cap}
                  type="button"
                  onClick={() => toggleCapability(cap)}
                  className={cn(
                    "flex flex-col items-center justify-center gap-2 rounded-xl border p-4 transition-all text-center",
                    capabilities.includes(cap)
                      ? "border-emerald-500 bg-emerald-500/10 text-emerald-400"
                      : "border-gray-800 bg-gray-900/20 text-gray-500 hover:border-gray-700"
                  )}
                >
                  <span className="text-[10px] font-bold uppercase tracking-tighter">
                    {cap.replace(/_/g, " ")}
                  </span>
                </button>
              ))}
            </div>
          </div>

          <div className="flex items-center justify-between border-t border-gray-800 pt-8">
            <p className="text-[10px] text-gray-500 italic max-w-xs">
              * Spawning an agent initiates a compute instance. Billing begins immediately upon successful deployment.
            </p>
            <SpinnerButton
              type="submit"
              loading={createAgent.isPending}
              loadingText="Spawning..."
              disabled={!name.trim() || !!nameError}
              className="rounded-lg bg-emerald-500 px-10 py-3 text-sm font-bold text-white hover:bg-emerald-600 shadow-lg shadow-emerald-500/20 active:scale-95 transition-all"
            >
              Confirm Deployment
            </SpinnerButton>
          </div>
        </form>
      </div>
    </div>
  );
}
