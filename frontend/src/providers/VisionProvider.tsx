"use client";

import {
  createContext,
  useContext,
  useState,
  useCallback,
  useRef,
  useEffect,
  type ReactNode,
} from "react";
import { api } from "@/lib/api";

type VisionDocument = {
  id: string;
  version: number;
  problem_statement: string;
  vision_statement: string;
};

type Revision = {
  id: string;
  version: number;
  changed_at: string;
  changed_by: string;
  change_reason: string;
  evidence_links: string[];
};

type VisionContextValue = {
  document: VisionDocument | null;
  history: Revision[];
  isDirty: boolean;
  lastSaved: Date | null;
  save: (changes: Partial<VisionDocument>, changeReason: string) => Promise<void>;
};

const VisionContext = createContext<VisionContextValue | null>(null);

export function VisionProvider({ children }: { children: ReactNode }) {
  const [document, setDocument] = useState<VisionDocument | null>(null);
  const [history, setHistory] = useState<Revision[]>([]);
  const [isDirty, setIsDirty] = useState(false);
  const [lastSaved, setLastSaved] = useState<Date | null>(null);
  const draftBuffer = useRef<string | null>(null);

  // Load vision document on mount
  useEffect(() => {
    api.get<VisionDocument>("/vision").then(setDocument).catch(() => {});
    api.get<Revision[]>("/vision/history").then(setHistory).catch(() => {});
  }, []);

  const save = useCallback(
    async (changes: Partial<VisionDocument>, changeReason: string) => {
      if (!document) return;
      const updated = await api.put<VisionDocument>("/vision", {
        ...changes,
        version: document.version,
        change_reason: changeReason,
      });
      setDocument(updated);
      setLastSaved(new Date());
      setIsDirty(false);
      // Reload history
      const revs = await api.get<Revision[]>("/vision/history");
      setHistory(revs);
    },
    [document],
  );

  return (
    <VisionContext.Provider value={{ document, history, isDirty, lastSaved, save }}>
      {children}
    </VisionContext.Provider>
  );
}

export function useVision(): VisionContextValue {
  const ctx = useContext(VisionContext);
  if (!ctx) throw new Error("useVision must be used within <VisionProvider>");
  return ctx;
}
