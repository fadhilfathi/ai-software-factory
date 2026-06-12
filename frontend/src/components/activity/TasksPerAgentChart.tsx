"use client";

/**
 * Tasks per agent — horizontal bar chart.
 *
 * TASK-410 — Agent Activity Dashboard.
 * Renders a recharts BarChart of total task counts per agent for the
 * selected project + time range. Reads pre-aggregated data from a
 * parent's `Task[]` to keep the chart purely presentational.
 *
 * The chart is intentionally small and self-contained:
 *   - takes the agent+count map as a prop
 *   - picks a deterministic colour from the project palette
 *   - shows a tool-tip with the count and agent name on hover
 *
 * Why a prop-driven chart: recharts needs a stable data reference and
 * doesn't play well with arbitrary nested objects. The page aggregates
 * the tasks into a flat array and we just render that.
 */

import {
  Bar,
  BarChart,
  CartesianGrid,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from "recharts";

export type AgentTaskCount = {
  agentId: string;
  agentName: string;
  total: number;
  active: number;
  completed: number;
};

export function TasksPerAgentChart({
  data,
  loading = false,
  height = 280,
}: {
  data: AgentTaskCount[];
  loading?: boolean;
  height?: number;
}) {
  if (loading) {
    return (
      <div
        className="flex items-center justify-center rounded-lg border border-gray-800 bg-gray-900/30 text-sm text-gray-500"
        style={{ height }}
      >
        Loading chart…
      </div>
    );
  }

  if (data.length === 0) {
    return (
      <div
        className="flex items-center justify-center rounded-lg border border-dashed border-gray-800 bg-gray-900/30 text-sm text-gray-500"
        style={{ height }}
      >
        No tasks to chart yet.
      </div>
    );
  }

  // Top 12 only — keeps the chart readable when there are many agents.
  const visible = [...data].sort((a, b) => b.total - a.total).slice(0, 12);

  return (
    <div className="rounded-lg border border-gray-800 bg-gray-900/30 p-4">
      <ResponsiveContainer width="100%" height={height}>
        <BarChart
          data={visible}
          layout="vertical"
          margin={{ top: 8, right: 24, left: 8, bottom: 8 }}
        >
          <CartesianGrid
            strokeDasharray="3 3"
            stroke="rgb(55, 65, 81)"
            horizontal={false}
          />
          <XAxis
            type="number"
            allowDecimals={false}
            stroke="rgb(156, 163, 175)"
            fontSize={11}
          />
          <YAxis
            type="category"
            dataKey="agentName"
            width={140}
            stroke="rgb(156, 163, 175)"
            fontSize={11}
            tickFormatter={(v: string) =>
              v.length > 20 ? `${v.slice(0, 18)}…` : v
            }
          />
          <Tooltip
            cursor={{ fill: "rgba(16, 185, 129, 0.05)" }}
            contentStyle={{
              background: "rgb(17, 24, 39)",
              border: "1px solid rgb(55, 65, 81)",
              borderRadius: 6,
              color: "rgb(243, 244, 246)",
              fontSize: 12,
            }}
            formatter={(value, name) => {
              const num = typeof value === "number" ? value : Number(value ?? 0);
              const label =
                name === "total"
                  ? "Total tasks"
                  : name === "active"
                    ? "Active"
                    : "Completed";
              return [num, label] as [number, string];
            }}
          />
          <Bar
            dataKey="total"
            stackId="a"
            fill="rgb(16, 185, 129)"
            name="total"
            radius={[0, 0, 0, 0]}
          />
          <Bar
            dataKey="active"
            stackId="b"
            fill="rgb(245, 158, 11)"
            name="active"
            radius={[0, 0, 0, 0]}
          />
          <Bar
            dataKey="completed"
            stackId="c"
            fill="rgb(56, 189, 248)"
            name="completed"
            radius={[0, 4, 4, 0]}
          />
        </BarChart>
      </ResponsiveContainer>
      <div className="mt-2 flex flex-wrap items-center gap-3 text-xs text-gray-400">
        <LegendDot color="rgb(16, 185, 129)" label="Total tasks" />
        <LegendDot color="rgb(245, 158, 11)" label="Active (in window)" />
        <LegendDot color="rgb(56, 189, 248)" label="Completed (in window)" />
      </div>
    </div>
  );
}

function LegendDot({ color, label }: { color: string; label: string }) {
  return (
    <span className="inline-flex items-center gap-1.5">
      <span
        className="inline-block h-2 w-2 rounded-sm"
        style={{ background: color }}
      />
      {label}
    </span>
  );
}
