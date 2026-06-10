# AI Software Factory — Design System

## Overview

This document defines the visual language, component specifications, and interaction patterns for the AI Software Factory platform. The design system is built for **Tailwind CSS** with a token-based approach that ensures consistency across all UI surfaces.

**Design Principles:**
1. **Clarity over decoration** — Every element communicates status or action
2. **Agent-forward** — Visual language distinguishes human from AI activity
3. **Density with breathing room** — Data-rich dashboards that don't overwhelm
4. **Accessible by default** — WCAG 2.2 AA minimum, AAA for text contrast

---

## 1. Color Palette

### 1.1 Brand Colors

| Token | Hex | Usage |
|-------|-----|-------|
| `brand-500` | `#6366F1` | Primary brand, CTA buttons, active states |
| `brand-600` | `#4F46E5` | Hover state for primary actions |
| `brand-700` | `#4338CA` | Pressed/active state |
| `brand-400` | `#818CF8` | Light accent, selected items |
| `brand-100` | `#E0E7FF` | Brand tint backgrounds |

### 1.2 Neutral Palette

| Token | Hex | Usage |
|-------|-----|-------|
| `gray-950` | `#0A0A0B` | Dark mode background, headers |
| `gray-900` | `#18181B` | Card backgrounds (dark), sidebar |
| `gray-800` | `#27272A` | Elevated surfaces (dark), borders |
| `gray-700` | `#3F3F46` | Subtle borders, dividers |
| `gray-600` | `#52525B` | Secondary text (dark mode) |
| `gray-500` | `#71717A` | Placeholder text, icons |
| `gray-400` | `#A1A1AA` | Muted text, disabled elements |
| `gray-300` | `#D4D4D8` | Body text (dark mode) |
| `gray-200` | `#E4E4E7` | Borders (light mode) |
| `gray-100` | `#F4F4F5` | Background surfaces (light mode) |
| `gray-50` | `#FAFAFA` | Page background (light mode) |
| `white` | `#FFFFFF` | Card backgrounds (light mode), text on dark |

### 1.3 Semantic Colors

| Token | Hex | Usage |
|-------|-----|-------|
| `success-500` | `#22C55E` | Approved, passed, deployed, online |
| `success-100` | `#DCFCE7` | Success background (light) |
| `success-800` | `#166534` | Success text (dark background) |
| `warning-500` | `#F59E0B` | Needs attention, in-progress, queued |
| `warning-100` | `#FEF3C7` | Warning background (light) |
| `warning-800` | `#92400E` | Warning text (dark background) |
| `error-500` | `#EF4444` | Failed, blocked, critical alert |
| `error-100` | `#FEE2E2` | Error background (light) |
| `error-800` | `#991B1B` | Error text (dark background) |
| `info-500` | `#3B82F6` | Informational, links, neutral actions |
| `info-100` | `#DBEAFE` | Info background (light) |
| `info-800` | `#1E3A5F` | Info text (dark background) |

### 1.4 Agent-Specific Colors

Each agent type has a distinct color for quick visual identification in dashboards, timelines, and activity feeds.

| Agent | Token | Hex | Icon Background |
|-------|-------|-----|-----------------|
| PM Agent | `agent-pm` | `#8B5CF6` | `bg-agent-pm/10 text-agent-pm` |
| Architect Agent | `agent-architect` | `#0EA5E9` | `bg-agent-architect/10 text-agent-architect` |
| Developer Agent | `agent-developer` | `#10B981` | `bg-agent-developer/10 text-agent-developer` |
| Review Agent | `agent-review` | `#F97316` | `bg-agent-review/10 text-agent-review` |
| QA Agent | `agent-qa` | `#EC4899` | `bg-agent-qa/10 text-agent-qa` |
| DevOps Agent | `agent-devops` | `#6366F1` | `bg-agent-devops/10 text-agent-devops` |

### 1.5 Tailwind Configuration Extension

```js
// tailwind.config.js — extend colors
module.exports = {
  theme: {
    extend: {
      colors: {
        brand: {
          100: '#E0E7FF',
          400: '#818CF8',
          500: '#6366F1',
          600: '#4F46E5',
          700: '#4338CA',
        },
        agent: {
          pm: '#8B5CF6',
          architect: '#0EA5E9',
          developer: '#10B981',
          review: '#F97316',
          qa: '#EC4899',
          devops: '#6366F1',
        },
      },
    },
  },
}
```

---

## 2. Typography

### 2.1 Font Stack

| Token | Font | Fallback | Usage |
|-------|------|----------|-------|
| `font-sans` | Inter | system-ui, -apple-system, sans-serif | Body text, UI elements |
| `font-mono` | JetBrains Mono | ui-monospace, monospace | Code blocks, terminal output, commit SHAs |

