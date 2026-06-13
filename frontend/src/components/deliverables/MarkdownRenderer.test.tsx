import { describe, expect, it } from "vitest"
import { render, screen } from "@testing-library/react"
import { MarkdownRenderer } from "@/components/deliverables/MarkdownRenderer"

/**
 * F-008 (security-review §5.1) acceptance test: deliverable markdown
 * content must be sanitized before render. The Lead's brief lists
 * the minimum bar as "inject `<script>alert('xss')</script>` in a
 * test deliverable → must NOT execute".
 *
 * We render several hostile payloads and assert the DOM contains no
 * `<script>` tags, no inline event handlers, no `javascript:` hrefs,
 * and no `<iframe>`/`<object>`/`<embed>`. The schema is documented
 * in `MarkdownRenderer.tsx` so Security-01 can lock it in for TASK-412.
 */
describe("MarkdownRenderer — F-008 sanitization", () => {
  it("strips raw <script> tags (the canonical F-008 fixture)", () => {
    const { container } = render(
      <MarkdownRenderer content="<script>alert('xss')</script>Hello" />,
    )
    // No <script> in the rendered DOM. The text "alert('xss')" can
    // appear as text content; we only forbid the element.
    expect(container.querySelector("script")).toBeNull()
    // The plain "Hello" content should survive.
    expect(container.textContent).toContain("Hello")
  })

  it("strips inline event handlers on allowed tags", () => {
    const { container } = render(
      <MarkdownRenderer
        content={'<img src="https://example.com/x.png" onerror="alert(1)" alt="x">'}
      />,
    )
    const img = container.querySelector("img")
    expect(img).not.toBeNull()
    // `onerror` is not on the attributes allowlist → must be stripped
    // by rehype-sanitize.
    expect(img?.getAttribute("onerror")).toBeNull()
    // Allowed attrs survive (src, alt).
    expect(img?.getAttribute("src")).toBe("https://example.com/x.png")
    expect(img?.getAttribute("alt")).toBe("x")
  })

  it("strips javascript: URLs from <a> hrefs", () => {
    const { container } = render(
      <MarkdownRenderer content="[click me](javascript:alert(1))" />,
    )
    const a = container.querySelector("a")
    // rehype-sanitize's default protocol list drops javascript: from
    // href. The link either renders without an href, or with a safe
    // one — never with a `javascript:` scheme.
    const href = a?.getAttribute("href") ?? ""
    expect(href.startsWith("javascript:")).toBe(false)
  })

  it("strips <iframe>, <object>, <embed> entirely", () => {
    const hostile = [
      "<iframe src='https://evil.example.com'></iframe>",
      "<object data='x.swf'></object>",
      "<embed src='x.swf'>",
    ].join("\n")
    const { container } = render(<MarkdownRenderer content={hostile} />)
    expect(container.querySelector("iframe")).toBeNull()
    expect(container.querySelector("object")).toBeNull()
    expect(container.querySelector("embed")).toBeNull()
  })

  it("strips <style> and inline style attributes", () => {
    const { container } = render(
      <MarkdownRenderer content="<style>body { color: red }</style><p style='color:red'>hi</p>" />,
    )
    expect(container.querySelector("style")).toBeNull()
    const p = container.querySelector("p")
    expect(p?.getAttribute("style")).toBeNull()
  })

  it("preserves safe GFM features (tables, task lists, autolinks)", () => {
    const md = [
      "## Heading",
      "",
      "| col1 | col2 |",
      "| ---- | ---- |",
      "| a    | b    |",
      "",
      "- [x] done",
      "- [ ] todo",
      "",
      "https://example.com",
    ].join("\n")
    const { container } = render(<MarkdownRenderer content={md} />)
    expect(container.querySelector("table")).not.toBeNull()
    expect(container.querySelector('input[type="checkbox"]')).not.toBeNull()
    // Autolinks render as <a href="https://example.com">.
    const autolink = container.querySelector('a[href="https://example.com"]')
    expect(autolink).not.toBeNull()
  })
})
