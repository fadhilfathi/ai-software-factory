# Cross-Agent Dispatch Log

Append a row for every handoff: Builder → Guardian review, Guardian → Ops push,
or any blocker.

Format:

| When (UTC) | From | To | Task | Action | Evidence |
|------------|------|-----|------|--------|----------|
| ...        | ...  | ... | ...  | ...    | ...      |

Ops maintains this file. Append-only.