**Load via:** `@import url('https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700&family=JetBrains+Mono:wght@400;500&display=swap');`

### 2.2 Type Scale

| Token | Size | Weight | Line Height | Tracking | Usage |
|-------|------|--------|-------------|----------|-------|
| `text-display` | 36px / 2.25rem | 700 | 1.2 | -0.025em | Page titles (rare, dashboard headers) |
| `text-h1` | 30px / 1.875rem | 700 | 1.33 | -0.02em | Section headings |
| `text-h2` | 24px / 1.5rem | 600 | 1.35 | -0.015em | Card titles, subsection headings |
| `text-h3` | 20px / 1.25rem | 600 | 1.4 | -0.01em | Group labels, agent names |
| `text-body` | 14px / 0.875rem | 400 | 1.6 | 0 | Default body text, descriptions |
| `text-body-sm` | 13px / 0.8125rem | 400 | 1.55 | 0 | Secondary text, metadata |
| `text-caption` | 12px / 0.75rem | 500 | 1.5 | 0.01em | Labels, timestamps, badges |
| `text-mono` | 13px / 0.8125rem | 400 | 1.6 | 0 | Code, technical values, IDs |

### 2.3 Tailwind Classes Reference

```html
<!-- Display / Page Title -->
<h1 class="text-display font-bold text-gray-900 dark:text-white">Project Dashboard</h1>

<!-- H1 / Section -->
<h1 class="text-3xl font-bold text-gray-900 dark:text-white">Active Projects</h1>

<!-- H2 / Card Title -->
<h2 class="text-2xl font-semibold text-gray-900 dark:text-white">Authentication Service</h2>

<!-- H3 / Group Label -->
<h3 class="text-xl font-semibold text-gray-900 dark:text-white">Agent Activity</h3>

<!-- Body -->
<p class="text-sm text-gray-700 dark:text-gray-300">The PM Agent is decomposing requirements...</p>

<!-- Body Small -->
<span class="text-[13px] text-gray-500 dark:text-gray-400">Last updated 2 minutes ago</span>

<!-- Caption / Label -->
<time class="text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">2026-06-10 14:32</time>

<!-- Mono / Code -->
<code class="text-[13px] font-mono text-brand-500">task_abc123</code>
```

### 2.4 Font Scale Tailwind Config

```js
module.exports = {
  theme: {
    extend: {
      fontSize: {
        'display': ['2.25rem', { lineHeight: '1.2', letterSpacing: '-0.025em', fontWeight: '700' }],
        'h1': ['1.875rem', { lineHeight: '1.33', letterSpacing: '-0.02em', fontWeight: '700' }],
        'h2': ['1.5rem', { lineHeight: '1.35', letterSpacing: '-0.015em', fontWeight: '600' }],
        'h3': ['1.25rem', { lineHeight: '1.4', letterSpacing: '-0.01em', fontWeight: '600' }],
        'body': ['0.875rem', { lineHeight: '1.6', fontWeight: '400' }],
        'body-sm': ['0.8125rem', { lineHeight: '1.55', fontWeight: '400' }],
        'caption': ['0.75rem', { lineHeight: '1.5', letterSpacing: '0.01em', fontWeight: '500' }],
      },
      fontFamily: {
        sans: ['Inter', 'system-ui', '-apple-system', 'sans-serif'],
        mono: ['JetBrains Mono', 'ui-monospace', 'monospace'],
      },
    },
  },
}
```

---

## 3. Spacing System

### 3.1 Base Unit

The spacing scale is based on a **4px base unit**. All spacing values are multiples of 4px to create consistent visual rhythm.

| Token | Value | Tailwind | Usage |
|-------|-------|----------|-------|
| `space-0` | 0px | `p-0` / `m-0` | Reset |
| `space-0.5` | 2px | `p-0.5` / `m-0.5` | Inline icon padding |
| `space-1` | 4px | `p-1` / `m-1` | Tight internal padding, icon gaps |
| `space-1.5` | 6px | `p-1.5` / `m-1.5` | Small badge padding |
| `space-2` | 8px | `p-2` / `m-2` | Button padding (y), small card padding |
| `space-3` | 12px | `p-3` / `m-3` | Card padding, input padding |
| `space-4` | 16px | `p-4` / `m-4` | Standard gap between related elements |
| `space-5` | 20px | `p-5` / `m-5` | Medium card padding |
| `space-6` | 24px | `p-6` / `m-6` | Section padding, large card padding |
| `space-8` | 32px | `p-8` / `m-8` | Page section spacing |
| `space-10` | 40px | `p-10` / `m-10` | Major section separation |
| `space-12` | 48px | `p-12` / `m-12` | Page-level vertical spacing |
| `space-16` | 64px | `p-16` / `m-16` | Hero/banner spacing |

