import '@testing-library/jest-dom/vitest';
import { vi } from 'vitest';

// Mock next/navigation so hooks that read the projectId from the URL
// (e.g. `useProjectFilters`) can be invoked inside renderHook without
// requiring a real app-router wrapper. The default projectId is empty;
// tests that need one should pass it via the URL helpers.
vi.mock('next/navigation', () => ({
  useSearchParams: () => new URLSearchParams(),
  useRouter: () => ({
    push: vi.fn(),
    replace: vi.fn(),
    back: vi.fn(),
    forward: vi.fn(),
    refresh: vi.fn(),
    prefetch: vi.fn(),
  }),
  usePathname: () => '/',
  useParams: () => ({}),
}));
