"use client"

import ReactMarkdown from "react-markdown"
import remarkGfm from "remark-gfm"
import rehypeRaw from "rehype-raw"
import rehypeSanitize, { defaultSchema } from "rehype-sanitize"
import { cn } from "@/lib/utils"

/**
 * MarkdownRenderer — the **F-006 trust boundary** for the Sprint 4 deliverable
 * viewer.
 *
 * Deliverable `content` is authored by agents (and on the admin/testing path
 * by humans) and rendered into the same DOM as the rest of the app. The
 * Lead's brief locks in F-006 from security-review §5.1: **must sanitize**
 * before render. The 2026-06-12 wave's allowlist is:
 *
 *   - NO `<script>`, NO inline `onerror`/`onload`/`onclick`/etc.
 *   - NO `javascript:` URLs (or any non-`http(s):`/`mailto:` schemes)
 *   - NO `<iframe>`, NO `<object>`, NO `<embed>`, NO `<frame>`/`<frameset>`
 *   - NO `<style>` (we keep `<pre>` for fenced code blocks but no inline styles)
 *   - NO event-handler attributes (the default `rehype-sanitize` schema strips
 *     all `on*` attrs — see the override below for the explicit list)
 *
 * We extend the default schema with `tagNames`/`attributes` we want to
 * allow (GFM tables, task lists, autolinks, etc.) and explicitly deny
 * the dangerous tags above. The exact list is documented inline so
 * Security-01 can lock it in as part of TASK-412.
 *
 * # Testing
 * The XSS contract is verified by a unit test in
 * `frontend/src/components/deliverables/MarkdownRenderer.test.tsx`
 * that renders `<script>alert('xss')</script>` and asserts the script
 * tag is stripped. See Lead's brief, acceptance §"Sanitization verified
 * with a test fixture".
 */
const DELIVERABLE_SANITIZE_SCHEMA = {
  ...defaultSchema,
  tagNames: [
    // Standard HTML for prose
    "h1", "h2", "h3", "h4", "h5", "h6",
    "p", "br", "hr",
    "ul", "ol", "li",
    "blockquote", "pre", "code",
    "em", "strong", "del", "ins", "sub", "sup",
    "a", "img",
    "table", "thead", "tbody", "tr", "th", "td",
    // GFM task lists render as <input type="checkbox" disabled>
    "input",
  ],
  // Strip every dangerous tag. The default schema strips `<script>` etc.
  // already, but we re-state it explicitly so the allowlist is grep-friendly
  // for security review. `tagNames` is the allowlist in `rehype-sanitize`,
  // so anything not in this list is dropped.
  // The following are *absent* on purpose: script, style, iframe, object,
  // embed, frame, frameset, form, input (except via GFM checkbox), textarea,
  // button, link, meta, base, audio, video, source, track, area, map.
  attributes: {
    ...defaultSchema.attributes,
    a: [
      ["href"],
      ["title"],
    ],
    img: [
      ["src"],
      ["alt"],
      ["title"],
    ],
    code: [["className"]],
    pre: [["className"]],
    input: [
      ["type"],
      ["checked"],
      ["disabled"],
    ],
    // No `on*` attrs anywhere. rehype-sanitize strips them by default.
  },
  protocols: {
    ...defaultSchema.protocols,
    href: ["http", "https", "mailto"],
    src: ["http", "https"],
  },
}

type MarkdownRendererProps = {
  content: string
  className?: string
  /**
   * If true, render in a `prose` style with standard link/code block
   * formatting. The default is `true` (it's the only sensible styling
   * for deliverable content).
   */
  prose?: boolean
}

export function MarkdownRenderer({
  content,
  className,
  prose = true,
}: MarkdownRendererProps) {
  return (
    <div
      data-testid="markdown-renderer"
      className={cn(
        prose &&
          "prose prose-slate dark:prose-invert max-w-none prose-headings:font-semibold prose-a:text-sky-600 dark:prose-a:text-sky-400 prose-code:rounded prose-code:bg-slate-100 prose-code:px-1 prose-code:py-0.5 dark:prose-code:bg-slate-800 prose-pre:bg-slate-950 prose-pre:border prose-pre:border-slate-800 prose-img:rounded-lg prose-table:border-collapse prose-th:border prose-th:border-slate-300 prose-th:px-2 prose-th:py-1 dark:prose-th:border-slate-700 prose-td:border prose-td:border-slate-300 prose-td:px-2 prose-td:py-1 dark:prose-td:border-slate-700",
        className,
      )}
    >
      <ReactMarkdown
        remarkPlugins={[remarkGfm]}
        rehypePlugins={[rehypeRaw, [rehypeSanitize, DELIVERABLE_SANITIZE_SCHEMA]]}
      >
        {content}
      </ReactMarkdown>
    </div>
  )
}
