/**
 * Utility functions for working with cron expressions
 */

import { Cron } from "croner";

/**
 * Convert a cron expression to a human-readable description
 * @param cronExpression The cron expression to describe
 * @returns A human-readable description of the cron expression
 */
export function describeCronExpression(cronExpression: string): string {
  try {
    // Parse the cron expression
    const parts = cronExpression.trim().split(/\s+/);

    if (parts.length < 5) {
      return "Invalid cron expression";
    }

    const [minute, hour, dayOfMonth, month, dayOfWeek] = parts;

    // Check for common patterns
    if (cronExpression === "* * * * *") {
      return "Every minute";
    }

    if (cronExpression === "0 * * * *") {
      return "Every hour";
    }

    if (cronExpression === "0 0 * * *") {
      return "Daily at midnight";
    }

    if (cronExpression === "0 0 * * 0") {
      return "Weekly on Sunday at midnight";
    }

    if (cronExpression === "0 0 1 * *") {
      return "Monthly on the 1st at midnight";
    }

    // Pattern: Every N hours
    if (
      minute === "0" &&
      hour.startsWith("*/") &&
      dayOfMonth === "*" &&
      month === "*" &&
      dayOfWeek === "*"
    ) {
      const hours = hour.substring(2);
      return `Every ${hours} hour${hours !== "1" ? "s" : ""}`;
    }

    // Pattern: Every N minutes
    if (
      minute.startsWith("*/") &&
      hour === "*" &&
      dayOfMonth === "*" &&
      month === "*" &&
      dayOfWeek === "*"
    ) {
      const minutes = minute.substring(2);
      return `Every ${minutes} minute${minutes !== "1" ? "s" : ""}`;
    }

    // Pattern: Every N days at specific time
    if (dayOfMonth.startsWith("*/") && month === "*" && dayOfWeek === "*") {
      const days = dayOfMonth.substring(2);
      const time = formatTime(hour, minute);
      return `Every ${days} day${days !== "1" ? "s" : ""} at ${time}`;
    }

    // Pattern: Specific hour daily
    if (
      minute === "0" &&
      !hour.includes("*") &&
      dayOfMonth === "*" &&
      month === "*" &&
      dayOfWeek === "*"
    ) {
      const time = formatTime(hour, minute);
      return `Daily at ${time}`;
    }

    // Pattern: Specific day of week
    if (dayOfMonth === "*" && month === "*" && !dayOfWeek.includes("*")) {
      const time = formatTime(hour, minute);
      const day = getDayName(dayOfWeek);
      return `Weekly on ${day} at ${time}`;
    }

    // Pattern: Specific day of month
    if (!dayOfMonth.includes("*") && month === "*" && dayOfWeek === "*") {
      const time = formatTime(hour, minute);
      return `Monthly on day ${dayOfMonth} at ${time}`;
    }

    // Generic fallback
    return `At ${formatTime(hour, minute)}`;
  } catch (error) {
    return "Invalid cron expression";
  }
}

/**
 * Format hour and minute into readable time
 */
function formatTime(hour: string, minute: string): string {
  if (hour === "*" && minute === "*") {
    return "every minute";
  }

  if (hour === "*") {
    return `minute ${minute} of every hour`;
  }

  const h = parseInt(hour);
  const m = parseInt(minute);

  if (isNaN(h) || isNaN(m)) {
    return `${hour}:${minute}`;
  }

  const period = h >= 12 ? "PM" : "AM";
  const displayHour = h === 0 ? 12 : h > 12 ? h - 12 : h;
  const displayMinute = m.toString().padStart(2, "0");

  return `${displayHour}:${displayMinute} ${period}`;
}

/**
 * Get day name from cron day of week value
 */
function getDayName(dayOfWeek: string): string {
  const days: { [key: string]: string } = {
    "0": "Sunday",
    "1": "Monday",
    "2": "Tuesday",
    "3": "Wednesday",
    "4": "Thursday",
    "5": "Friday",
    "6": "Saturday",
    "7": "Sunday",
  };

  return days[dayOfWeek] || `day ${dayOfWeek}`;
}

/**
 * Get the next run time for a cron expression
 * @param cronExpression The cron expression
 * @returns The next run time as a Date, or null if invalid
 */
export function getNextRunTime(cronExpression: string): Date | null {
  try {
    const cron = new Cron(cronExpression);
    return cron.nextRun() || null;
  } catch {
    return null;
  }
}

/**
 * Format the time until next run in human-readable format
 * @param nextRun The next run time
 * @returns A human-readable string describing when the next run will occur
 */
export function formatNextRunTime(nextRun: Date): string {
  const now = new Date();
  const diff = nextRun.getTime() - now.getTime();
  const seconds = Math.floor(diff / 1000);
  const minutes = Math.floor(seconds / 60);
  const hours = Math.floor(minutes / 60);
  const days = Math.floor(hours / 24);

  if (diff < 0) {
    return "Overdue";
  }

  if (days > 0) {
    const remainingHours = hours % 24;
    if (remainingHours > 0) {
      return `in ${days} day${days > 1 ? "s" : ""} and ${remainingHours} hour${remainingHours > 1 ? "s" : ""}`;
    }
    return `in ${days} day${days > 1 ? "s" : ""}`;
  }

  if (hours > 0) {
    const remainingMinutes = minutes % 60;
    if (remainingMinutes > 0) {
      return `in ${hours} hour${hours > 1 ? "s" : ""} and ${remainingMinutes} minute${remainingMinutes > 1 ? "s" : ""}`;
    }
    return `in ${hours} hour${hours > 1 ? "s" : ""}`;
  }

  if (minutes > 0) {
    return `in ${minutes} minute${minutes > 1 ? "s" : ""}`;
  }

  return `in ${seconds} second${seconds !== 1 ? "s" : ""}`;
}

/**
 * Validate a cron expression
 * @param cronExpression The cron expression to validate
 * @returns true if valid, false otherwise
 */
export function validateCronExpression(cronExpression: string): boolean {
  try {
    new Cron(cronExpression);
    return true;
  } catch {
    return false;
  }
}