### 3.2 Layout Spacing

| Context | Value | Tailwind |
|---------|-------|----------|
| Page padding (mobile) | 16px | `p-4` |
| Page padding (desktop) | 24-32px | `p-6` to `p-8` |
| Card internal padding | 16-24px | `p-4` to `p-6` |
| Gap between cards (grid) | 16px | `gap-4` |
| Gap between sections | 32-48px | `gap-8` to `gap-12` |
| Sidebar width | 240-280px | `w-60` to `w-72` |
| Content max-width | 1200px | `max-w-screen-xl` |

### 3.3 Component Spacing Patterns

```html
<!-- Card -->
<div class="p-6 space-y-4">
  <!-- Card Header: icon + title + badge -->
  <div class="flex items-center gap-3">
    <span class="p-2 rounded-lg bg-brand-100 dark:bg-brand-500/10">
      <Icon class="w-5 h-5 text-brand-500" />
    </span>
    <h3 class="text-h3 text-gray-900 dark:text-white">Agent Tasks</h3>
    <span class="ml-auto px-2 py-0.5 text-caption bg-success-100 text-success-800 rounded-full">12 active</span>
  </div>
  <!-- Card Body -->
  <p class="text-body text-gray-500 dark:text-gray-400">Tasks being processed...</p>
</div>

<!-- List Item -->
<div class="flex items-center gap-4 px-4 py-3 hover:bg-gray-50 dark:hover:bg-gray-800">
  <StatusDot />
  <div class="flex-1 min-w-0">
    <p class="text-body font-medium text-gray-900 dark:text-white truncate">Implement JWT auth</p>
    <p class="text-caption text-gray-500 dark:text-gray-400">Developer Agent • 12 min ago</p>
  </div>
  <Badge status="in-progress" />
</div>
```

---

## 4. Component Library

### 4.1 Buttons

#### Primary Button
- **Default:** `bg-brand-500 text-white hover:bg-brand-600 active:bg-brand-700`
- **Size (default):** `h-10 px-4 py-2 text-body font-medium rounded-lg`
- **Size (sm):** `h-8 px-3 py-1.5 text-caption font-medium rounded-md`
- **Size (lg):** `h-12 px-6 py-3 text-body font-semibold rounded-lg`
- **Focus ring:** `focus-visible:ring-2 focus-visible:ring-brand-500 focus-visible:ring-offset-2`
- **Disabled:** `opacity-50 cursor-not-allowed pointer-events-none`
- **Loading:** Spinner icon replaces label, button width preserved

#### Secondary Button
- **Default:** `border border-gray-200 dark:border-gray-700 text-gray-700 dark:text-gray-300 hover:bg-gray-50 dark:hover:bg-gray-800`

#### Ghost Button
- **Default:** `text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-white hover:bg-gray-100 dark:hover:bg-gray-800`

#### Danger Button
- **Default:** `bg-error-500 text-white hover:bg-red-600 active:bg-red-700`

```html
<!-- Primary -->
<button class="inline-flex items-center justify-center gap-2 h-10 px-4 py-2 text-sm font-medium rounded-lg
  bg-brand-500 text-white hover:bg-brand-600 active:bg-brand-700
  focus-visible:ring-2 focus-visible:ring-brand-500 focus-visible:ring-offset-2
  disabled:opacity-50 disabled:cursor-not-allowed disabled:pointer-events-none
  transition-colors">
  <PlusIcon class="w-4 h-4" />
  New Project
</button>

<!-- Secondary -->
<button class="inline-flex items-center justify-center gap-2 h-10 px-4 py-2 text-sm font-medium rounded-lg
  border border-gray-200 dark:border-gray-700 text-gray-700 dark:text-gray-300
  hover:bg-gray-50 dark:hover:bg-gray-800
  focus-visible:ring-2 focus-visible:ring-brand-500 focus-visible:ring-offset-2
  transition-colors">
  Cancel
</button>
```

#### Interaction States Summary

| State | Visual Change |
|-------|--------------|
| Default | Base colors |
| Hover | Darken background by 1 step |
| Focus | 2px brand ring with 2px offset |
| Active/Pressed | Darken background by 2 steps |
| Disabled | 50% opacity, cursor not-allowed |
| Loading | Spinner replaces label, fixed width |

---

### 4.2 Input Fields

#### Text Input
- **Height:** `h-10` (40px)
- **Padding:** `px-3 py-2`
- **Border:** `border border-gray-200 dark:border-gray-700`
- **Background:** `bg-white dark:bg-gray-900`
- **Text:** `text-body text-gray-900 dark:text-white`
- **Placeholder:** `placeholder-gray-400 dark:placeholder-gray-500`
- **Border radius:** `rounded-lg`
- **Focus:** `ring-2 ring-brand-500 ring-offset-0 border-brand-500`
- **Error:** `border-error-500 ring-2 ring-error-500 ring-offset-0`
- **Disabled:** `bg-gray-50 dark:bg-gray-800 opacity-60 cursor-not-allowed`

