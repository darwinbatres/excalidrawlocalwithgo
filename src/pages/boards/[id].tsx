/**
 * Board Editor Page
 *
 * Full-screen Excalidraw editor with:
 * - Autosave (via API)
 * - Version history
 * - Title editing
 */

import React, { useState, useEffect, useCallback } from "react";
import { useRouter } from "next/router";
import Link from "next/link";
import Head from "next/head";
import dynamic from "next/dynamic";
import { toast } from "sonner";
import { useApp } from "@/contexts/AppContext";
import { boardApi, ApiError, type BoardWithScene } from "@/services/api.client";
import { Button } from "@/components/ui/Button";
import { Spinner } from "@/components/ui/Spinner";
import { ShareDialog } from "@/components/ui/ShareDialog";
import { formatApiError } from "@/lib/hooks";

// Dynamic import for BoardEditor (needs client-side only due to Excalidraw)
const BoardEditor = dynamic(
  () =>
    import("@/components/excalidraw/BoardEditor").then(
      (mod) => mod.BoardEditor
    ),
  {
    ssr: false,
    loading: () => (
      <div className="flex items-center justify-center h-full bg-white dark:bg-gray-950">
        <div className="text-center">
          <Spinner />
          <p className="mt-4 text-gray-600 dark:text-gray-400">
            Loading editor...
          </p>
        </div>
      </div>
    ),
  }
);

