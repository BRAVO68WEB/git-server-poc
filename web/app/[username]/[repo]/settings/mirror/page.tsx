"use client";

import { useState, useEffect } from "react";
import { useParams, useRouter } from "next/navigation";
import Link from "next/link";
import {
  getMirrorSettings,
  updateMirrorSettings,
  getRepository,
} from "@/lib/api";
import {
  describeCronExpression,
  getNextRunTime,
  formatNextRunTime,
  validateCronExpression,
} from "@/lib/cron-utils";
import type {
  MirrorSettingsResponse,
  UpdateMirrorSettingsRequest,
  RepoResponse,
} from "@/lib/types";

export default function MirrorSettingsPage() {
  const params = useParams();
  const router = useRouter();
  const username = params.username as string;
  const repoName = params.repo as string;

  const [repository, setRepository] = useState<RepoResponse | null>(null);
  const [settings, setSettings] = useState<MirrorSettingsResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState(false);

  const [formData, setFormData] = useState({
    mirror_enabled: false,
    mirror_direction: "upstream" as "upstream" | "downstream" | "both",
    upstream_url: "",
    upstream_username: "",
    upstream_password: "",
    downstream_url: "",
    downstream_username: "",
    downstream_password: "",
    sync_schedule: "0 */1 * * *", // Default: every hour
  });

  useEffect(() => {
    loadSettings();
  }, [username, repoName]);

  const loadSettings = async () => {
    try {
      setLoading(true);
      const [repoData, settingsData] = await Promise.all([
        getRepository(username, repoName),
        getMirrorSettings(username, repoName),
      ]);

      setRepository(repoData);
      setSettings(settingsData);

      // Populate form
      setFormData({
        mirror_enabled: settingsData.mirror_enabled,
        mirror_direction:
          (settingsData.mirror_direction as
            | "upstream"
            | "downstream"
            | "both") || "upstream",
        upstream_url: settingsData.upstream_url || "",
        upstream_username: settingsData.upstream_username || "",
        upstream_password: "",
        downstream_url: settingsData.downstream_url || "",
        downstream_username: settingsData.downstream_username || "",
        downstream_password: "",
        sync_schedule: settingsData.sync_schedule || "0 */1 * * *",
      });
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load settings");
    } finally {
      setLoading(false);
    }
  };

  const handleChange = (
    e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>,
  ) => {
    const { name, value, type } = e.target;
    if (type === "checkbox") {
      const checked = (e.target as HTMLInputElement).checked;
      setFormData((prev) => ({ ...prev, [name]: checked }));
    } else if (type === "number") {
      setFormData((prev) => ({ ...prev, [name]: parseInt(value) }));
    } else {
      setFormData((prev) => ({ ...prev, [name]: value }));
    }
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);
    setSuccess(false);
    setSaving(true);

    try {
      // Build update request (only include changed fields)
      const updateData: UpdateMirrorSettingsRequest = {
        mirror_enabled: formData.mirror_enabled,
        mirror_direction: formData.mirror_direction,
        sync_schedule: formData.sync_schedule,
      };

      // Include upstream if direction is upstream or both
      if (
        formData.mirror_direction === "upstream" ||
        formData.mirror_direction === "both"
      ) {
        updateData.upstream_url = formData.upstream_url;
        updateData.upstream_username = formData.upstream_username;
        if (formData.upstream_password) {
          updateData.upstream_password = formData.upstream_password;
        }
      }

      // Include downstream if direction is downstream or both
      if (
        formData.mirror_direction === "downstream" ||
        formData.mirror_direction === "both"
      ) {
        updateData.downstream_url = formData.downstream_url;
        updateData.downstream_username = formData.downstream_username;
        if (formData.downstream_password) {
          updateData.downstream_password = formData.downstream_password;
        }
      }

      await updateMirrorSettings(username, repoName, updateData);
      setSuccess(true);

      // Reload settings
      await loadSettings();

      // Clear password fields
      setFormData((prev) => ({
        ...prev,
        upstream_password: "",
        downstream_password: "",
      }));

      setTimeout(() => setSuccess(false), 3000);
    } catch (err) {
      setError(
        err instanceof Error ? err.message : "Failed to update settings",
      );
    } finally {
      setSaving(false);
    }
  };

  if (loading) {
    return <div className="text-center text-muted py-8">Loading...</div>;
  }

  return (
    <div className="space-y-8">
      <form onSubmit={handleSubmit} className="space-y-8">
        {error && (
          <div className="bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 text-red-600 dark:text-red-400 px-4 py-3 rounded-md text-sm">
            {error}
          </div>
        )}

        {success && (
          <div className="bg-green-50 dark:bg-green-900/20 border border-green-200 dark:border-green-800 text-green-600 dark:text-green-400 px-4 py-3 rounded-md text-sm">
            Mirror settings updated successfully!
          </div>
        )}

        {/* Enable Mirror */}
        <div className="border border-base rounded-md p-4 bg-panel">
          <div className="flex items-start gap-3">
            <input
              id="mirror_enabled"
              name="mirror_enabled"
              type="checkbox"
              checked={formData.mirror_enabled}
              onChange={handleChange}
              className="mt-1 h-4 w-4 text-accent border-base rounded focus:ring-accent"
            />
            <div className="flex-1">
              <label
                htmlFor="mirror_enabled"
                className="block text-sm font-medium text-base cursor-pointer"
              >
                Enable Mirror Sync
              </label>
              <p className="text-xs text-muted mt-1">
                Automatically sync this repository with external sources or
                destinations.
              </p>
            </div>
          </div>
        </div>

        {formData.mirror_enabled && (
          <>
            {/* Mirror Direction */}
            <div>
              <label
                htmlFor="mirror_direction"
                className="block text-sm font-medium text-base mb-2"
              >
                Mirror Direction
              </label>
              <select
                id="mirror_direction"
                name="mirror_direction"
                value={formData.mirror_direction}
                onChange={handleChange}
                className="w-full px-3 py-2 border border-base rounded-md shadow-sm bg-panel text-base focus:outline-none focus:ring-2 focus:ring-accent focus:border-transparent"
              >
                <option value="upstream">Upstream (Pull from source)</option>
                <option value="downstream">
                  Downstream (Push to destination)
                </option>
                <option value="both">Both (Pull and Push)</option>
              </select>
              <p className="mt-1 text-xs text-muted">
                Choose whether to pull from an external source, push to an
                external destination, or both.
              </p>
            </div>

            {/* Sync Schedule */}
            <div>
              <label
                htmlFor="sync_schedule"
                className="block text-sm font-medium text-base mb-2"
              >
                Sync Schedule (Cron Expression)
              </label>
              <input
                id="sync_schedule"
                name="sync_schedule"
                type="text"
                value={formData.sync_schedule}
                onChange={handleChange}
                className={`w-full px-3 py-2 border rounded-md shadow-sm bg-panel text-base focus:outline-none focus:ring-2 focus:ring-accent focus:border-transparent ${
                  !validateCronExpression(formData.sync_schedule)
                    ? "border-red-500"
                    : "border-base"
                }`}
                placeholder="0 */1 * * *"
              />
              <div className="mt-2 space-y-1">
                {validateCronExpression(formData.sync_schedule) ? (
                  <>
                    <p className="text-sm text-green-600 dark:text-green-400">
                      ✓ Valid: {describeCronExpression(formData.sync_schedule)}
                    </p>
                    {(() => {
                      const nextRun = getNextRunTime(formData.sync_schedule);
                      return nextRun ? (
                        <p className="text-sm text-muted">
                          Next sync: {formatNextRunTime(nextRun)} (
                          {nextRun.toLocaleString()})
                        </p>
                      ) : null;
                    })()}
                  </>
                ) : (
                  <p className="text-sm text-red-600 dark:text-red-400">
                    ✗ Invalid cron expression
                  </p>
                )}
              </div>
              <div className="mt-2 text-xs text-muted space-y-1">
                <p className="font-medium">Common examples:</p>
                <ul className="list-disc list-inside space-y-1 ml-2">
                  <li>
                    <code className="bg-base px-1 py-0.5 rounded">
                      0 */1 * * *
                    </code>{" "}
                    - Every hour
                  </li>
                  <li>
                    <code className="bg-base px-1 py-0.5 rounded">
                      0 */2 * * *
                    </code>{" "}
                    - Every 2 hours
                  </li>
                  <li>
                    <code className="bg-base px-1 py-0.5 rounded">
                      0 0 * * *
                    </code>{" "}
                    - Daily at midnight
                  </li>
                  <li>
                    <code className="bg-base px-1 py-0.5 rounded">
                      0 0 * * 0
                    </code>{" "}
                    - Weekly on Sunday
                  </li>
                </ul>
              </div>
            </div>

            {/* Upstream Settings */}
            {(formData.mirror_direction === "upstream" ||
              formData.mirror_direction === "both") && (
              <div className="border border-base rounded-md p-6 bg-panel space-y-4">
                <h3 className="text-lg font-medium text-base">
                  Upstream Configuration
                </h3>
                <p className="text-sm text-muted">
                  Pull updates from an external Git repository.
                </p>

                <div>
                  <label
                    htmlFor="upstream_url"
                    className="block text-sm font-medium text-base"
                  >
                    Source URL <span className="text-red-500">*</span>
                  </label>
                  <input
                    id="upstream_url"
                    name="upstream_url"
                    type="url"
                    required
                    value={formData.upstream_url}
                    onChange={handleChange}
                    className="mt-1 block w-full px-3 py-2 border border-base rounded-md shadow-sm bg-panel text-base focus:outline-none focus:ring-2 focus:ring-accent focus:border-transparent"
                    placeholder="https://github.com/user/repo.git"
                  />
                </div>

                <div>
                  <label
                    htmlFor="upstream_username"
                    className="block text-sm font-medium text-base"
                  >
                    Username (optional)
                  </label>
                  <input
                    id="upstream_username"
                    name="upstream_username"
                    type="text"
                    value={formData.upstream_username}
                    onChange={handleChange}
                    className="mt-1 block w-full px-3 py-2 border border-base rounded-md shadow-sm bg-panel text-base focus:outline-none focus:ring-2 focus:ring-accent focus:border-transparent"
                    placeholder="username or token"
                  />
                </div>

                <div>
                  <label
                    htmlFor="upstream_password"
                    className="block text-sm font-medium text-base"
                  >
                    Password / Token (optional)
                  </label>
                  <input
                    id="upstream_password"
                    name="upstream_password"
                    type="password"
                    value={formData.upstream_password}
                    onChange={handleChange}
                    className="mt-1 block w-full px-3 py-2 border border-base rounded-md shadow-sm bg-panel text-base focus:outline-none focus:ring-2 focus:ring-accent focus:border-transparent"
                    placeholder="Leave empty to keep existing"
                  />
                  <p className="mt-1 text-xs text-muted">
                    Use personal access token for GitHub/GitLab. Leave empty if
                    already set.
                  </p>
                </div>
              </div>
            )}

            {/* Downstream Settings */}
            {(formData.mirror_direction === "downstream" ||
              formData.mirror_direction === "both") && (
              <div className="border border-base rounded-md p-6 bg-panel space-y-4">
                <h3 className="text-lg font-medium text-base">
                  Downstream Configuration
                </h3>
                <p className="text-sm text-muted">
                  Push updates to an external Git repository.
                </p>

                <div>
                  <label
                    htmlFor="downstream_url"
                    className="block text-sm font-medium text-base"
                  >
                    Destination URL <span className="text-red-500">*</span>
                  </label>
                  <input
                    id="downstream_url"
                    name="downstream_url"
                    type="url"
                    required
                    value={formData.downstream_url}
                    onChange={handleChange}
                    className="mt-1 block w-full px-3 py-2 border border-base rounded-md shadow-sm bg-panel text-base focus:outline-none focus:ring-2 focus:ring-accent focus:border-transparent"
                    placeholder="https://github.com/user/backup-repo.git"
                  />
                </div>

                <div>
                  <label
                    htmlFor="downstream_username"
                    className="block text-sm font-medium text-base"
                  >
                    Username (optional)
                  </label>
                  <input
                    id="downstream_username"
                    name="downstream_username"
                    type="text"
                    value={formData.downstream_username}
                    onChange={handleChange}
                    className="mt-1 block w-full px-3 py-2 border border-base rounded-md shadow-sm bg-panel text-base focus:outline-none focus:ring-2 focus:ring-accent focus:border-transparent"
                    placeholder="username or token"
                  />
                </div>

                <div>
                  <label
                    htmlFor="downstream_password"
                    className="block text-sm font-medium text-base"
                  >
                    Password / Token (optional)
                  </label>
                  <input
                    id="downstream_password"
                    name="downstream_password"
                    type="password"
                    value={formData.downstream_password}
                    onChange={handleChange}
                    className="mt-1 block w-full px-3 py-2 border border-base rounded-md shadow-sm bg-panel text-base focus:outline-none focus:ring-2 focus:ring-accent focus:border-transparent"
                    placeholder="Leave empty to keep existing"
                  />
                  <p className="mt-1 text-xs text-muted">
                    Use personal access token for GitHub/GitLab. Leave empty if
                    already set.
                  </p>
                </div>
              </div>
            )}

            {/* Sync Status */}
            {settings && (
              <div className="border border-base rounded-md p-4 bg-panel">
                <h3 className="text-sm font-medium text-base mb-2">
                  Sync Status
                </h3>
                <div className="space-y-2 text-sm">
                  <div className="flex justify-between">
                    <span className="text-muted">Status:</span>
                    <span
                      className={
                        settings.sync_status === "success"
                          ? "text-green-500"
                          : settings.sync_status === "failed"
                            ? "text-red-500"
                            : settings.sync_status === "syncing"
                              ? "text-blue-500"
                              : "text-muted"
                      }
                    >
                      {settings.sync_status || "idle"}
                    </span>
                  </div>
                  {settings.last_synced_at && (
                    <div className="flex justify-between">
                      <span className="text-muted">Last Synced:</span>
                      <span className="text-base">
                        {new Date(settings.last_synced_at).toLocaleString()}
                      </span>
                    </div>
                  )}
                  {settings.next_sync_at && (
                    <div className="flex justify-between">
                      <span className="text-muted">Next Sync:</span>
                      <span className="text-base">
                        {new Date(settings.next_sync_at).toLocaleString()}
                      </span>
                    </div>
                  )}
                  {settings.sync_schedule && (
                    <div className="flex justify-between">
                      <span className="text-muted">Schedule:</span>
                      <code className="text-xs bg-base px-1 py-0.5 rounded">
                        {settings.sync_schedule}
                      </code>
                    </div>
                  )}
                  {settings.sync_error && (
                    <div className="mt-2 text-xs text-red-500">
                      Error: {settings.sync_error}
                    </div>
                  )}
                </div>
              </div>
            )}
          </>
        )}

        {/* Submit Button */}
        <div className="flex items-center justify-end pt-4 border-t border-base">
          <button
            type="submit"
            disabled={saving}
            className="px-4 py-2 border border-transparent rounded-md shadow-sm text-sm font-medium text-white bg-accent hover:opacity-90 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-accent disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {saving ? "Saving..." : "Save Changes"}
          </button>
        </div>
      </form>
    </div>
  );
}