#### Textarea
- Same styling as text input
- **Min height:** `min-h-[100px]`
- **Resize:** `resize-y`

#### Select
- Same styling as text input
- **Chevron:** Custom dropdown arrow icon, `pointer-events-none`

```html
<!-- Text Input -->
<div class="space-y-1.5">
  <label class="text-caption font-medium text-gray-700 dark:text-gray-300">
    Project Name
  </label>
  <input
    type="text"
    placeholder="e.g. User Authentication Service"
    class="w-full h-10 px-3 py-2 text-sm rounded-lg
      bg-white dark:bg-gray-900
      border border-gray-200 dark:border-gray-700
      text-gray-900 dark:text-white
      placeholder-gray-400 dark:placeholder-gray-500
      focus:ring-2 focus:ring-brand-500 focus:border-brand-500
      disabled:bg-gray-50 dark:disabled:bg-gray-800 disabled:opacity-60 disabled:cursor-not-allowed
      transition-colors" />
</div>

<!-- Error State -->
<div class="space-y-1.5">
  <label class="text-caption font-medium text-gray-700 dark:text-gray-300">
    API Key
  </label>
  <input
    type="text"
    class="w-full h-10 px-3 py-2 text-sm rounded-lg
      border border-error-500 ring-2 ring-error-500 ring-offset-0
      bg-white dark:bg-gray-900 text-gray-900 dark:text-white" />
  <p class="text-caption text-error-500">API key is required for agent deployment.</p>
</div>
```

---

### 4.3 Cards

#### Standard Card
- **Background:** `bg-white dark:bg-gray-900`
- **Border:** `border border-gray-200 dark:border-gray-800`
- **Border radius:** `rounded-xl`
- **Padding:** `p-6`
- **Shadow:** `shadow-sm` (light mode only)
- **Hover (interactive):** `hover:border-gray-300 dark:hover:border-gray-700 hover:shadow-md`

#### Card with Agent Status
- **Left accent border:** 3px colored border matching agent color

```html
<!-- Standard Card -->
<div class="bg-white dark:bg-gray-900 border border-gray-200 dark:border-gray-800
  rounded-xl p-6 shadow-sm">
  <h3 class="text-h3 text-gray-900 dark:text-white mb-2">Project Overview</h3>
  <p class="text-body text-gray-500 dark:text-gray-400">...</p>
</div>

<!-- Agent Activity Card -->
<div class="bg-white dark:bg-gray-900 border border-gray-200 dark:border-gray-800
  rounded-xl p-6 shadow-sm border-l-3 border-l-agent-developer">
  <div class="flex items-center gap-3 mb-3">
    <div class="w-8 h-8 rounded-full bg-agent-developer/10 flex items-center justify-center">
      <CodeIcon class="w-4 h-4 text-agent-developer" />
    </div>
    <div>
      <p class="text-body font-medium text-gray-900 dark:text-white">Developer Agent</p>
      <p class="text-caption text-gray-500 dark:text-gray-400">Writing auth.ts • 3 min ago</p>
    </div>
    <Badge variant="in-progress" class="ml-auto" />
  </div>
  <p class="text-body-sm text-gray-600 dark:text-gray-400">
    Implementing JWT token validation with refresh token rotation...
  </p>
</div>
```

---

### 4.4 Status Badges

| Status | Background | Text | Dot Color | Usage |
|--------|-----------|------|-----------|-------|
| `completed` | `bg-success-100 dark:bg-success-500/10` | `text-success-800 dark:text-success-500` | `bg-success-500` | Done, approved, deployed |
| `in-progress` | `bg-info-100 dark:bg-info-500/10` | `text-info-800 dark:text-info-500` | `bg-info-500` | Active, running |
| `queued` | `bg-warning-100 dark:bg-warning-500/10` | `text-warning-800 dark:text-warning-500` | `bg-warning-500` | Waiting, pending |
| `failed` | `bg-error-100 dark:bg-error-500/10` | `text-error-800 dark:text-error-500` | `bg-error-500` | Error, blocked |
| `pending` | `bg-gray-100 dark:bg-gray-800` | `text-gray-600 dark:text-gray-400` | `bg-gray-400` | Not started |

```html
<!-- Badge Component -->
<span class="inline-flex items-center gap-1.5 px-2 py-0.5 text-caption font-medium rounded-full
  bg-success-100 dark:bg-success-500/10 text-success-800 dark:text-success-500">
  <span class="w-1.5 h-1.5 rounded-full bg-success-500"></span>
  Deployed
</span>

<span class="inline-flex items-center gap-1.5 px-2 py-0.5 text-caption font-medium rounded-full
  bg-info-100 dark:bg-info-500/10 text-info-800 dark:text-info-500">
  <span class="w-1.5 h-1.5 rounded-full bg-info-500 animate-pulse"></span>
  Running
</span>
```

