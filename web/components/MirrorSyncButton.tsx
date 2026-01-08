"use client";

import { useState } from "react";
import { syncMirrorRepository } from "@/lib/api";

interface MirrorSyncButtonProps {
  owner: string;
  repo: string;
  mirrorEnabled: boolean;
  syncStatus?: string;
  lastSyncedAt?: string;
  onSyncComplete?: () => void;
}

export default function MirrorSyncButton({
  owner,
  repo,
  mirrorEnabled,
  syncStatus,
  lastSyncedAt,
  onSyncComplete,
}: MirrorSyncButtonProps) {
  const [syncing, setSyncing] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState(false);

  if (!mirrorEnabled) {
    return null;
  }

  const handleSync = async () => {
    setSyncing(true);
    setError(null);
    setSuccess(false);

    try {
      await syncMirrorRepository(owner, repo);
      setSuccess(true);

      // Call callback if provided
      if (onSyncComplete) {
        // Wait a bit for the sync to start
        setTimeout(() => {
          onSyncComplete();
        }, 1000);
      }

      // Clear success message after 3 seconds
      setTimeout(() => {
        setSuccess(false);
      }, 3000);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to sync mirror");
    } finally {
      setSyncing(false);
    }
  };

  const isSyncing = syncStatus === "syncing" || syncing;
  const hasFailed = syncStatus === "failed";

  const formatLastSync = (timestamp?: string) => {
    if (!timestamp) return "Never";

    const date = new Date(timestamp);
    const now = new Date();
    const diff = now.getTime() - date.getTime();

    const minutes = Math.floor(diff / 60000);
    const hours = Math.floor(diff / 3600000);
    const days = Math.floor(diff / 86400000);

    if (minutes < 1) return "Just now";
    if (minutes < 60) return `${minutes}m ago`;
    if (hours < 24) return `${hours}h ago`;
    if (days < 30) return `${days}d ago`;

    return date.toLocaleDateString();
  };

  return (
    <div className="flex items-center gap-3">
      <div className="flex items-center gap-2 text-sm text-muted">
        <svg
          xmlns="http://www.w3.org/2000/svg"
          width="16"
          height="16"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          strokeWidth="2"
          strokeLinecap="round"
          strokeLinejoin="round"
          className={isSyncing ? "animate-spin" : ""}
        >
          <path d="M21.5 2v6h-6M2.5 22v-6h6M2 11.5a10 10 0 0 1 18.8-4.3M22 12.5a10 10 0 0 1-18.8 4.2" />
        </svg>
        <span>
          Mirror •{" "}
          {isSyncing ? (
            <span className="text-blue-500">Syncing...</span>
          ) : hasFailed ? (
            <span className="text-red-500">Sync failed</span>
          ) : (
            <span>Last synced: {formatLastSync(lastSyncedAt)}</span>
          )}
        </span>
      </div>

      <button
        onClick={handleSync}
        disabled={isSyncing}
        className={`flex items-center gap-2 px-3 py-1.5 text-sm rounded-md transition-colors ${
          success
            ? "bg-green-100 dark:bg-green-900/20 text-green-700 dark:text-green-400 border border-green-300 dark:border-green-700"
            : isSyncing
              ? "bg-blue-100 dark:bg-blue-900/20 text-blue-700 dark:text-blue-400 border border-blue-300 dark:border-blue-700 cursor-not-allowed"
              : "bg-panel border border-base hover:bg-base text-base"
        }`}
        title="Sync mirror repository"
      >
        <svg
          xmlns="http://www.w3.org/2000/svg"
          width="14"
          height="14"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          strokeWidth="2"
          strokeLinecap="round"
          strokeLinejoin="round"
          className={isSyncing ? "animate-spin" : ""}
        >
          <path d="M21.5 2v6h-6M2.5 22v-6h6M2 11.5a10 10 0 0 1 18.8-4.3M22 12.5a10 10 0 0 1-18.8 4.2" />
        </svg>
        {success ? "Synced!" : isSyncing ? "Syncing..." : "Sync now"}
      </button>

      {error && (
        <div className="text-xs text-red-500 dark:text-red-400">{error}</div>
      )}
    </div>
  );
}
