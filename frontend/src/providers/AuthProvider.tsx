"use client";

import {
  createContext,
  useContext,
  useState,
  useCallback,
  useEffect,
  type ReactNode,
} from "react";
import { api, setAccessToken } from "@/lib/api";

type User = {
  id: string;
  name: string;
  email: string;
  role: string;
};

type AuthContextValue = {
  user: User | null;
  isLoading: boolean;
  error: string | null;
  login: (email: string, password: string) => Promise<void>;
  logout: () => void;
  hasPermission: (requiredRole: string) => boolean;
};

const AuthContext = createContext<AuthContextValue | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Attempt silent refresh on mount
  useEffect(() => {
    api
      .get<{ user: User; access_token: string }>("/auth/me")
      .then((data) => {
        setUser(data.user);
        setAccessToken(data.access_token);
      })
      .catch(() => {
        setUser(null);
        setAccessToken(null);
      })
      .finally(() => setIsLoading(false));
  }, []);

  const login = useCallback(async (email: string, password: string) => {
    setError(null);
    setIsLoading(true);
    try {
      const data = await api.post<{ user: User; access_token: string }>("/auth/login", {
        email,
        password,
      });
      setUser(data.user);
      setAccessToken(data.access_token);
    } catch (err) {
      const message = err instanceof Error ? err.message : "Login failed";
      setError(message);
      throw err;
    } finally {
      setIsLoading(false);
    }
  }, []);

  const logout = useCallback(() => {
    setUser(null);
    setAccessToken(null);
  }, []);

  const hasPermission = useCallback(
    (requiredRole: string) => {
      if (!user) return false;
      const hierarchy: Record<string, number> = {
        viewer: 0,
        analyst: 1,
        designer: 2,
        qa: 3,
        security: 3,
        engineering_lead: 4,
        product_manager: 5,
        admin: 6,
      };
      return (hierarchy[user.role] ?? 0) >= (hierarchy[requiredRole] ?? 0);
    },
    [user],
  );

  return (
    <AuthContext.Provider value={{ user, isLoading, error, login, logout, hasPermission }}>
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth(): AuthContextValue {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error("useAuth must be used within <AuthProvider>");
  return ctx;
}