---

### 4.5 Navigation / Sidebar

- **Width:** 260px (`w-65`)
- **Background:** `bg-gray-950 dark:bg-gray-950`
- **Item height:** `h-10`
- **Item padding:** `px-3 py-2`
- **Item border radius:** `rounded-lg`
- **Item text:** `text-body-sm text-gray-400`
- **Item hover:** `hover:bg-gray-800 hover:text-gray-200`
- **Item active:** `bg-brand-500/10 text-brand-400`
- **Section label:** `text-caption uppercase tracking-wider text-gray-500 px-3 py-2`
- **Gap between items:** `gap-0.5` (2px)
- **Gap between sections:** `mt-6`

```html
<aside class="w-65 h-screen bg-gray-950 flex flex-col p-4">
  <!-- Logo -->
  <div class="flex items-center gap-2 px-3 mb-8">
    <Logo class="w-8 h-8 text-brand-500" />
    <span class="text-h3 text-white font-semibold">AI Factory</span>
  </div>

  <!-- Navigation -->
  <nav class="flex-1 space-y-1">
    <span class="block text-xs font-medium uppercase tracking-wider text-gray-500 px-3 py-2">Overview</span>
    <a href="/dashboard" class="flex items-center gap-3 h-10 px-3 py-2 rounded-lg text-sm
      text-gray-400 hover:bg-gray-800 hover:text-gray-200
      bg-brand-500/10 text-brand-400 transition-colors">
      <DashboardIcon class="w-5 h-5" />
      Dashboard
    </a>
    <a href="/projects" class="flex items-center gap-3 h-10 px-3 py-2 rounded-lg text-sm
      text-gray-400 hover:bg-gray-800 hover:text-gray-200 transition-colors">
      <FolderIcon class="w-5 h-5" />
      Projects
    </a>
  </nav>
</aside>
```

---

### 4.6 Data Table

- **Header row:** `bg-gray-50 dark:bg-gray-800/50 border-b border-gray-200 dark:border-gray-700`
- **Header text:** `text-caption uppercase tracking-wider text-gray-500 dark:text-gray-400 font-medium`
- **Body row:** `border-b border-gray-100 dark:border-gray-800 hover:bg-gray-50 dark:hover:bg-gray-800/50`
- **Cell padding:** `px-4 py-3`
- **Cell text:** `text-body text-gray-700 dark:text-gray-300`
- **Cell mono:** `text-mono text-gray-600 dark:text-gray-400` (for IDs, timestamps)
- **Empty state:** `py-12 text-center text-body text-gray-400`

```html
<div class="bg-white dark:bg-gray-900 border border-gray-200 dark:border-gray-800 rounded-xl overflow-hidden">
  <table class="w-full">
    <thead>
      <tr class="bg-gray-50 dark:bg-gray-800/50 border-b border-gray-200 dark:border-gray-700">
        <th class="px-4 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500">Task</th>
        <th class="px-4 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500">Agent</th>
        <th class="px-4 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500">Status</th>
        <th class="px-4 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500">Duration</th>
      </tr>
    </thead>
    <tbody class="divide-y divide-gray-100 dark:divide-gray-800">
      <tr class="hover:bg-gray-50 dark:hover:bg-gray-800/50 transition-colors">
        <td class="px-4 py-3 text-sm text-gray-700 dark:text-gray-300">Implement JWT auth</td>
        <td class="px-4 py-3"><AgentBadge type="developer" /></td>
        <td class="px-4 py-3"><Badge status="in-progress" /></td>
        <td class="px-4 py-3 text-xs font-mono text-gray-500">12m 34s</td>
      </tr>
    </tbody>
  </table>
</div>
```

---

### 4.7 Progress / Status Indicators

#### Linear Progress Bar
- **Track:** `h-2 bg-gray-200 dark:bg-gray-700 rounded-full`
- **Fill:** `h-2 rounded-full transition-all duration-500`
- **Fill colors:** `bg-brand-500` (default), `bg-success-500` (complete), `bg-error-500` (error)

#### Agent Status Indicator
- **Dot size:** `w-2 h-2`
- **Active:** `bg-success-500 animate-pulse`
- **Idle:** `bg-gray-400`
- **Failed:** `bg-error-500`

