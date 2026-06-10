/**
 * TypeScript types matching the Go backend models.
 * All field names use snake_case to align with API JSON responses.
 */

// ─── Generic API Response Wrappers ────────────────────────────────────────────

export type APIResponse<T> = { data: T };

export type PaginatedResponse<T> = {
  data: T[];
  pagination: {
    page: number;
    limit: number;
    total: number;
    pages: number;
  };
};

export type ApiErrorBody = {
  error: {
    code: string;
    message: string;
    details?: { field: string; message: string }[];
  };
  request_id?: string;
};

// ─── Projects ────────────────────────────────────────────────────────────────

export type ProjectStatus = "initializing" | "in_progress" | "completed" | "archived";

export type Project = {
  id: string;
  name: string;
  description?: string;
  status: ProjectStatus;
  template?: string;
  progress?: number;
  active_agents?: number;
  agents_spawned?: string[];
  artifacts?: unknown[];
  agents?: unknown[];
  created_at: string;
  updated_at: string;
};

export type CreateProjectPayload = {
  name: string;
  description?: string;
  template?: string;
  agents?: string[];
};

// ─── Tasks ───────────────────────────────────────────────────────────────────

export type TaskPriority = "low" | "medium" | "high" | "critical";
export type TaskStatus = "backlog" | "todo" | "in_progress" | "review" | "done";

export type Task = {
  id: string;
  project_id: string;
  title: string;
  description?: string;
  type?: string;
  acceptance_criteria?: string[];
  priority: TaskPriority;
  status: TaskStatus;
  estimated_hours?: number;
  assignee_agent_id?: string;
  created_at: string;
  updated_at: string;
};

export type CreateTaskPayload = {
  project_id: string;
  title: string;
  description?: string;
  type?: string;
  acceptance_criteria?: string[];
  priority?: TaskPriority;
  assignee_agent_id?: string;
};

export type UpdateTaskPayload = {
  status?: TaskStatus;
  assignee_agent_id?: string;
  priority?: TaskPriority;
};

// ─── Agents ──────────────────────────────────────────────────────────────────

export type AgentType = "pm" | "developer" | "reviewer" | "devops";
export type AgentStatus_ = "spawning" | "idle" | "working" | "completed" | "failed";

export type AgentConfig = {
  model?: string;
  temperature?: number;
};

export type Agent = {
  id: string;
  type: AgentType;
  status: AgentStatus_;
  project_id?: string;
  config?: AgentConfig;
  current_task?: string;
  tasks_completed?: number;
  uptime?: number;
  created_at: string;
  updated_at: string;
};

export type Assignment = {
  id: string;
  agent_id: string;
  task_id: string;
  status: string;
  estimated_completion?: string;
  created_at: string;
};

// ─── Auth / Users ────────────────────────────────────────────────────────────

export type User = {
  id: string;
  name: string;
  email: string;
  role: string;
};

export type LoginPayload = {
  email: string;
  password: string;
};

export type LoginResponse = {
  user: User;
  access_token: string;
};

export type RegisterPayload = {
  name: string;
  email: string;
  password: string;
  role?: string;
};

// ─── Code / Reviews / Deployments ────────────────────────────────────────────

export type CodeGeneratePayload = {
  project_id: string;
  prompt: string;
  language: string;
};

export type ReviewPayload = {
  project_id: string;
  file_path: string;
  content: string;
};

export type Review = {
  id: string;
  project_id: string;
  file_path: string;
  status: string;
  comments: unknown[];
  created_at: string;
};

export type DeploymentPayload = {
  project_id: string;
  target: string;
};

export type Deployment = {
  id: string;
  project_id: string;
  target: string;
  status: string;
  url?: string;
  created_at: string;
};

// ─── Dashboard (composite) ───────────────────────────────────────────────────

export type DashboardMetrics = {
  active_projects: number;
  completed_projects: number;
  success_rate: number;
  total_spend: number;
};

export type ActivityItem = {
  id: string;
  agent_type: string;
  text: string;
  project_id?: string;
  created_at: string;
};
