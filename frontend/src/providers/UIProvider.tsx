"use client";

import {
  createContext,
  useContext,
  useState,
  useCallback,
  type ReactNode,
} from "react";

type Toast = {
  id: string;
  type: "success" | "error" | "info" | "warning";
  message: string;
};

type UIContextValue = {
  sidebarCollapsed: boolean;
  setSidebarCollapsed: (collapsed: boolean) => void;
  activeModal: string | null;
  openModal: (modal: string) => void;
  closeModal: () => void;
  toasts: Toast[];
  addToast: (toast: Omit<Toast, "id">) => void;
  removeToast: (id: string) => void;
};

const UIContext = createContext<UIContextValue | null>(null);

export function UIProvider({ children }: { children: ReactNode }) {
  const [sidebarCollapsed, setSidebarCollapsed] = useState(false);
  const [activeModal, setActiveModal] = useState<string | null>(null);
  const [toasts, setToasts] = useState<Toast[]>([]);

  const openModal = useCallback((modal: string) => setActiveModal(modal), []);
  const closeModal = useCallback(() => setActiveModal(null), []);

  const addToast = useCallback((toast: Omit<Toast, "id">) => {
    const id = `toast_${crypto.randomUUID().slice(0, 8)}`;
    setToasts((prev) => [...prev, { ...toast, id }]);
    // Auto-dismiss after 3 seconds
    setTimeout(() => {
      setToasts((prev) => prev.filter((t) => t.id !== id));
    }, 3000);
  }, []);

  const removeToast = useCallback((id: string) => {
    setToasts((prev) => prev.filter((t) => t.id !== id));
  }, []);

  return (
    <UIContext.Provider
      value={{
        sidebarCollapsed,
        setSidebarCollapsed,
        activeModal,
        openModal,
        closeModal,
        toasts,
        addToast,
        removeToast,
      }}
    >
      {children}
    </UIContext.Provider>
  );
}

export function useUI(): UIContextValue {
  const ctx = useContext(UIContext);
  if (!ctx) throw new Error("useUI must be used within <UIProvider>");
  return ctx;
}
