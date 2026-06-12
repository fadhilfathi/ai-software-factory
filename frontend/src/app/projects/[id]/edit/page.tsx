"use client";

import { use, useState } from "react";
import { useRouter } from "next/navigation";
import { PageHeader } from "@/components/layout/PageHeader";
import { useProject, useUpdateProject } from "@/lib/hooks";
import type { Project } from "@/lib/types";
import { Input } from "@/components/form/Input";
import { Textarea } from "@/components/form/Textarea";
import { Select } from "@/components/form/Select";
import { SpinnerButton } from "@/components/ui/Spinner";
import { ErrorBlock } from "@/components/ui/ErrorBlock";
import { Skeleton } from "@/components/ui/Skeleton";

const TEMPLATE_OPTIONS = [
  { value: "Node.js + AWS", label: "Node.js + AWS" },
  { value: "Python + Railway", label: "Python + Railway" },
  { value: "Go + Vercel", label: "Go + Vercel" },
  { value: "Rust + Self-hosted", label: "Rust + Self-hosted" },
];

export default function EditProjectPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = use(params);
  const router = useRouter();
  const { data: project, isLoading, isError } = useProject(id);
  const updateProject = useUpdateProject();

  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [template, setTemplate] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [initialized, setInitialized] = useState(false);

  if (project && !initialized) {
    setName(project.name ?? "");
    setDescription(project.description ?? "");
    setTemplate((project as { template?: string }).template ?? "");
    setInitialized(true);
  }

  const nameError = name.length > 0 && name.trim().length < 2 ? "Name must be at least 2 characters" : null;

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!name.trim()) return;
    setError(null);
    try {
      await updateProject.mutateAsync({
        id,
        name: name.trim(),
        description: description.trim() || undefined,
        template: template || undefined,
      } as Partial<Project> & { id?: string; template?: string });
      router.push(`/projects/${id}`);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to update project");
    }
  };

  if (isLoading) {
    return (
      <div className="mx-auto max-w-2xl">
        <PageHeader title="Edit Project" />
        <div className="space-y-4">
          <Skeleton className="h-10 w-full" />
          <Skeleton className="h-24 w-full" />
          <Skeleton className="h-10 w-full" />
        </div>
      </div>
    );
  }

  if (isError || !project) {
    return (
      <div className="mx-auto max-w-2xl">
        <PageHeader title="Project Not Found" />
        <ErrorBlock
          message="Could not load this project for editing."
          title="Error"
        />
      </div>
    );
  }

  return (
    <div className="mx-auto max-w-2xl">
      <PageHeader
        title="Edit Project"
        actions={
          <button
            onClick={() => router.push(`/projects/${id}`)}
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
          label="Project Name"
          value={name}
          onChange={(e) => setName(e.target.value)}
          placeholder="My Awesome Project"
          required
          error={nameError}
        />

        <Textarea
          label="Description"
          value={description}
          onChange={(e) => setDescription(e.target.value)}
          placeholder="Describe your project..."
          rows={4}
        />

        <Select
          label="Template"
          value={template}
          onChange={(e) => setTemplate(e.target.value)}
          options={TEMPLATE_OPTIONS}
          placeholder="Select a template (optional)"
        />

        <div className="flex justify-end gap-3 pt-4">
          <button
            type="button"
            onClick={() => router.push(`/projects/${id}`)}
            className="rounded-lg border border-gray-800 px-4 py-2 text-sm font-medium text-gray-300 hover:bg-gray-800 transition-colors"
          >
            Cancel
          </button>
          <SpinnerButton
            type="submit"
            loading={updateProject.isPending}
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