export default function BoardPage() {
  const router = useRouter();
  const { id } = router.query;
  const { isLoading: authLoading, isAuthenticated } = useApp();

  const [board, setBoard] = useState<BoardWithScene | null>(null);
  const [isEditing, setIsEditing] = useState(false);
  const [editTitle, setEditTitle] = useState("");
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [viewMode, setViewMode] = useState(false);
  // eslint-disable-next-line @typescript-eslint/no-unused-vars
  const [saving, setSaving] = useState(false);
  const [showShare, setShowShare] = useState(false);

  // Load board from API
  const loadBoard = useCallback(async () => {
    if (!id || typeof id !== "string") return;

    setIsLoading(true);
    setError(null);

    try {
      const boardData = await boardApi.get(id);
      setBoard(boardData);
      setEditTitle(boardData.title);
    } catch (err) {
      console.error("Failed to load board:", err);
      if (err instanceof ApiError && err.status === 404) {
        setError("Board not found");
      } else {
        setError(
          err instanceof ApiError ? err.message : "Failed to load board"
        );
      }
    } finally {
      setIsLoading(false);
    }
  }, [id]);

  useEffect(() => {
    if (isAuthenticated && id) {
      loadBoard();
    } else if (!authLoading && !isAuthenticated) {
      setIsLoading(false);
    }
  }, [loadBoard, isAuthenticated, authLoading, id]);

  // Handle title save
  const handleSaveTitle = async () => {
    if (!board || !editTitle.trim()) return;

    setSaving(true);
    try {
      const newTitle = editTitle.trim();
      const updated = await boardApi.update(board.id, {
        title: newTitle,
      });
      setBoard({ ...board, ...updated });
      setIsEditing(false);
      toast("Board renamed", {
        description: newTitle,
      });
    } catch (err) {
      console.error("Failed to update title:", err);
      toast.error("Failed to rename board", {
        description: formatApiError(err, "Please try again"),
      });
    } finally {
      setSaving(false);
    }
  };

  if (authLoading || isLoading) {
    return (
      <div className="h-screen flex items-center justify-center bg-white dark:bg-gray-950">
        <Spinner />
      </div>
    );
  }

  if (!isAuthenticated) {
    return (
      <div className="h-screen flex items-center justify-center bg-white dark:bg-gray-950">
        <div className="text-center">
          <h1 className="text-xl font-semibold text-gray-900 dark:text-gray-100">
            Please sign in
          </h1>
          <p className="mt-2 text-gray-600 dark:text-gray-400">
            You need to be signed in to view this board.
          </p>
          <Link href="/">
            <Button className="mt-4">Go to Home</Button>
          </Link>
        </div>
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
            This board doesn&apos;t exist or you don&apos;t have access to it.
          </p>
          <Link href="/">
            <Button className="mt-4">Go to Home</Button>
          </Link>
        </div>
      </div>
    );
  }

  return (
    <>
      <Head>
        <title>{board.title} - Drawgo</title>
      </Head>

      <div className="h-screen flex flex-col bg-white dark:bg-gray-950">
        {/* Top bar */}
        <div className="h-12 flex items-center justify-between px-4 border-b border-gray-200 dark:border-gray-800 bg-white dark:bg-gray-900 shrink-0">
          {/* Left side */}
          <div className="flex items-center gap-3">
            <Link
              href="/"
              className="p-1.5 text-gray-500 hover:text-gray-700 dark:hover:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-800 rounded transition-colors"
              title="Back to boards"
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
                  d="M10 19l-7-7m0 0l7-7m-7 7h18"
                />
              </svg>
            </Link>

            {/* Title */}
            {isEditing ? (
              <div className="flex items-center gap-2">
                <input
                  type="text"
                  value={editTitle}
                  onChange={(e) => setEditTitle(e.target.value)}
                  onKeyDown={(e) => {
                    if (e.key === "Enter") handleSaveTitle();
                    if (e.key === "Escape") {
                      setEditTitle(board.title);
                      setIsEditing(false);
                    }
                  }}
                  className="px-2 py-1 text-sm font-medium border border-indigo-500 rounded focus:outline-none focus:ring-2 focus:ring-indigo-500 bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100"
                  autoFocus
                />
                <button
                  onClick={handleSaveTitle}
                  className="p-1 text-green-600 hover:bg-green-50 dark:hover:bg-green-900/20 rounded"
                >
                  <svg
                    className="w-4 h-4"
                    fill="currentColor"
                    viewBox="0 0 20 20"
                  >
                    <path
                      fillRule="evenodd"
                      d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z"
                      clipRule="evenodd"
                    />
                  </svg>
                </button>
                <button
                  onClick={() => {
                    setEditTitle(board.title);
                    setIsEditing(false);
                  }}
                  className="p-1 text-gray-500 hover:bg-gray-100 dark:hover:bg-gray-800 rounded"
                >
                  <svg
                    className="w-4 h-4"
                    fill="currentColor"
                    viewBox="0 0 20 20"
                  >
                    <path
                      fillRule="evenodd"
                      d="M4.293 4.293a1 1 0 011.414 0L10 8.586l4.293-4.293a1 1 0 111.414 1.414L11.414 10l4.293 4.293a1 1 0 01-1.414 1.414L10 11.414l-4.293 4.293a1 1 0 01-1.414-1.414L8.586 10 4.293 5.707a1 1 0 010-1.414z"
                      clipRule="evenodd"
                    />
                  </svg>
                </button>
              </div>
            ) : (
              <button
                onClick={() => setIsEditing(true)}
                className="text-sm font-medium text-gray-900 dark:text-gray-100 hover:text-indigo-600 dark:hover:text-indigo-400 transition-colors flex items-center gap-1 group"
              >
                {board.title}
                <svg
                  className="w-3.5 h-3.5 opacity-0 group-hover:opacity-100 transition-opacity"
                  fill="none"
                  stroke="currentColor"
                  viewBox="0 0 24 24"
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={2}
                    d="M15.232 5.232l3.536 3.536m-2.036-5.036a2.5 2.5 0 113.536 3.536L6.5 21.036H3v-3.572L16.732 3.732z"
                  />
                </svg>
              </button>
            )}
          </div>

          {/* Right side */}
          <div className="flex items-center gap-4">
            <button
              onClick={() => setShowShare(true)}
              className="flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium text-gray-700 dark:text-gray-300 bg-gray-100 dark:bg-gray-800 hover:bg-gray-200 dark:hover:bg-gray-700 rounded-lg transition-colors"
            >
              <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8.684 13.342C8.886 12.938 9 12.482 9 12c0-.482-.114-.938-.316-1.342m0 2.684a3 3 0 110-2.684m0 2.684l6.632 3.316m-6.632-6l6.632-3.316m0 0a3 3 0 105.367-2.684 3 3 0 00-5.367 2.684zm0 9.316a3 3 0 105.368 2.684 3 3 0 00-5.368-2.684z" />
              </svg>
              Share
            </button>
            {/* Keyboard shortcuts hint */}
            <div className="text-xs text-gray-400 dark:text-gray-500">
              Press{" "}
              <kbd className="px-1.5 py-0.5 bg-gray-100 dark:bg-gray-800 rounded text-gray-600 dark:text-gray-400">
                ?
              </kbd>{" "}
              for shortcuts
            </div>
          </div>
        </div>

        {/* Editor */}
        <div className="flex-1 relative">
          <BoardEditor
            boardId={board.id}
            viewMode={viewMode}
            onViewModeChange={setViewMode}
          />
        </div>
      </div>

      {board && (
        <ShareDialog
          isOpen={showShare}
          boardId={board.id}
          onClose={() => setShowShare(false)}
        />
      )}
    </>
  );
}
