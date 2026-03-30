/**
 * PresenceBar — Shows who is currently viewing/editing the board.
 *
 * Displays colored avatar circles with tooltip names.
 * Positioned in the top-right of the board editor.
 */

import React, { useMemo } from "react";
import type { ViewerInfo } from "@/types";

interface PresenceBarProps {
  viewers: ViewerInfo[];
  /** Current user ID — shown with a border indicator */
  currentUserId?: string;
}

/** Max avatars before "+N" overflow */
const MAX_VISIBLE = 5;

export function PresenceBar({ viewers, currentUserId }: PresenceBarProps) {
  const sorted = useMemo(() => {
    // Current user first, then alphabetical
    return [...viewers].sort((a, b) => {
      if (a.userId === currentUserId) return -1;
      if (b.userId === currentUserId) return 1;
      return (a.name || "").localeCompare(b.name || "");
    });
  }, [viewers, currentUserId]);

  if (sorted.length === 0) return null;

  const visible = sorted.slice(0, MAX_VISIBLE);
  const overflow = sorted.length - MAX_VISIBLE;

  return (
    <div className="absolute top-3 right-3 z-30 flex items-center gap-1">
      <div className="flex -space-x-2">
        {visible.map((viewer) => (
          <div
            key={viewer.userId}
            className={`relative w-8 h-8 rounded-full flex items-center justify-center text-xs font-semibold text-white ring-2 ring-white dark:ring-gray-900 ${
              viewer.userId === currentUserId ? "ring-indigo-500" : ""
            }`}
            style={{ backgroundColor: viewer.color }}
            title={`${viewer.name}${viewer.userId === currentUserId ? " (you)" : ""} — ${viewer.role}`}
          >
            {getInitials(viewer.name)}
          </div>
        ))}
      </div>

      {overflow > 0 && (
        <div
          className="w-8 h-8 rounded-full bg-gray-200 dark:bg-gray-700 flex items-center justify-center text-xs font-medium text-gray-600 dark:text-gray-300 ring-2 ring-white dark:ring-gray-900"
          title={`${overflow} more viewer${overflow > 1 ? "s" : ""}`}
        >
          +{overflow}
        </div>
      )}
    </div>
  );
}

function getInitials(name: string): string {
  if (!name) return "?";
  const parts = name.trim().split(/\s+/);
  if (parts.length >= 2) {
    return (parts[0][0] + parts[parts.length - 1][0]).toUpperCase();
  }
  return name.slice(0, 2).toUpperCase();
}
