/**
 * LiveCursors — Renders remote user cursors on the canvas.
 *
 * Receives cursor positions from WebSocket and renders colored
 * cursors with user name labels. Uses requestAnimationFrame-based
 * interpolation for butter-smooth 60fps cursor movement.
 */

import React, { useMemo, useRef, useEffect, useCallback, useState } from "react";
import type { CursorPayload } from "@/types";

/** Interpolation factor per frame — higher = snappier, lower = smoother */
const LERP_FACTOR = 0.35;
/** Below this distance (px), snap instantly */
const SNAP_THRESHOLD = 0.5;

interface InterpolatedCursor extends CursorPayload {
  /** Current rendered X (interpolated) */
  renderX: number;
  /** Current rendered Y (interpolated) */
  renderY: number;
}

interface LiveCursorsProps {
  cursors: Map<string, CursorPayload>;
  /** Current user ID — excluded from rendering */
  currentUserId?: string;
}

export function LiveCursors({ cursors, currentUserId }: LiveCursorsProps) {
  // Track interpolated positions across frames
  const interpRef = useRef<Map<string, InterpolatedCursor>>(new Map());
  const rafRef = useRef<number>(0);
  const containerRef = useRef<HTMLDivElement>(null);
  const cursorEls = useRef<Map<string, HTMLDivElement>>(new Map());
  const [, forceRender] = useState(0);

  // Build list of remote cursors from props
  const remoteCursors = useMemo(() => {
    const result: CursorPayload[] = [];
    cursors.forEach((cursor) => {
      if (cursor.userId !== currentUserId) {
        result.push(cursor);
      }
    });
    return result;
  }, [cursors, currentUserId]);

  // Update target positions when new data arrives
  useEffect(() => {
    const interp = interpRef.current;
    const activeIds = new Set<string>();

    for (const cursor of remoteCursors) {
      activeIds.add(cursor.userId);
      const existing = interp.get(cursor.userId);
      if (existing) {
        // Update target — keep current render position for smooth interpolation
        existing.x = cursor.x;
        existing.y = cursor.y;
        existing.name = cursor.name;
        existing.color = cursor.color;
      } else {
        // New cursor — start at target position (no interpolation for first frame)
        interp.set(cursor.userId, {
          ...cursor,
          renderX: cursor.x,
          renderY: cursor.y,
        });
      }
    }

    // Remove cursors that are no longer present
    for (const id of interp.keys()) {
      if (!activeIds.has(id)) {
        interp.delete(id);
        cursorEls.current.delete(id);
      }
    }

    // Trigger a re-render to create/remove DOM elements
    forceRender((n) => n + 1);
  }, [remoteCursors]);

  // Animation loop — interpolates positions at display refresh rate
  const animate = useCallback(() => {
    const interp = interpRef.current;
    let needsNextFrame = false;

    for (const [id, cursor] of interp) {
      const dx = cursor.x - cursor.renderX;
      const dy = cursor.y - cursor.renderY;
      const dist = Math.abs(dx) + Math.abs(dy);

      if (dist > SNAP_THRESHOLD) {
        cursor.renderX += dx * LERP_FACTOR;
        cursor.renderY += dy * LERP_FACTOR;
        needsNextFrame = true;
      } else {
        cursor.renderX = cursor.x;
        cursor.renderY = cursor.y;
      }

      // Direct DOM manipulation to avoid React re-renders at 60fps
      const el = cursorEls.current.get(id);
      if (el) {
        el.style.transform = `translate3d(${cursor.renderX}px, ${cursor.renderY}px, 0)`;
      }
    }

    if (needsNextFrame) {
      rafRef.current = requestAnimationFrame(animate);
    }
  }, []);

  // Start/stop animation loop when cursors exist
  useEffect(() => {
    if (interpRef.current.size > 0) {
      rafRef.current = requestAnimationFrame(animate);
    }
    return () => {
      if (rafRef.current) {
        cancelAnimationFrame(rafRef.current);
      }
    };
  }, [animate, remoteCursors]);

  // Kick animation when new cursor data arrives
  useEffect(() => {
    if (interpRef.current.size > 0) {
      cancelAnimationFrame(rafRef.current);
      rafRef.current = requestAnimationFrame(animate);
    }
  }, [remoteCursors, animate]);

  if (remoteCursors.length === 0) return null;

  return (
    <div
      ref={containerRef}
      className="pointer-events-none fixed inset-0 z-50 overflow-hidden"
      aria-hidden="true"
    >
      {remoteCursors.map((cursor) => (
        <div
          key={cursor.userId}
          ref={(el) => {
            if (el) cursorEls.current.set(cursor.userId, el);
          }}
          className="absolute will-change-transform"
          style={{
            transform: `translate3d(${cursor.x}px, ${cursor.y}px, 0)`,
          }}
        >
          {/* Cursor arrow */}
          <svg
            width="20"
            height="20"
            viewBox="0 0 20 20"
            fill={cursor.color}
            className="drop-shadow-sm"
          >
            <path d="M3 3l14 6.5L10.5 12 9 17.5z" />
          </svg>
          {/* Name label */}
          <div
            className="absolute left-4 top-4 whitespace-nowrap rounded px-1.5 py-0.5 text-xs font-medium text-white shadow-sm"
            style={{ backgroundColor: cursor.color }}
          >
            {cursor.name}
          </div>
        </div>
      ))}
    </div>
  );
}
