"use client";

import { useState, useCallback } from "react";

type DragItem = {
  id: string;
  column: string;
};

/**
 * Hook for managing kanban drag-and-drop state.
 * Actual DnD library integration (e.g. @dnd-kit) is handled at the component level.
 * This hook manages the logical state — active drag item and column transitions.
 */
export function useKanbanDrag() {
  const [activeDrag, setActiveDrag] = useState<DragItem | null>(null);

  const startDrag = useCallback((item: DragItem) => {
    setActiveDrag(item);
  }, []);

  const endDrag = useCallback(() => {
    setActiveDrag(null);
  }, []);

  return { activeDrag, startDrag, endDrag };
}