```html
<!-- Linear Progress -->
<div class="w-full space-y-1.5">
  <div class="flex justify-between">
    <span class="text-caption text-gray-700 dark:text-gray-300">Project Progress</span>
    <span class="text-caption font-medium text-gray-500">68%</span>
  </div>
  <div class="h-2 bg-gray-200 dark:bg-gray-700 rounded-full overflow-hidden">
    <div class="h-2 bg-brand-500 rounded-full transition-all duration-500" style="width: 68%"></div>
  </div>
</div>

<!-- Agent Activity Dot -->
<div class="flex items-center gap-2">
  <span class="relative flex h-2.5 w-2.5">
    <span class="animate-ping absolute inline-flex h-full w-full rounded-full bg-success-500 opacity-75"></span>
    <span class="relative inline-flex rounded-full h-2.5 w-2.5 bg-success-500"></span>
  </span>
  <span class="text-body-sm text-gray-600 dark:text-gray-400">3 agents active</span>
</div>
```

---

### 4.8 Alerts / Toasts

- **Border radius:** `rounded-xl`
- **Padding:** `p-4`
- **Border:** Left 4px accent border
- **Icon:** 20x20, left of text
- **Close button:** 20x20, top-right

| Type | Background | Border | Icon Color |
|------|-----------|--------|------------|
| Success | `bg-success-50 dark:bg-success-500/5` | `border-success-500` | `text-success-500` |
| Warning | `bg-warning-50 dark:bg-warning-500/5` | `border-warning-500` | `text-warning-500` |
| Error | `bg-error-50 dark:bg-error-500/5` | `border-error-500` | `text-error-500` |
| Info | `bg-info-50 dark:bg-info-500/5` | `border-info-500` | `text-info-500` |

```html
<!-- Success Toast -->
<div class="flex items-start gap-3 p-4 bg-success-50 dark:bg-success-500/5
  border-l-4 border-success-500 rounded-xl" role="alert">
  <CheckCircleIcon class="w-5 h-5 text-success-500 mt-0.5 shrink-0" />
  <div class="flex-1">
    <p class="text-body font-medium text-success-800 dark:text-success-500">Deployment Successful</p>
    <p class="text-body-sm text-success-800/70 dark:text-success-500/70 mt-0.5">
      v2.1.0 deployed to production. Health checks passing.
    </p>
  </div>
  <button class="p-1 text-success-800/50 hover:text-success-800 dark:text-success-500/50 dark:hover:text-success-500">
    <XMarkIcon class="w-4 h-4" />
  </button>
</div>
```

---

### 4.9 Code Block / Terminal Output

- **Font:** `font-mono` (JetBrains Mono)
- **Size:** `text-mono` (13px / 0.8125rem)
- **Background:** `bg-gray-950`
- **Text:** `text-gray-300`
- **Padding:** `p-4`
- **Border radius:** `rounded-xl`
- **Line numbers:** `text-gray-600 select-none`
- **Keywords:** `text-brand-400` (purple)
- **Strings:** `text-success-400` (green)
- **Comments:** `text-gray-600`
- **Copy button:** Top-right, ghost style

```html
<div class="relative bg-gray-950 rounded-xl overflow-hidden">
  <div class="flex items-center justify-between px-4 py-2 border-b border-gray-800">
    <span class="text-caption text-gray-500 font-mono">agent-message.json</span>
    <button class="text-gray-500 hover:text-gray-300 transition-colors">
      <CopyIcon class="w-4 h-4" />
    </button>
  </div>
  <pre class="p-4 overflow-x-auto text-sm font-mono leading-relaxed"><code><span class="text-gray-600">1</span>  {
<span class="text-gray-600">2</span>    <span class="text-brand-400">"type"</span>: <span class="text-success-400">"task_complete"</span>,
<span class="text-gray-600">3</span>    <span class="text-brand-400">"from_agent"</span>: <span class="text-success-400">"agent_dev_001"</span>,
<span class="text-gray-600">4</span>    <span class="text-brand-400">"payload"</span>: {
<span class="text-gray-600">5</span>      <span class="text-brand-400">"files_changed"</span>: [<span class="text-success-400">"src/auth/login.ts"</span>]
<span class="text-gray-600">6</span>    }
<span class="text-gray-600">7</span>  }</code></pre>
</div>
```

---

### 4.10 Modal / Dialog

- **Overlay:** `bg-black/50 backdrop-blur-sm`
- **Container:** `bg-white dark:bg-gray-900 rounded-2xl shadow-2xl`
- **Max width:** `max-w-lg` (512px) default, `max-w-xl` (576px) for forms
- **Padding:** `p-6`
- **Header margin:** `mb-4`
- **Footer margin:** `mt-6`, `pt-4 border-t border-gray-200 dark:border-gray-800`
- **Footer alignment:** `flex justify-end gap-3`
- **Close button:** Top-right, ghost style, `w-8 h-8`
- **Animation:** `animate-in fade-in zoom-in-95 duration-200`

