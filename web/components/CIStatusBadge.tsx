"use client";

import { useState, useEffect } from "react";
import Link from "next/link";
import { getLatestCIJob, getCIStatusColor, getCIStatusBgColor } from "@/lib/api";
import { CIJob, CIJobStatus } from "@/lib/types";

interface CIStatusBadgeProps {
  owner: string;
  repo: string;
  className?: string;
}

// Status icon component
function StatusIcon({ status, size = "sm" }: { status: string; size?: "sm" | "md" }) {
  const sizeClass = size === "sm" ? "w-3 h-3" : "w-4 h-4";

  switch (status) {
    case "success":
      return (
        <svg className={sizeClass} fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
        </svg>
      );
    case "failed":
    case "error":
      return (
        <svg className={sizeClass} fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
        </svg>
      );
    case "running":
      return (
        <svg className={`${sizeClass} animate-spin`} fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
          <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
        </svg>
      );
    case "pending":
    case "queued":
      return (
        <svg className={sizeClass} fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <circle cx="12" cy="12" r="10" strokeWidth={2} />
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 6v6l4 2" />
        </svg>
      );
    case "cancelled":
      return (
        <svg className={sizeClass} fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <circle cx="12" cy="12" r="10" strokeWidth={2} />
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 9h6v6H9z" />
        </svg>
      );
    default:
      return null;
  }
}

export default function CIStatusBadge({ owner, repo, className = "" }: CIStatusBadgeProps) {
  const [job, setJob] = useState<CIJob | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(false);

  useEffect(() => {
    async function fetchLatestJob() {
      try {
        setLoading(true);
        setError(false);
        const latestJob = await getLatestCIJob(owner, repo);
        setJob(latestJob);
      } catch (err) {
        // No CI jobs or CI not enabled - this is not an error
        setError(true);
      } finally {
        setLoading(false);
      }
    }

    fetchLatestJob();
  }, [owner, repo]);

  // Refresh running jobs periodically
  useEffect(() => {
    if (!job) return;

    const isRunning = job.status === "running" || job.status === "pending" || job.status === "queued";

    if (!isRunning) return;

    const interval = setInterval(async () => {
      try {
        const latestJob = await getLatestCIJob(owner, repo);
        setJob(latestJob);
      } catch (err) {
        console.error("Failed to refresh CI status:", err);
      }
    }, 5000);

    return () => clearInterval(interval);
  }, [job, owner, repo]);

  // Don't show anything while loading or if there's an error (no CI)
  if (loading || error || !job) {
    return null;
  }

  return (
    <Link
      href={`/${owner}/${repo}/ci/${job.id}`}
      className={`inline-flex items-center gap-1.5 px-2 py-1 rounded-full text-xs font-medium border transition-colors hover:opacity-80 ${getCIStatusBgColor(
        job.status
      )} ${getCIStatusColor(job.status)} ${className}`}
      title={`Latest CI: ${job.status}`}
    >
      <StatusIcon status={job.status} />
      <span className="capitalize">{job.status}</span>
    </Link>
  );
}

// Compact version for inline use
export function CIStatusDot({ owner, repo, className = "" }: CIStatusBadgeProps) {
  const [job, setJob] = useState<CIJob | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    async function fetchLatestJob() {
      try {
        setLoading(true);
        const latestJob = await getLatestCIJob(owner, repo);
        setJob(latestJob);
      } catch (err) {
        // No CI jobs - ignore
      } finally {
        setLoading(false);
      }
    }

    fetchLatestJob();
  }, [owner, repo]);

  if (loading || !job) {
    return null;
  }

  const dotColor = getDotColor(job.status);

  return (
    <Link
      href={`/${owner}/${repo}/ci`}
      className={`inline-flex items-center ${className}`}
      title={`CI: ${job.status}`}
    >
      <span
        className={`w-2 h-2 rounded-full ${dotColor} ${
          job.status === "running" ? "animate-pulse" : ""
        }`}
      />
    </Link>
  );
}

function getDotColor(status: CIJobStatus): string {
  switch (status) {
    case "success":
      return "bg-green-500";
    case "failed":
    case "error":
      return "bg-red-500";
    case "running":
      return "bg-blue-500";
    case "pending":
    case "queued":
      return "bg-yellow-500";
    case "cancelled":
      return "bg-gray-500";
    case "timed_out":
      return "bg-orange-500";
    default:
      return "bg-gray-400";
  }
}
