"use client";

import { useState, useEffect, useRef, useCallback } from "react";
import { useParams, useRouter } from "next/navigation";
import Link from "next/link";
import {
  getCIJob,
  getCIJobLogs,
  cancelCIJob,
  retryCIJob,
  subscribeToCIJobStream,
  getCIStatusColor,
  getCIStatusBgColor,
  formatCIDuration,
  getArtifactDownloadUrl,
} from "@/lib/api";
import { CIJob, CIJobLog, CIJobEvent } from "@/lib/types";

// Status icon component
function StatusIcon({
  status,
  size = "md",
}: {
  status: string;
  size?: "sm" | "md" | "lg";
}) {
  const sizeClass =
    size === "sm" ? "w-3 h-3" : size === "lg" ? "w-6 h-6" : "w-4 h-4";

  switch (status) {
    case "success":
      return (
        <svg
          className={sizeClass}
          fill="none"
          stroke="currentColor"
          viewBox="0 0 24 24"
        >
          <path
            strokeLinecap="round"
            strokeLinejoin="round"
            strokeWidth={2}
            d="M5 13l4 4L19 7"
          />
        </svg>
      );
    case "failed":
    case "error":
      return (
        <svg
          className={sizeClass}
          fill="none"
          stroke="currentColor"
          viewBox="0 0 24 24"
        >
          <path
            strokeLinecap="round"
            strokeLinejoin="round"
            strokeWidth={2}
            d="M6 18L18 6M6 6l12 12"
          />
        </svg>
      );
    case "running":
      return (
        <svg
          className={`${sizeClass} animate-spin`}
          fill="none"
          stroke="currentColor"
          viewBox="0 0 24 24"
        >
          <circle
            className="opacity-25"
            cx="12"
            cy="12"
            r="10"
            stroke="currentColor"
            strokeWidth="4"
          />
          <path
            className="opacity-75"
            fill="currentColor"
            d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"
          />
        </svg>
      );
    case "pending":
    case "queued":
      return (
        <svg
          className={sizeClass}
          fill="none"
          stroke="currentColor"
          viewBox="0 0 24 24"
        >
          <circle cx="12" cy="12" r="10" strokeWidth={2} />
          <path
            strokeLinecap="round"
            strokeLinejoin="round"
            strokeWidth={2}
            d="M12 6v6l4 2"
          />
        </svg>
      );
    case "cancelled":
      return (
        <svg
          className={sizeClass}
          fill="none"
          stroke="currentColor"
          viewBox="0 0 24 24"
        >
          <circle cx="12" cy="12" r="10" strokeWidth={2} />
          <path
            strokeLinecap="round"
            strokeLinejoin="round"
            strokeWidth={2}
            d="M9 9h6v6H9z"
          />
        </svg>
      );
    default:
      return (
        <svg
          className={sizeClass}
          fill="none"
          stroke="currentColor"
          viewBox="0 0 24 24"
        >
          <circle cx="12" cy="12" r="10" strokeWidth={2} />
        </svg>
      );
  }
}

// Log level badge colors
function getLogLevelColor(level: string): string {
  switch (level) {
    case "error":
      return "text-red-400";
    case "warning":
    case "warn":
      return "text-yellow-400";
    case "debug":
      return "text-gray-400";
    default:
      return "text-gray-300";
  }
}

// Format timestamp for logs
function formatLogTime(timestamp: string): string {
  const date = new Date(timestamp);
  return date.toLocaleTimeString("en-US", {
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
    hour12: false,
  });
}

// Format relative time
function formatRelativeTime(dateStr: string): string {
  const date = new Date(dateStr);
  const now = new Date();
  const diffMs = now.getTime() - date.getTime();
  const diffSecs = Math.floor(diffMs / 1000);
  const diffMins = Math.floor(diffSecs / 60);
  const diffHours = Math.floor(diffMins / 60);
  const diffDays = Math.floor(diffHours / 24);

  if (diffSecs < 60) return "just now";
  if (diffMins < 60) return `${diffMins}m ago`;
  if (diffHours < 24) return `${diffHours}h ago`;
  if (diffDays < 7) return `${diffDays}d ago`;
  return date.toLocaleDateString();
}

