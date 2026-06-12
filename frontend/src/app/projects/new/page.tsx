"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { PageHeader } from "@/components/layout/PageHeader";
import { useCreateProject } from "@/lib/hooks";
import { Input } from "@/components/form/Input";
import { Textarea } from "@/components/form/Textarea";
import { Select } from "@/components/form/Select";
import { SpinnerButton } from "@/components/ui/Spinner";
import { ErrorBlock } from "@/components/ui/ErrorBlock";

const TEMPLATE_OPTIONS = [
  { value: "Node.js + AWS", label: "Fullstack Node.js", description: "Modern Express backend with AWS deployment scripts." },
  { value: "Python + Railway", label: "FastAPI Backend", description: "High-performance Python API ready for Railway." },
  { value: "Go + Vercel", label: "Go Microservice", description: "Clean Go architecture optimized for serverless." },
  { value: "Rust + Self-hosted", label: "Rust Performance", description: "Blazing fast Rust service with Docker support." },
];

export default function NewProjectPage() {
  const router = useRouter();
  const createProject = useCreateProject();

  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [template, setTemplate] = useState("");
  const [error, setError] = useState<string | null>(null);

  const nameError = name.length > 0 && name.trim().length < 2 ? "Name must be at least 2 characters" : null;

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!name.trim()) return;
    setError(null);
    try {
      const result = await createProject.mutateAsync({
        name: name.trim(),
        description: description.trim() || undefined,
        template: template || undefined,
      });
      router.push(`/projects/${result.id}`);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to create project");
    }
  };

  return (
    <div className="mx-auto max-w-2xl py-8">
      <div className="mb-8">
        <PageHeader
          title="Create New Project"
          subtitle="Define your project vision and let the agents handle the rest."
          actions={
            <button
              onClick={() => router.push("/projects")}
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

        <form onSubmit={handleSubmit} className="space-y-8">
          <div className="space-y-6">
            <h3 className="text-sm font-bold uppercase tracking-widest text-emerald-500">1. Basic Information</h3>
            <div className="grid gap-6">
              <Input
                label="Project Name"
                value={name}
                onChange={(e) => setName(e.target.value)}
                placeholder="e.g., Enterprise Auth Service"
                required
                error={nameError}
                className="bg-gray-900/50"
              />

              <Textarea
                label="Description"
                value={description}
                onChange={(e) => setDescription(e.target.value)}
                placeholder="What are we building? Describe the core functionality and goals..."
                rows={4}
                className="bg-gray-900/50"
              />
            </div>
          </div>

          <div className="space-y-6">
            <h3 className="text-sm font-bold uppercase tracking-widest text-emerald-500">2. Architecture Template</h3>
            <div className="grid gap-4 sm:grid-cols-2">
              {TEMPLATE_OPTIONS.map((option) => (
                <label
                  key={option.value}
                  className={`relative flex cursor-pointer flex-col rounded-xl border p-4 transition-all hover:bg-gray-900/40 ${
                    template === option.value
                      ? "border-emerald-500 bg-emerald-500/5 ring-1 ring-emerald-500"
                      : "border-gray-800 bg-gray-900/20"
                  }`}
                >
                  <input
                    type="radio"
                    name="template"
                    value={option.value}
                    checked={template === option.value}
                    onChange={(e) => setTemplate(e.target.value)}
                    className="sr-only"
                  />
                  <span className={`text-sm font-bold ${template === option.value ? "text-emerald-400" : "text-gray-200"}`}>
                    {option.label}
                  </span>
                  <span className="mt-1 text-[11px] leading-relaxed text-gray-500">
                    {option.description}
                  </span>
                  {template === option.value && (
                    <div className="absolute top-2 right-2">
                      <div className="h-1.5 w-1.5 rounded-full bg-emerald-500" />
                    </div>
                  )}
                </label>
              ))}
            </div>
          </div>

          <div className="flex items-center justify-between border-t border-gray-800 pt-8">
            <p className="text-xs text-gray-500 italic">
              * Agents will be automatically assigned to your project after creation.
            </p>
            <SpinnerButton
              type="submit"
              loading={createProject.isPending}
              loadingText="Initializing..."
              disabled={!name.trim() || !!nameError}
              className="rounded-lg bg-emerald-500 px-8 py-3 text-sm font-bold text-white hover:bg-emerald-600 shadow-lg shadow-emerald-500/20 active:scale-95 transition-all"
            >
              Launch Project
            </SpinnerButton>
          </div>
        </form>
      </div>
    </div>
  );
}
