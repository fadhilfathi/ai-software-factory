import { renderHook, act, waitFor } from "@testing-library/react";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { useUpdateTask } from "./hooks";
import { api } from "./api";
import { queryKeys } from "./queryKeys";
import { ReactNode } from "react";
import type { Task } from "./types";

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
  status: "todo",
  priority: "medium",
  created_at: new Date().toISOString(),
  updated_at: new Date().toISOString(),
};

describe("useUpdateTask", () => {
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

    let rejectApi!: (reason?: any) => void;
    (api.patch as any).mockImplementationOnce(() => new Promise((_, reject) => {
      rejectApi = reject;
    }));

    const { result } = renderHook(() => useUpdateTask(), { wrapper });

    act(() => {
      result.current.mutate({ id: "task-1", status: "in_progress" });
    });

    // Check optimistic update in cache immediately
    await waitFor(() => {
      const updatedList: any = queryClient.getQueryData(queryKeys.tasks.list({ project_id: "proj-1" }));
      expect(updatedList.data[0].status).toBe("in_progress");

      const updatedDetail: any = queryClient.getQueryData(queryKeys.tasks.detail("task-1"));
      expect(updatedDetail.status).toBe("in_progress");
    });

    // Now reject the API
    act(() => {
      rejectApi(new Error("API Error"));
    });

    // Wait for the mutation to fail and rollback
    await waitFor(() => {
      expect(result.current.isError).toBe(true);
    });

    const rolledBackList: any = queryClient.getQueryData(queryKeys.tasks.list({ project_id: "proj-1" }));
    expect(rolledBackList.data[0].status).toBe("todo");

    const rolledBackDetail: any = queryClient.getQueryData(queryKeys.tasks.detail("task-1"));
    expect(rolledBackDetail.status).toBe("todo");
  });

  it("should optimistically update task status and persist on success", async () => {
    const mockTasksList = { data: [mockTask], pagination: { total: 1, page: 1, limit: 10 } };
    queryClient.setQueryData(queryKeys.tasks.list({ project_id: "proj-1" }), mockTasksList);
    queryClient.setQueryData(queryKeys.tasks.detail("task-1"), mockTask);

    let resolveApi!: (value: any) => void;
    const updatedTask = { ...mockTask, status: "in_progress" };
    (api.patch as any).mockImplementationOnce(() => new Promise((resolve) => {
      resolveApi = resolve;
    }));

    const { result } = renderHook(() => useUpdateTask(), { wrapper });

    act(() => {
      result.current.mutate({ id: "task-1", status: "in_progress" });
    });

    // Check optimistic update
    await waitFor(() => {
      const updatedList: any = queryClient.getQueryData(queryKeys.tasks.list({ project_id: "proj-1" }));
      expect(updatedList.data[0].status).toBe("in_progress");
    });

    // Now resolve the API
    act(() => {
      resolveApi(updatedTask);
    });

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true);
    });

    expect(api.patch).toHaveBeenCalledWith("/v1/tasks/task-1", { status: "in_progress" });
  });
});