```html
<!-- Modal Backdrop -->
<div class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 backdrop-blur-sm">
  <div class="w-full max-w-lg bg-white dark:bg-gray-900 rounded-2xl shadow-2xl p-6 mx-4
    animate-in fade-in zoom-in-95 duration-200">
    <!-- Header -->
    <div class="flex items-center justify-between mb-4">
      <h2 class="text-h2 text-gray-900 dark:text-white">Create New Project</h2>
      <button class="p-1.5 rounded-lg text-gray-400 hover:text-gray-600 dark:hover:text-gray-200 hover:bg-gray-100 dark:hover:bg-gray-800">
        <XMarkIcon class="w-5 h-5" />
      </button>
    </div>
    <!-- Body -->
    <div class="space-y-4">
      <!-- Form fields here -->
    </div>
    <!-- Footer -->
    <div class="flex justify-end gap-3 mt-6 pt-4 border-t border-gray-200 dark:border-gray-800">
      <button class="secondary-button">Cancel</button>
      <button class="primary-button">Create Project</button>
    </div>
  </div>
</div>
```

---

## 5. Responsive Breakpoints

| Breakpoint | Width | Tailwind Prefix | Layout |
|------------|-------|-----------------|--------|
| Mobile | < 640px | `sm:` | Single column, collapsed sidebar |
| Tablet | 640-1024px | `md:` | 2-column grid, collapsed sidebar |
| Desktop | 1024-1280px | `lg:` | Full layout with sidebar |
| Wide | > 1280px | `xl:` | Expanded content area |

### Responsive Patterns

| Element | Mobile | Tablet | Desktop |
|---------|--------|--------|---------|
| Sidebar | Hidden (hamburger) | Icon-only (64px) | Full (260px) |
| Cards | Full width, stacked | 2-column grid | 3-column grid |
| Tables | Horizontal scroll | Horizontal scroll | Full width |
| Page padding | 16px | 24px | 32px |
| Modals | Full width bottom sheet | Centered | Centered |

---

## 6. Dark Mode

Dark mode is implemented via Tailwind's `dark:` variant using the `class` strategy.

```js
// tailwind.config.js
module.exports = {
  darkMode: 'class',
}
```

### Dark Mode Token Mapping

| Light Mode | Dark Mode |
|-----------|-----------|
| `bg-white` | `dark:bg-gray-900` |
| `bg-gray-50` | `dark:bg-gray-950` |
| `text-gray-900` | `dark:text-white` |
| `text-gray-700` | `dark:text-gray-300` |
| `text-gray-500` | `dark:text-gray-400` |
| `border-gray-200` | `dark:border-gray-800` |
| `border-gray-100` | `dark:border-gray-800/50` |
| `shadow-sm` | (removed in dark) |
| Semantic bg-100 | `dark:bg-{color}-500/10` |

### Implementation

```html
<html class="dark">
  <body class="bg-gray-50 dark:bg-gray-950 text-gray-900 dark:text-white">
    ...
  </body>
</html>
```

Toggle via `document.documentElement.classList.toggle('dark')` or system preference via `prefers-color-scheme`.

---

## 7. Accessibility Requirements

### 7.1 Color Contrast

| Element | Ratio Required | Token Pair |
|---------|---------------|------------|
| Body text | ≥ 4.5:1 | `text-gray-700` on `bg-white` = 8.6:1 |
| Large text (≥18px bold) | ≥ 3:1 | `text-gray-500` on `bg-white` = 4.6:1 |
| UI components | ≥ 3:1 | `border-gray-300` on `bg-white` = 3.9:1 |
| Brand on white | ≥ 4.5:1 | `text-brand-500` on `bg-white` = 4.6:1 |
| White on brand | ≥ 4.5:1 | `text-white` on `bg-brand-500` = 4.6:1 |

### 7.2 Focus Management

- **All interactive elements** must have visible focus indicators
- **Focus ring:** `focus-visible:ring-2 focus-visible:ring-brand-500 focus-visible:ring-offset-2`
- **Skip link:** Hidden link at page top, visible on focus: "Skip to main content"
- **Focus trap:** Modals trap focus; ESC closes; Tab cycles within modal

### 7.3 Keyboard Navigation

| Key | Action |
|-----|--------|
| `Tab` | Move to next interactive element |
| `Shift+Tab` | Move to previous interactive element |
| `Enter` / `Space` | Activate button/link |
| `Escape` | Close modal/dropdown |
| `Arrow keys` | Navigate within select, tabs, tree |

### 7.4 Semantic HTML

- Use `<nav>`, `<main>`, `<aside>`, `<header>`, `<footer>` landmarks
- Use `<h1>` through `<h6>` in hierarchy (no skipped levels)
- Use `aria-label` for icon-only buttons
- Use `aria-live="polite"` for status updates (agent progress)
- Use `role="alert"` for error toasts
- Use `aria-expanded` for collapsible sections

