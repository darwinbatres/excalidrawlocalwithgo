/**
 * Shared Board Page — public access via share token
 */

import React, { useEffect, useState } from "react";
import { useRouter } from "next/router";
import Head from "next/head";
import dynamic from "next/dynamic";
import { shareApi, ApiError, type BoardWithScene } from "@/services/api.client";
import { Spinner } from "@/components/ui/Spinner";
import type { BoardRole } from "@/types";

const BoardEditor = dynamic(
  () =>
    import("@/components/excalidraw/BoardEditor").then(
      (mod) => mod.BoardEditor
    ),
  { ssr: false }
);

export default function SharedBoardPage() {
  const router = useRouter();
  const { token } = router.query;

  const [board, setBoard] = useState<BoardWithScene | null>(null);
  const [role, setRole] = useState<BoardRole>("VIEWER");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [viewMode, setViewMode] = useState(false);

  useEffect(() => {
    if (!token || typeof token !== "string") return;

    async function load() {
      try {
        setLoading(true);
        const data = await shareApi.getShared(token as string);
        // Backend returns BoardWithVersion which matches BoardWithScene shape
        setBoard(data.board as BoardWithScene);
        setRole(data.shareRole);
        setViewMode(data.shareRole !== "EDITOR");
      } catch (err) {
        if (err instanceof ApiError) {
          setError(
            err.status === 404
              ? "This share link is invalid or has expired."
              : err.message
          );
        } else {
          setError("Failed to load shared board");
        }
      } finally {
        setLoading(false);
      }
    }

    load();
  }, [token]);

  if (loading) {
    return (
      <div className="h-screen flex items-center justify-center bg-white dark:bg-gray-950">
        <Spinner size="lg" />
      </div>
    );
  }

  if (error || !board) {
    return (
      <div className="h-screen flex items-center justify-center bg-white dark:bg-gray-950">
        <div className="text-center">
          <h1 className="text-xl font-semibold text-gray-900 dark:text-gray-100">
            {error || "Board not found"}
          </h1>
          <p className="mt-2 text-gray-600 dark:text-gray-400">
            Check that your share link is correct.
          </p>
        </div>
      </div>
    );
  }

  return (
    <>
      <Head>
        <title>{board.title} (Shared) - Drawgo</title>
      </Head>

      <div className="h-screen flex flex-col bg-white dark:bg-gray-950">
        {/* Minimal top bar */}
        <div className="h-10 flex items-center justify-between px-4 border-b border-gray-200 dark:border-gray-800 bg-white dark:bg-gray-900 shrink-0">
          <span className="text-sm font-medium text-gray-900 dark:text-gray-100 truncate">
            {board.title}
          </span>
          <span className="text-xs px-2 py-0.5 rounded bg-gray-100 dark:bg-gray-800 text-gray-500 dark:text-gray-400">
            {role === "EDITOR" ? "Edit access" : "View only"}
          </span>
        </div>

        <div className="flex-1 relative">
          <BoardEditor
            boardId={board.id}
            viewMode={viewMode}
            onViewModeChange={role === "EDITOR" ? (v: boolean) => setViewMode(v) : undefined}
            preloadedBoard={board}
            shareToken={token as string}
          />
        </div>
      </div>
    </>
  );
}
