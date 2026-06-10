"use client";

import { useState, useCallback } from "react";
import { PageHeader } from "@/components/layout/PageHeader";
import { cn } from "@/lib/utils";
import { Input } from "@/components/form/Input";
import { Select } from "@/components/form/Select";
import { Toggle } from "@/components/ui/Toggle";
import { ConfirmDialog } from "@/components/shared/ConfirmDialog";
import { SpinnerButton } from "@/components/ui/Spinner";
import { useUI } from "@/providers/UIProvider";

type SettingsTab = "General" | "Integrations" | "Notifications" | "Billing" | "Security";

const TABS: SettingsTab[] = ["General", "Integrations", "Notifications", "Billing", "Security"];

type NotificationSetting = {
  key: string;
  label: string;
  enabled: boolean;
};

const SESSION_TIMEOUTS = [
  { value: "15", label: "15 minutes" },
  { value: "30", label: "30 minutes" },
  { value: "60", label: "1 hour" },
  { value: "240", label: "4 hours" },
  { value: "480", label: "8 hours" },
];

export default function SettingsPage() {
  const [activeTab, setActiveTab] = useState<SettingsTab>("General");
  const [saving, setSaving] = useState(false);
  const [confirmAction, setConfirmAction] = useState<string | null>(null);
  const { addToast } = useUI();

  // General settings
  const [platformName, setPlatformName] = useState("AI Software Factory");
  const [defaultStack, setDefaultStack] = useState("Node.js");
  const [deployTarget, setDeployTarget] = useState("AWS");

  // Notifications
  const [notifications, setNotifications] = useState<NotificationSetting[]>([
    { key: "project_done", label: "Project Done", enabled: true },
    { key: "gate_failed", label: "Gate Failed", enabled: true },
    { key: "agent_error", label: "Agent Error", enabled: true },
    { key: "budget_80", label: "Budget 80%", enabled: true },
    { key: "daily_summary", label: "Daily Summary", enabled: false },
  ]);

  // Integrations
  const [integrations, setIntegrations] = useState([
    { name: "GitHub", connected: true },
    { name: "AWS", connected: true },
    { name: "Slack", connected: false },
    { name: "Webhook", connected: false },
  ]);

  // Security
  const [twoFactor, setTwoFactor] = useState(false);
  const [sessionTimeout, setSessionTimeout] = useState("30");

  const handleSave = async () => {
    setSaving(true);
    await new Promise((r) => setTimeout(r, 800));
    setSaving(false);
    addToast({ type: "success", message: "Settings saved successfully" });
  };

  const toggleIntegration = (name: string) => {
    setIntegrations((prev) =>
      prev.map((i) => (i.name === name ? { ...i, connected: !i.connected } : i)),
    );
  };

  const toggleNotification = (key: string) => {
    setNotifications((prev) =>
      prev.map((n) => (n.key === key ? { ...n, enabled: !n.enabled } : n)),
    );
  };

  const handleDangerAction = useCallback((action: string) => {
    setConfirmAction(action);
  }, []);

  const handleConfirm = useCallback(() => {
    setConfirmAction(null);
    addToast({ type: "info", message: `${confirmAction} — action simulated (no-op)` });
  }, [confirmAction, addToast]);

  return (
    <div>
      <PageHeader
        title="Settings"
        actions={
          <SpinnerButton onClick={handleSave} loading={saving} loadingText="Saving...">
            Save Changes
          </SpinnerButton>
        }
      />

      <div className="grid gap-6 lg:grid-cols-4">
        {/* Settings Nav */}
        <nav className="space-y-1 lg:col-span-1" aria-label="Settings sections">
          {TABS.map((tab) => (
            <button
              key={tab}
              onClick={() => setActiveTab(tab)}
              className={cn(
                "w-full rounded-lg px-4 py-2.5 text-left text-sm font-medium transition-colors",
                activeTab === tab
                  ? "bg-emerald-500/10 text-emerald-400"
                  : "text-gray-400 hover:bg-gray-800 hover:text-gray-200",
              )}
              type="button"
            >
              {tab}
            </button>
          ))}
        </nav>

        {/* Settings Panel */}
        <div className="lg:col-span-3">
          {/* General */}
          {activeTab === "General" && (
            <div className="space-y-6">
              <div className="rounded-lg border border-gray-800 bg-gray-950 p-4">
                <h3 className="mb-4 text-sm font-semibold text-gray-300 uppercase tracking-wider">
                  Platform
                </h3>
                <div className="space-y-4">
                  <Input
                    label="Platform Name"
                    value={platformName}
                    onChange={(e) => setPlatformName(e.target.value)}
                  />
                  <Select
                    label="Default Stack"
                    value={defaultStack}
                    onChange={(e) => setDefaultStack(e.target.value)}
                    options={[
                      { value: "Node.js", label: "Node.js" },
                      { value: "Python", label: "Python" },
                      { value: "Go", label: "Go" },
                      { value: "Rust", label: "Rust" },
                    ]}
                  />
                  <Select
                    label="Default Deploy Target"
                    value={deployTarget}
                    onChange={(e) => setDeployTarget(e.target.value)}
                    options={[
                      { value: "AWS", label: "AWS" },
                      { value: "Vercel", label: "Vercel" },
                      { value: "Railway", label: "Railway" },
                      { value: "Self-hosted", label: "Self-hosted" },
                    ]}
                  />
                </div>
              </div>

              <div className="rounded-lg border border-gray-800 bg-gray-950 p-4">
                <h3 className="mb-4 text-sm font-semibold text-gray-300 uppercase tracking-wider">
                  Theme
                </h3>
                <div className="flex items-center justify-between">
                  <div>
                    <p className="text-sm text-gray-200">Dark Mode</p>
                    <p className="text-xs text-gray-500">Currently using dark theme (only supported)</p>
                  </div>
                  <div className="h-6 w-11 rounded-full bg-emerald-500 relative pointer-events-none opacity-60">
                    <div className="absolute right-1 top-1 h-4 w-4 rounded-full bg-white" />
                  </div>
                </div>
              </div>
            </div>
          )}

          {/* Integrations */}
          {activeTab === "Integrations" && (
            <div className="space-y-3">
              {integrations.map((integration) => (
                <div
                  key={integration.name}
                  className="flex items-center justify-between rounded-lg border border-gray-800 bg-gray-950 px-4 py-3"
                >
                  <div>
                    <p className="text-sm font-medium text-gray-200">{integration.name}</p>
                    <p className="text-xs text-gray-500">
                      {integration.connected ? "Connected" : "Disconnected"}
                    </p>
                  </div>
                  <button
                    onClick={() => toggleIntegration(integration.name)}
                    className={cn(
                      "rounded-lg px-4 py-1.5 text-xs font-medium transition-colors",
                      integration.connected
                        ? "bg-red-500/10 text-red-400 hover:bg-red-500/20"
                        : "bg-emerald-500 text-white hover:bg-emerald-600",
                    )}
                    type="button"
                  >
                    {integration.connected ? "Disconnect" : "Connect"}
                  </button>
                </div>
              ))}
            </div>
          )}

          {/* Notifications */}
          {activeTab === "Notifications" && (
            <div className="space-y-3">
              {notifications.map((item) => (
                <div
                  key={item.key}
                  className="rounded-lg border border-gray-800 bg-gray-950 px-4 py-3 hover:border-gray-700 transition-colors"
                >
                  <Toggle
                    label={item.label}
                    checked={item.enabled}
                    onChange={() => toggleNotification(item.key)}
                  />
                </div>
              ))}
            </div>
          )}

          {/* Billing */}
          {activeTab === "Billing" && (
            <div className="space-y-6">
              <div className="rounded-lg border border-gray-800 bg-gray-950 p-6 text-center">
                <div className="mb-4 flex justify-center">
                  <div className="rounded-full bg-emerald-500/10 p-3">
                    <span className="text-2xl">💳</span>
                  </div>
                </div>
                <h3 className="text-lg font-semibold text-gray-200">Free Plan</h3>
                <p className="mt-1 text-sm text-gray-400">
                  You&apos;re currently on the free tier.
                </p>
                <div className="mt-4 space-y-2 text-left text-sm text-gray-400">
                  <div className="flex items-center gap-2">
                    <span className="text-emerald-400">✓</span> Up to 5 active projects
                  </div>
                  <div className="flex items-center gap-2">
                    <span className="text-emerald-400">✓</span> 6 agent types
                  </div>
                  <div className="flex items-center gap-2">
                    <span className="text-emerald-400">✓</span> Community support
                  </div>
                  <div className="flex items-center gap-2 text-gray-600">
                    <span>✗</span> Priority support
                  </div>
                  <div className="flex items-center gap-2 text-gray-600">
                    <span>✗</span> Custom integrations
                  </div>
                </div>
                <button
                  className="mt-6 rounded-lg bg-emerald-500 px-6 py-2 text-sm font-medium text-white hover:bg-emerald-600 transition-colors"
                  type="button"
                >
                  Upgrade to Pro
                </button>
              </div>
            </div>
          )}

          {/* Security */}
          {activeTab === "Security" && (
            <div className="space-y-6">
              <div className="rounded-lg border border-gray-800 bg-gray-950 p-4">
                <h3 className="mb-4 text-sm font-semibold text-gray-300 uppercase tracking-wider">
                  Authentication
                </h3>
                <Toggle
                  label="Two-Factor Authentication"
                  description="Add an extra layer of security to your account"
                  checked={twoFactor}
                  onChange={setTwoFactor}
                />
              </div>

              <div className="rounded-lg border border-gray-800 bg-gray-950 p-4">
                <h3 className="mb-4 text-sm font-semibold text-gray-300 uppercase tracking-wider">
                  Session
                </h3>
                <Select
                  label="Session Timeout (minutes)"
                  value={sessionTimeout}
                  onChange={(e) => setSessionTimeout(e.target.value)}
                  options={SESSION_TIMEOUTS}
                />
              </div>

              <div className="rounded-lg border border-red-800/50 bg-red-950/30 p-4">
                <h3 className="mb-1 text-sm font-semibold text-red-400">Danger Zone</h3>
                <p className="mb-3 text-xs text-gray-500">
                  Irreversible actions — proceed with caution.
                </p>
                <div className="flex flex-wrap gap-3">
                  <button
                    onClick={() => handleDangerAction("Rotate API Keys")}
                    className="rounded-lg border border-red-800/50 px-4 py-1.5 text-xs font-medium text-red-400 hover:bg-red-950/50 transition-colors"
                    type="button"
                  >
                    Rotate API Keys
                  </button>
                  <button
                    onClick={() => handleDangerAction("Delete Account")}
                    className="rounded-lg border border-red-800/50 px-4 py-1.5 text-xs font-medium text-red-400 hover:bg-red-950/50 transition-colors"
                    type="button"
                  >
                    Delete Account
                  </button>
                </div>
              </div>
            </div>
          )}
        </div>
      </div>

      {/* Confirm Dialog */}
      <ConfirmDialog
        open={confirmAction !== null}
        onConfirm={handleConfirm}
        onCancel={() => setConfirmAction(null)}
        title={confirmAction ?? ""}
        message={`Are you sure you want to ${confirmAction?.toLowerCase()}? This action cannot be undone.`}
        confirmLabel={confirmAction ?? "Confirm"}
        variant="danger"
      />
    </div>
  );
}
