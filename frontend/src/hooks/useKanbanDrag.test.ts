import { renderHook, act } from "@testing-library/react";
import { describe, it, expect } from "vitest";
import { useKanbanDrag } from "./useKanbanDrag";

describe("useKanbanDrag", () => {
  it("should initialize with no active drag", () => {
    const { result } = renderHook(() => useKanbanDrag());
    expect(result.current.activeDrag).toBeNull();
  });

  it("should set active drag item when startDrag is called", () => {
    const { result } = renderHook(() => useKanbanDrag());

    act(() => {
      result.current.startDrag({ id: "task-1", column: "todo" });
    });

    expect(result.current.activeDrag).toEqual({ id: "task-1", column: "todo" });
  });

  it("should clear active drag item when endDrag is called", () => {
    const { result } = renderHook(() => useKanbanDrag());

    act(() => {
      result.current.startDrag({ id: "task-1", column: "todo" });
    });
    expect(result.current.activeDrag).toEqual({ id: "task-1", column: "todo" });

    act(() => {
      result.current.endDrag();
    });
    expect(result.current.activeDrag).toBeNull();
  });
});
