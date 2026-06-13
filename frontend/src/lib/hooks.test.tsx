import { renderHook, act, waitFor } from "@testing-library/react";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { useUpdateTaskStatus } from "./hooks";
import { api } from "./api";
import { queryKeys } from "./queryKeys";
import { ReactNode } from "react";
import type { PaginatedResponse, Task } from "./types";

vi.mock("./api", () => ({
  api: {
    patch: vi.fn(),
  },
}));

const mockTask: Task = {
  id: "task-1",
  project_id: "proj-1",
  title: "Test Task",
  description: "Test Description",
  status: "backlog",
  priority: "medium",
  created_at: new Date().toISOString(),
  updated_at: new Date().toISOString(),
};

describe("useUpdateTaskStatus", () => {
  let queryClient: QueryClient;

  beforeEach(() => {
    queryClient = new QueryClient({
      defaultOptions: {
        queries: { retry: false },
        mutations: { retry: false },
      },
    });
    vi.clearAllMocks();
  });

  const wrapper = ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  );

  it("should optimistically update task status and rollback on error", async () => {
    const mockTasksList = { data: [mockTask], pagination: { total: 1, page: 1, limit: 10 } };
    queryClient.setQueryData(queryKeys.tasks.list({ project_id: "proj-1" }), mockTasksList);
    queryClient.setQueryData(queryKeys.tasks.detail("task-1"), mockTask);

    let rejectApi!: (reason?: unknown) => void;
    vi.mocked(api.patch).mockImplementationOnce(() => new Promise((_, reject) => {
      rejectApi = reject;
    }));

    const { result } = renderHook(() => useUpdateTaskStatus(), { wrapper });

    act(() => {
      result.current.mutate({ id: "task-1", status: "in_progress" });
    });

    await waitFor(() => {
      const updatedList: PaginatedResponse<Task> | undefined = queryClient.getQueryData(queryKeys.tasks.list({ project_id: "proj-1" }));
      expect(updatedList?.data[0].status).toBe("in_progress");

      const updatedDetail: Task | undefined = queryClient.getQueryData(queryKeys.tasks.detail("task-1"));
      expect(updatedDetail?.status).toBe("in_progress");
    });

    act(() => {
      rejectApi(new Error("API Error"));
    });

    await waitFor(() => {
      expect(result.current.isError).toBe(true);
    });

    const rolledBackList: PaginatedResponse<Task> | undefined = queryClient.getQueryData(queryKeys.tasks.list({ project_id: "proj-1" }));
    expect(rolledBackList?.data[0].status).toBe("backlog");

    const rolledBackDetail: Task | undefined = queryClient.getQueryData(queryKeys.tasks.detail("task-1"));
    expect(rolledBackDetail?.status).toBe("backlog");
  });

  it("should optimistically update task status and persist on success", async () => {
    const mockTasksList = { data: [mockTask], pagination: { total: 1, page: 1, limit: 10 } };
    queryClient.setQueryData(queryKeys.tasks.list({ project_id: "proj-1" }), mockTasksList);
    queryClient.setQueryData(queryKeys.tasks.detail("task-1"), mockTask);

    let resolveApi!: (value: unknown) => void;
    const updatedTask = { ...mockTask, status: "in_progress" };
    vi.mocked(api.patch).mockImplementationOnce(() => new Promise((resolve) => {
      resolveApi = resolve;
    }));

    const { result } = renderHook(() => useUpdateTaskStatus(), { wrapper });

    act(() => {
      result.current.mutate({ id: "task-1", status: "in_progress" });
    });

    await waitFor(() => {
      const updatedList: PaginatedResponse<Task> | undefined = queryClient.getQueryData(queryKeys.tasks.list({ project_id: "proj-1" }));
      expect(updatedList?.data[0].status).toBe("in_progress");
    });

    act(() => {
      resolveApi(updatedTask);
    });

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true);
    });

    expect(api.patch).toHaveBeenCalledWith("/v1/tasks/task-1/status", { status: "in_progress" });
  });
});
