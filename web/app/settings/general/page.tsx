"use client";

import { useState, useEffect } from "react";
import { useRouter } from "next/navigation";
import {
  updateUsername,
  getUserInfo,
  setUserInfo,
  isAuthenticated,
} from "@/lib/api";

export default function GeneralSettingsPage() {
  const router = useRouter();
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);
  const [username, setUsername] = useState("");

  useEffect(() => {
    // Check if user is authenticated
    if (!isAuthenticated()) {
      router.push("/auth/login");
      return;
    }

    const user = getUserInfo();
    if (user) {
      setUsername(user.username);
    }
  }, [router]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);
    setSuccess(null);
    setLoading(true);

    try {
      const response = await updateUsername(username);
      setUserInfo(response.user);
      setSuccess("Username updated successfully");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to update username");
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="max-w-3xl space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-xl font-semibold text-base">General Settings</h2>
          <p className="text-sm text-muted mt-1">
            Manage your account details.
          </p>
        </div>
      </div>

      {error && (
        <div className="bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 text-red-600 dark:text-red-400 px-4 py-3 rounded-md text-sm">
          {error}
        </div>
      )}

      {success && (
        <div className="bg-green-50 dark:bg-green-900/20 border border-green-200 dark:border-green-800 text-green-600 dark:text-green-400 px-4 py-3 rounded-md text-sm">
          {success}
        </div>
      )}

      <form
        onSubmit={handleSubmit}
        className="border border-base rounded-md bg-panel"
      >
        <div className="px-4 py-3 border-b border-base">
          <h3 className="font-medium text-base">Profile</h3>
        </div>

        <div className="p-4 space-y-4">
          <div>
            <label
              htmlFor="username"
              className="block text-sm font-medium text-base"
            >
              Username
            </label>
            <input
              id="username"
              name="username"
              type="text"
              required
              value={username}
              onChange={(e) => setUsername(e.target.value)}
              className="mt-1 block w-full px-3 py-2 border border-base rounded-md shadow-sm bg-base text-base focus:outline-none focus:ring-2 focus:ring-accent focus:border-transparent"
              placeholder="username"
              maxLength={50}
            />
            <p className="mt-1 text-xs text-muted">
              Your unique username on the platform.
            </p>
          </div>
        </div>

        <div className="px-4 py-3 bg-base/50 border-t border-base flex justify-end">
          <button
            type="submit"
            disabled={loading}
            className="px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white text-sm font-medium rounded-md shadow-sm transition-colors focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {loading ? "Saving..." : "Save changes"}
          </button>
        </div>
      </form>
    </div>
  );
}
