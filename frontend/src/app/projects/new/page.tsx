"use client";

import { PageHeader } from "@/components/layout/PageHeader";
import { useCreateProject } from "@/lib/hooks";
import { useRouter } from "next/navigation";
import { useState } from "react";
import { StepIndicator } from "@/components/form/StepIndicator";
import { Input } from "@/components/form/Input";
import { Textarea } from "@/components/form/Textarea";
import { Select } from "@/components/form/Select";
import { Checkbox } from "@/components/form/Checkbox";
import { SpinnerButton } from "@/components/ui/Spinner";
import { ErrorBlock } from "@/components/ui/ErrorBlock";

const AGENT_OPTIONS = ["PM", "Architect", "Developer", "Reviewer", "QA", "DevOps"];

const TECH_STACKS = [
  { value: "Node.js", label: "Node.js" },
  { value: "Python", label: "Python" },
  { value: "Go", label: "Go" },
  { value: "Rust", label: "Rust" },
];

const DEPLOY_TARGETS = [
  { value: "AWS", label: "AWS" },
  { value: "Vercel", label: "Vercel" },
  { value: "Railway", label: "Railway" },
  { value: "Self-hosted", label: "Self-hosted" },
];

const STEPS = [
  { num: 1, label: "Describe" },
  { num: 2, label: "Configure" },
  { num: 3, label: "Review" },
];

export default function NewProjectPage() {
  const router = useRouter();
  const createProject = useCreateProject();

  const [step, setStep] = useState(1);
  const [description, setDescription] = useState("");
  const [name, setName] = useState("");
  const [techStack, setTechStack] = useState("Node.js");
  const [deployTarget, setDeployTarget] = useState("AWS");
  const [selectedAgents, setSelectedAgents] = useState<string[]>([...AGENT_OPTIONS]);
  const [error, setError] = useState<string | null>(null);

  const canContinueStep1 = description.trim().length >= 10 && name.trim().length >= 2;
  const canContinueStep2 = selectedAgents.length > 0;
  const isSubmitting = createProject.isPending;

  const handleSubmit = async () => {
    setError(null);
    try {
      const result = await createProject.mutateAsync({
        name: name.trim(),
        description: description.trim(),
        template: `${techStack} + ${deployTarget}`,
        agents: selectedAgents.map((a) => a.toLowerCase()),
      });
      router.push(`/projects/${result.data.id}`);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to create project");
    }
  };

  return (
    <div className="mx-auto max-w-2xl">
      <PageHeader
        title="New Project"
        actions={
          <button
            onClick={() => router.push("/projects")}
            className="text-sm text-gray-400 hover:text-gray-200 transition-colors"
            type="button"
          >
            Cancel
          </button>
        }
      />

      <StepIndicator steps={STEPS} currentStep={step} />

      {/* Error */}
      {error && <ErrorBlock message={error} className="mb-4" />}

      {/* Step 1: Describe */}
      {step === 1 && (
        <div className="space-y-4">
          <Input
            label="Project Name"
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder="My Awesome Project"
            required
          />
          <Textarea
            label="Project Description"
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            placeholder="Describe what you want to build..."
            rows={5}
            hint={`${description.length} / 10 minimum characters`}
          />
        </div>
      )}

      {/* Step 2: Configure */}
      {step === 2 && (
        <div className="space-y-4">
          <Select
            label="Tech Stack"
            value={techStack}
            onChange={(e) => setTechStack(e.target.value)}
            options={TECH_STACKS}
          />
          <Select
            label="Deploy Target"
            value={deployTarget}
            onChange={(e) => setDeployTarget(e.target.value)}
            options={DEPLOY_TARGETS}
          />
          <fieldset>
            <legend className="mb-1 block text-sm font-medium text-gray-300">Agents</legend>
            <div className="space-y-2">
              {AGENT_OPTIONS.map((agent) => {
                const checked = selectedAgents.includes(agent);
                return (
                  <Checkbox
                    key={agent}
                    label={agent}
                    checked={checked}
                    onChange={() => {
                      setSelectedAgents((prev) =>
                        prev.includes(agent)
                          ? prev.filter((a) => a !== agent)
                          : [...prev, agent],
                      );
                    }}
                  />
                );
              })}
            </div>
          </fieldset>
        </div>
      )}

      {/* Step 3: Review */}
      {step === 3 && (
        <div className="overflow-hidden rounded-lg border border-gray-800 bg-gray-950">
          <div className="border-b border-gray-800 px-6 py-4">
            <h3 className="text-lg font-semibold text-gray-200">Review Configuration</h3>
          </div>
          <div className="divide-y divide-gray-800">
            <ReviewRow label="Name" value={name || "(not set)"} />
            <ReviewRow label="Description" value={description || "(not set)"} />
            <ReviewRow label="Tech Stack" value={techStack} />
            <ReviewRow label="Deploy Target" value={deployTarget} />
            <ReviewRow
              label={`Agents (${selectedAgents.length})`}
              value={selectedAgents.join(", ") || "None selected"}
            />
          </div>
        </div>
      )}

      {/* Navigation Buttons */}
      <div className="mt-8 flex justify-between">
        {step > 1 ? (
          <button
            onClick={() => setStep(step - 1)}
            disabled={isSubmitting}
            className="rounded-lg border border-gray-800 px-6 py-2 text-sm font-medium text-gray-300 hover:bg-gray-800 transition-colors disabled:opacity-50"
            type="button"
          >
            Back
          </button>
        ) : (
          <div />
        )}

        {step < 3 ? (
          <button
            onClick={() => setStep(step + 1)}
            disabled={(step === 1 && !canContinueStep1) || (step === 2 && !canContinueStep2)}
            className="rounded-lg bg-emerald-500 px-6 py-2 text-sm font-medium text-white hover:bg-emerald-600 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
            type="button"
          >
            Continue
          </button>
        ) : (
          <SpinnerButton
            onClick={handleSubmit}
            loading={isSubmitting}
            loadingText="Creating..."
          >
            Create Project
          </SpinnerButton>
        )}
      </div>
    </div>
  );
}

function ReviewRow({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex justify-between px-6 py-3">
      <span className="text-sm text-gray-400">{label}</span>
      <span className="text-sm text-gray-200 max-w-[250px] text-right truncate">
        {value}
      </span>
    </div>
  );
}