---

## 8. Iconography

### 8.1 Icon Set

- **Library:** Heroicons (v2) — consistent with Tailwind
- **Style:** Outline (24x24) for nav/actions, Solid (24x24) for status indicators
- **Sizes:** 16x16 (inline), 20x20 (buttons/inputs), 24x24 (navigation), 32x32 (empty states)
- **Color:** Inherit from parent (`text-current`) or use semantic tokens

### 8.2 Agent Icons

| Agent | Icon | Solid Color |
|-------|------|-------------|
| PM | `DocumentTextIcon` | `text-agent-pm` |
| Architect | `BuildingOffice2Icon` | `text-agent-architect` |
| Developer | `CodeBracketIcon` | `text-agent-developer` |
| Review | `EyeIcon` | `text-agent-review` |
| QA | `BeakerIcon` | `text-agent-qa` |
| DevOps | `CloudIcon` | `text-agent-devops` |

---

## 9. Animation & Motion

### 9.1 Transitions

| Property | Duration | Easing | Usage |
|----------|----------|--------|-------|
| Color | 150ms | ease-in-out | Hover states, focus rings |
| Opacity | 200ms | ease-in-out | Fade in/out |
| Transform | 200ms | ease-out | Scale on hover, slide in |
| Layout | 300ms | ease-in-out | Accordion expand, sidebar |

### 9.2 Micro-interactions

- **Button press:** `scale-95` on `active` (100ms)
- **Card hover:** `shadow-md` transition (150ms)
- **Status dot pulse:** `animate-pulse` (2s infinite)
- **Skeleton loading:** `animate-pulse` with `bg-gray-200 dark:bg-gray-700 rounded`
- **Page transitions:** Fade in (200ms)

### 9.3 Reduced Motion

```css
@media (prefers-reduced-motion: reduce) {
  *, *::before, *::after {
    animation-duration: 0.01ms !important;
    animation-iteration-count: 1 !important;
    transition-duration: 0.01ms !important;
  }
}
```

---

## 10. Tailwind Configuration Summary

```js
// tailwind.config.js
module.exports = {
  content: ['./src/**/*.{js,ts,jsx,tsx}'],
  darkMode: 'class',
  theme: {
    extend: {
      colors: {
        brand: {
          100: '#E0E7FF',
          400: '#818CF8',
          500: '#6366F1',
          600: '#4F46E5',
          700: '#4338CA',
        },
        agent: {
          pm: '#8B5CF6',
          architect: '#0EA5E9',
          developer: '#10B981',
          review: '#F97316',
          qa: '#EC4899',
          devops: '#6366F1',
        },
      },
      fontSize: {
        'display': ['2.25rem', { lineHeight: '1.2', letterSpacing: '-0.025em', fontWeight: '700' }],
        'h1': ['1.875rem', { lineHeight: '1.33', letterSpacing: '-0.02em', fontWeight: '700' }],
        'h2': ['1.5rem', { lineHeight: '1.35', letterSpacing: '-0.015em', fontWeight: '600' }],
        'h3': ['1.25rem', { lineHeight: '1.4', letterSpacing: '-0.01em', fontWeight: '600' }],
        'body': ['0.875rem', { lineHeight: '1.6', fontWeight: '400' }],
        'body-sm': ['0.8125rem', { lineHeight: '1.55', fontWeight: '400' }],
        'caption': ['0.75rem', { lineHeight: '1.5', letterSpacing: '0.01em', fontWeight: '500' }],
      },
      fontFamily: {
        sans: ['Inter', 'system-ui', '-apple-system', 'sans-serif'],
        mono: ['JetBrains Mono', 'ui-monospace', 'monospace'],
      },
      borderRadius: {
        'xl': '0.75rem',
        '2xl': '1rem',
      },
    },
  },
  plugins: [],
}
```

---

## 11. File Structure

```
src/
├── styles/
│   ├── globals.css          # Tailwind directives, font imports, base styles
│   └── animations.css       # Custom keyframes (if needed)
├── components/
│   ├── ui/                  # Primitive components
│   │   ├── Button.tsx
│   │   ├── Input.tsx
│   │   ├── Badge.tsx
│   │   ├── Card.tsx
│   │   ├── Modal.tsx
│   │   ├── Table.tsx
│   │   └── Toast.tsx
│   ├── layout/              # Layout components
│   │   ├── Sidebar.tsx
│   │   ├── Header.tsx
│   │   └── PageContainer.tsx
│   └── agents/              # Agent-specific components
│       ├── AgentAvatar.tsx
│       ├── AgentStatus.tsx
│       └── AgentActivityCard.tsx
└── lib/
    └── design-tokens.ts     # Exported tokens for non-Tailwind usage
```