export default function CIJobDetailPage() {
  const params = useParams();
  const router = useRouter();
  const username = params.username as string;
  const repo = params.repo as string;
  const jobId = params.jobId as string;

  const [job, setJob] = useState<CIJob | null>(null);
  const [logs, setLogs] = useState<CIJobLog[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [isStreaming, setIsStreaming] = useState(false);
  const [autoScroll, setAutoScroll] = useState(true);
  const [actionLoading, setActionLoading] = useState<"cancel" | "retry" | null>(
    null,
  );
  const [selectedStep, setSelectedStep] = useState<string | null>(null);

  const logsEndRef = useRef<HTMLDivElement>(null);
  const logsContainerRef = useRef<HTMLDivElement>(null);
  const eventSourceRef = useRef<EventSource | null>(null);

  // Scroll to bottom when new logs arrive
  useEffect(() => {
    if (autoScroll && logsEndRef.current) {
      logsEndRef.current.scrollIntoView({ behavior: "smooth" });
    }
  }, [logs, autoScroll]);

  // Handle scroll to detect if user scrolled up
  const handleScroll = useCallback(() => {
    if (!logsContainerRef.current) return;
    const { scrollTop, scrollHeight, clientHeight } = logsContainerRef.current;
    const isAtBottom = scrollHeight - scrollTop - clientHeight < 50;
    setAutoScroll(isAtBottom);
  }, []);

  // Fetch initial job data
  useEffect(() => {
    async function fetchJob() {
      try {
        setLoading(true);
        setError(null);

        const [jobData, logsData] = await Promise.all([
          getCIJob(username, repo, jobId),
          getCIJobLogs(username, repo, jobId, 10000, 0),
        ]);

        setJob(jobData);
        setLogs(logsData.logs);
      } catch (err) {
        console.error("Failed to fetch job:", err);
        setError(err instanceof Error ? err.message : "Failed to load job");
      } finally {
        setLoading(false);
      }
    }

    fetchJob();
  }, [username, repo, jobId]);

  // Set up SSE streaming for running jobs
  useEffect(() => {
    if (!job) return;

    const isRunning =
      job.status === "running" ||
      job.status === "pending" ||
      job.status === "queued";

    if (!isRunning) {
      setIsStreaming(false);
      if (eventSourceRef.current) {
        eventSourceRef.current.close();
        eventSourceRef.current = null;
      }
      return;
    }

    setIsStreaming(true);

    const eventSource = subscribeToCIJobStream(
      username,
      repo,
      jobId,
      (event: CIJobEvent) => {
        switch (event.type) {
          case "log":
            const logData = event.data as CIJobLog;
            setLogs((prev) => {
              // Check if log already exists
              if (prev.some((l) => l.sequence === logData.sequence)) {
                return prev;
              }
              // Add and sort by sequence
              return [...prev, logData].sort((a, b) => a.sequence - b.sequence);
            });
            break;
          case "status":
            // Refresh job data on status change
            getCIJob(username, repo, jobId).then(setJob).catch(console.error);
            break;
        }
      },
      (error) => {
        console.error("SSE error:", error);
        setIsStreaming(false);
      },
    );

    eventSourceRef.current = eventSource;

    return () => {
      eventSource.close();
      eventSourceRef.current = null;
    };
  }, [job?.status, username, repo, jobId]);

  // Handle cancel action
  const handleCancel = async () => {
    if (!confirm("Are you sure you want to cancel this job?")) return;

    try {
      setActionLoading("cancel");
      await cancelCIJob(username, repo, jobId);
      // Refresh job data
      const updatedJob = await getCIJob(username, repo, jobId);
      setJob(updatedJob);
    } catch (err) {
      console.error("Failed to cancel job:", err);
      alert(err instanceof Error ? err.message : "Failed to cancel job");
    } finally {
      setActionLoading(null);
    }
  };

  // Handle retry action
  const handleRetry = async () => {
    try {
      setActionLoading("retry");
      const result = await retryCIJob(username, repo, jobId);
      // Navigate to new job
      router.push(`/${username}/${repo}/ci/${result.new_job_id}`);
    } catch (err) {
      console.error("Failed to retry job:", err);
      alert(err instanceof Error ? err.message : "Failed to retry job");
    } finally {
      setActionLoading(null);
    }
  };

  // Filter logs by selected step
  const filteredLogs = selectedStep
    ? logs.filter((log) => log.step_name === selectedStep)
    : logs;

  // Get unique step names from logs
  const stepNames = [
    ...new Set(logs.filter((l) => l.step_name).map((l) => l.step_name!)),
  ];

  if (loading) {
    return (
      <div className="flex items-center justify-center py-20">
        <div className="animate-spin rounded-full h-8 w-8 border-t-2 border-b-2 border-accent"></div>
      </div>
    );
  }

  if (error || !job) {
    return (
      <div className="bg-red-500/10 border border-red-500/30 rounded-md p-4 text-red-500">
        <p className="font-medium">Error loading CI job</p>
        <p className="text-sm mt-1">{error || "Job not found"}</p>
        <Link
          href={`/${username}/${repo}/ci`}
          className="inline-block mt-4 text-sm text-accent hover:underline"
        >
          ← Back to CI jobs
        </Link>
      </div>
    );
  }

  const isRunning =
    job.status === "running" ||
    job.status === "pending" ||
    job.status === "queued";
  const canRetry =
    job.status === "failed" ||
    job.status === "error" ||
    job.status === "cancelled" ||
    job.status === "timed_out";

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-start justify-between">
        <div>
          <div className="flex items-center gap-3 mb-2">
            <Link
              href={`/${username}/${repo}/ci`}
              className="text-muted hover:text-base transition-colors"
            >
              <svg
                className="w-5 h-5"
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M15 19l-7-7 7-7"
                />
              </svg>
            </Link>
            <h1 className="text-xl font-semibold text-base">
              Job #{job.id.substring(0, 8)}
            </h1>
            <span
              className={`inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-sm font-medium border ${getCIStatusBgColor(
                job.status,
              )} ${getCIStatusColor(job.status)}`}
            >
              <StatusIcon status={job.status} />
              {job.status}
            </span>
            {isStreaming && (
              <span className="inline-flex items-center gap-1 text-xs text-green-500">
                <span className="w-2 h-2 bg-green-500 rounded-full animate-pulse"></span>
                Live
              </span>
            )}
          </div>
          <p className="text-sm text-muted">
            Triggered by <span className="text-base">{job.trigger_actor}</span>{" "}
            via <span className="text-base">{job.trigger_type}</span>
            {job.started_at && (
              <>
                {" • "}
                Started {formatRelativeTime(job.started_at)}
              </>
            )}
            {job.duration_seconds !== undefined && (
              <>
                {" • "}
                Duration: {formatCIDuration(job.duration_seconds)}
              </>
            )}
          </p>
        </div>

        <div className="flex items-center gap-2">
          {isRunning && (
            <button
              onClick={handleCancel}
              disabled={actionLoading !== null}
              className="px-4 py-2 text-sm font-medium text-red-500 bg-red-500/10 border border-red-500/30 rounded-md hover:bg-red-500/20 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
            >
              {actionLoading === "cancel" ? "Cancelling..." : "Cancel"}
            </button>
          )}
          {canRetry && (
            <button
              onClick={handleRetry}
              disabled={actionLoading !== null}
              className="px-4 py-2 text-sm font-medium text-accent bg-accent/10 border border-accent/30 rounded-md hover:bg-accent/20 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
            >
              {actionLoading === "retry" ? "Retrying..." : "Retry"}
            </button>
          )}
        </div>
      </div>

      {/* Job Info */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        <div className="bg-panel border border-base rounded-md p-4">
          <div className="text-xs text-muted uppercase tracking-wide mb-1">
            Commit
          </div>
          <code className="text-sm font-mono text-accent">
            {job.commit_sha.substring(0, 7)}
          </code>
        </div>
        <div className="bg-panel border border-base rounded-md p-4">
          <div className="text-xs text-muted uppercase tracking-wide mb-1">
            {job.ref_type === "tag" ? "Tag" : "Branch"}
          </div>
          <span className="text-sm text-base">{job.ref_name}</span>
        </div>
        <div className="bg-panel border border-base rounded-md p-4">
          <div className="text-xs text-muted uppercase tracking-wide mb-1">
            Run ID
          </div>
          <code className="text-sm font-mono text-muted">
            {job.run_id.substring(0, 8)}
          </code>
        </div>
        <div className="bg-panel border border-base rounded-md p-4">
          <div className="text-xs text-muted uppercase tracking-wide mb-1">
            Config
          </div>
          <code className="text-sm font-mono text-muted">
            {job.config_path}
          </code>
        </div>
      </div>

      {/* Error message */}
      {job.error && (
        <div className="bg-red-500/10 border border-red-500/30 rounded-md p-4 text-red-500">
          <p className="font-medium">Error</p>
          <p className="text-sm mt-1">{job.error}</p>
        </div>
      )}

      {/* Steps */}
      {job.steps && job.steps.length > 0 && (
        <div className="bg-panel border border-base rounded-md p-4">
          <h3 className="text-sm font-medium text-base mb-3">Steps</h3>
          <div className="flex flex-wrap gap-2">
            {job.steps.map((step) => (
              <button
                key={step.id}
                onClick={() =>
                  setSelectedStep(selectedStep === step.name ? null : step.name)
                }
                className={`inline-flex items-center gap-2 px-3 py-1.5 text-sm rounded-md border transition-colors ${
                  selectedStep === step.name
                    ? "bg-accent/20 border-accent/50 text-accent"
                    : "bg-base border-base hover:border-accent/30"
                }`}
              >
                <StatusIcon status={step.status} size="sm" />
                <span>{step.name}</span>
                {step.duration_seconds !== undefined && (
                  <span className="text-xs text-muted">
                    ({formatCIDuration(step.duration_seconds)})
                  </span>
                )}
              </button>
            ))}
          </div>
        </div>
      )}

      {/* Step filter from logs (if no steps in job data) */}
      {(!job.steps || job.steps.length === 0) && stepNames.length > 0 && (
        <div className="flex items-center gap-2 flex-wrap">
          <span className="text-sm text-muted">Filter by step:</span>
          <button
            onClick={() => setSelectedStep(null)}
            className={`px-3 py-1 text-sm rounded-md border transition-colors ${
              selectedStep === null
                ? "bg-accent/20 border-accent/50 text-accent"
                : "bg-panel border-base hover:border-accent/30"
            }`}
          >
            All
          </button>
          {stepNames.map((step) => (
            <button
              key={step}
              onClick={() =>
                setSelectedStep(selectedStep === step ? null : step)
              }
              className={`px-3 py-1 text-sm rounded-md border transition-colors ${
                selectedStep === step
                  ? "bg-accent/20 border-accent/50 text-accent"
                  : "bg-panel border-base hover:border-accent/30"
              }`}
            >
              {step}
            </button>
          ))}
        </div>
      )}

      {/* Logs */}
      <div className="bg-gray-900 border border-base rounded-md overflow-hidden">
        <div className="flex items-center justify-between px-4 py-2 bg-gray-800 border-b border-gray-700">
          <h3 className="text-sm font-medium text-gray-300">
            Logs
            {selectedStep && (
              <span className="text-muted"> - {selectedStep}</span>
            )}
          </h3>
          <div className="flex items-center gap-3">
            <span className="text-xs text-gray-500">
              {filteredLogs.length} lines
            </span>
            <label className="inline-flex items-center gap-2 text-xs text-gray-400 cursor-pointer">
              <input
                type="checkbox"
                checked={autoScroll}
                onChange={(e) => setAutoScroll(e.target.checked)}
                className="rounded border-gray-600 bg-gray-700 text-accent focus:ring-accent focus:ring-offset-gray-900"
              />
              Auto-scroll
            </label>
          </div>
        </div>

        <div
          ref={logsContainerRef}
          onScroll={handleScroll}
          className="h-[500px] overflow-y-auto font-mono text-sm"
        >
          {filteredLogs.length === 0 ? (
            <div className="flex items-center justify-center h-full text-gray-500">
              {isRunning ? "Waiting for logs..." : "No logs available"}
            </div>
          ) : (
            <table className="w-full">
              <tbody>
                {filteredLogs.map((log, index) => (
                  <tr
                    key={`${log.sequence}-${index}`}
                    className="hover:bg-gray-800/50"
                  >
                    <td className="px-3 py-0.5 text-gray-500 text-xs whitespace-nowrap select-none w-20">
                      {formatLogTime(log.timestamp)}
                    </td>
                    {log.step_name && !selectedStep && (
                      <td className="px-2 py-0.5 text-gray-400 text-xs whitespace-nowrap w-24 truncate">
                        {log.step_name}
                      </td>
                    )}
                    <td
                      className={`px-3 py-0.5 whitespace-pre-wrap break-all ${getLogLevelColor(log.level)}`}
                    >
                      {log.message}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
          <div ref={logsEndRef} />
        </div>
      </div>

      {/* Artifacts */}
      {job.artifacts && job.artifacts.length > 0 && (
        <div className="bg-panel border border-base rounded-md p-4">
          <h3 className="text-sm font-medium text-base mb-3">Artifacts</h3>
          <div className="space-y-2">
            {job.artifacts.map((artifact) => (
              <div
                key={artifact.id}
                className="flex items-center justify-between p-3 bg-base border border-base rounded-md"
              >
                <div className="flex items-center gap-3">
                  <svg
                    className="w-5 h-5 text-muted"
                    fill="none"
                    stroke="currentColor"
                    viewBox="0 0 24 24"
                  >
                    <path
                      strokeLinecap="round"
                      strokeLinejoin="round"
                      strokeWidth={2}
                      d="M7 21h10a2 2 0 002-2V9.414a1 1 0 00-.293-.707l-5.414-5.414A1 1 0 0012.586 3H7a2 2 0 00-2 2v14a2 2 0 002 2z"
                    />
                  </svg>
                  <div>
                    <div className="text-sm font-medium text-base">
                      {artifact.name}
                    </div>
                    <div className="text-xs text-muted">
                      {(artifact.size / 1024).toFixed(1)} KB
                    </div>
                  </div>
                </div>
                <a
                  href={getArtifactDownloadUrl(
                    username,
                    repo,
                    jobId,
                    artifact.name,
                  )}
                  download={artifact.name}
                  className="px-3 py-1 text-sm text-accent bg-accent/10 border border-accent/30 rounded-md hover:bg-accent/20 transition-colors"
                >
                  Download
                </a>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}
