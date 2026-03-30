"use client";

import React, {
  useState,
  useCallback,
  useMemo,
  useRef,
  useEffect,
} from "react";
import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";
import { MermaidRenderer } from "./MermaidRenderer";
import {
  MAX_MARKDOWN_CARD_SIZE,
  formatBytes,
  getContentSize,
} from "@/lib/constants";

interface MarkdownCardEditorProps {
  isOpen: boolean;
  initialMarkdown: string;
  onSave: (markdown: string) => void;
  onClose: () => void;
}

const SAMPLE_MARKDOWN = `# Welcome to Markdown Cards

This is a **markdown card** with full support for:

## Features

- **Bold** and *italic* text
- [Links](https://example.com)
- Lists (ordered and unordered)
- Tables
- Code blocks
- And more!

## Code Example

\`\`\`javascript
function hello() {
  console.log("Hello, world!");
}
\`\`\`

## Mermaid Diagrams

\`\`\`mermaid
graph TD
    A[Start] --> B{Decision}
    B -->|Yes| C[Do Something]
    B -->|No| D[Do Nothing]
    C --> E[End]
    D --> E
\`\`\`

## Table Example

| Feature | Supported |
|---------|-----------|
| GFM     | ✅        |
| Tables  | ✅        |
| Mermaid | ✅        |

> This is a blockquote. Great for callouts!

---

*Double-click the card on canvas to edit.*
`;

export default function MarkdownCardEditor({
  isOpen,
  initialMarkdown,
  onSave,
  onClose,
}: MarkdownCardEditorProps) {
  const [markdown, setMarkdown] = useState(initialMarkdown);
  const [activeTab, setActiveTab] = useState<"edit" | "preview">("edit");
  const textareaRef = useRef<HTMLTextAreaElement>(null);

  // Calculate content size
  const contentSize = useMemo(() => getContentSize(markdown), [markdown]);
  const isOverLimit = contentSize > MAX_MARKDOWN_CARD_SIZE;
  const sizePercentage = Math.min(
    100,
    (contentSize / MAX_MARKDOWN_CARD_SIZE) * 100
  );
  const isNearLimit = sizePercentage > 80;

  useEffect(() => {
    setMarkdown(initialMarkdown);
  }, [initialMarkdown]);

  useEffect(() => {
    if (isOpen && textareaRef.current) {
      textareaRef.current.focus();
    }
  }, [isOpen]);

  const handleSave = useCallback(() => {
    // Prevent saving if content is too large
    if (isOverLimit) {
      return;
    }
    onSave(markdown);
    onClose();
  }, [markdown, onSave, onClose, isOverLimit]);

  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      if (e.key === "Escape") {
        onClose();
      }
      if ((e.metaKey || e.ctrlKey) && e.key === "s") {
        e.preventDefault();
        handleSave();
      }
    },
    [onClose, handleSave]
  );

  const insertSample = useCallback(() => {
    setMarkdown(SAMPLE_MARKDOWN);
  }, []);

  if (!isOpen) return null;

  return (
    <div
      className="fixed inset-0 z-[9999] flex items-center justify-center bg-black/50"
      onClick={onClose}
      onKeyDown={handleKeyDown}
    >
      <div
        className="bg-white dark:bg-gray-800 rounded-xl shadow-2xl w-[90vw] max-w-5xl h-[85vh] flex flex-col"
        onClick={(e) => e.stopPropagation()}
      >
        {/* Header */}
        <div className="flex items-center justify-between px-6 py-4 border-b dark:border-gray-700">
          <h2 className="text-xl font-semibold text-gray-900 dark:text-white">
            Edit Markdown Card
          </h2>
          <div className="flex items-center gap-4">
            <button
              onClick={insertSample}
              className="text-sm text-blue-500 hover:text-blue-600 underline"
            >
              Insert Sample
            </button>
            <div className="flex rounded-lg overflow-hidden border dark:border-gray-600">
              <button
                onClick={() => setActiveTab("edit")}
                className={`px-4 py-1.5 text-sm font-medium transition-colors ${
                  activeTab === "edit"
                    ? "bg-blue-500 text-white"
                    : "bg-gray-100 dark:bg-gray-700 text-gray-700 dark:text-gray-300"
                }`}
              >
                Edit
              </button>
              <button
                onClick={() => setActiveTab("preview")}
                className={`px-4 py-1.5 text-sm font-medium transition-colors ${
                  activeTab === "preview"
                    ? "bg-blue-500 text-white"
                    : "bg-gray-100 dark:bg-gray-700 text-gray-700 dark:text-gray-300"
                }`}
              >
                Preview
              </button>
            </div>
            <button
              onClick={onClose}
              className="text-gray-400 hover:text-gray-600 dark:hover:text-gray-200 text-2xl leading-none"
            >
              &times;
            </button>
          </div>
        </div>

        {/* Content */}
        <div className="flex-1 overflow-hidden p-4">
          {activeTab === "edit" ? (
            <textarea
              ref={textareaRef}
              value={markdown}
              onChange={(e) => setMarkdown(e.target.value)}
              placeholder="Enter your markdown here..."
              className="w-full h-full p-4 font-mono text-sm resize-none rounded-lg border dark:border-gray-600 bg-gray-50 dark:bg-gray-900 text-gray-900 dark:text-gray-100 focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
          ) : (
            <div className="w-full h-full overflow-auto p-4 rounded-lg border dark:border-gray-600 bg-white dark:bg-gray-900">
              <div className="prose dark:prose-invert max-w-none">
                <ReactMarkdown
                  remarkPlugins={[remarkGfm]}
                  components={{
                    code({ className, children }) {
                      const match = /language-(\w+)/.exec(className || "");
                      const lang = match ? match[1] : "";
                      const code = String(children).replace(/\n$/, "");

                      if (lang === "mermaid") {
                        return <MermaidRenderer code={code} />;
                      }

                      if (!className) {
                        return (
                          <code className="px-1 py-0.5 bg-gray-100 dark:bg-gray-800 rounded text-sm">
                            {children}
                          </code>
                        );
                      }

                      return (
                        <pre className="p-3 bg-gray-100 dark:bg-gray-800 rounded overflow-auto">
                          <code className={className}>{children}</code>
                        </pre>
                      );
                    },
                  }}
                >
                  {markdown || "*Nothing to preview*"}
                </ReactMarkdown>
              </div>
            </div>
          )}
        </div>

        {/* Size limit error banner */}
        {isOverLimit && (
          <div className="px-6 py-3 bg-red-50 dark:bg-red-900/20 border-t border-red-200 dark:border-red-800">
            <div className="flex items-start gap-3">
              <svg
                className="w-5 h-5 text-red-600 dark:text-red-400 flex-shrink-0 mt-0.5"
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"
                />
              </svg>
              <div className="flex-1">
                <p className="text-sm font-medium text-red-800 dark:text-red-200">
                  Content exceeds maximum size limit
                </p>
                <p className="text-sm text-red-600 dark:text-red-400 mt-1">
                  Your content is {formatBytes(contentSize)} but the maximum
                  allowed is {formatBytes(MAX_MARKDOWN_CARD_SIZE)}. Please
                  reduce by {formatBytes(contentSize - MAX_MARKDOWN_CARD_SIZE)}.
                </p>
                <p className="text-xs text-red-500 dark:text-red-500 mt-2">
                  💡 Tip: Try removing images, shortening text, or splitting
                  content into multiple cards.
                </p>
              </div>
            </div>
          </div>
        )}

        {/* Size warning banner */}
        {isNearLimit && !isOverLimit && (
          <div className="px-6 py-2 bg-amber-50 dark:bg-amber-900/20 border-t border-amber-200 dark:border-amber-800">
            <p className="text-xs text-amber-700 dark:text-amber-300 flex items-center gap-2">
              <svg
                className="w-4 h-4"
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
                />
              </svg>
              Approaching size limit: {formatBytes(contentSize)} /{" "}
              {formatBytes(MAX_MARKDOWN_CARD_SIZE)} (
              {Math.round(sizePercentage)}%)
            </p>
          </div>
        )}

        {/* Footer */}
        <div className="flex items-center justify-between px-6 py-4 border-t dark:border-gray-700">
          <div className="flex items-center gap-4">
            <p className="text-sm text-gray-500">
              <kbd className="px-1.5 py-0.5 bg-gray-100 dark:bg-gray-700 rounded text-xs">
                ⌘S
              </kbd>{" "}
              to save,{" "}
              <kbd className="px-1.5 py-0.5 bg-gray-100 dark:bg-gray-700 rounded text-xs">
                Esc
              </kbd>{" "}
              to cancel
            </p>
            {/* Size indicator */}
            <div
              className={`text-xs flex items-center gap-1.5 ${
                isOverLimit
                  ? "text-red-600 dark:text-red-400"
                  : isNearLimit
                  ? "text-amber-600 dark:text-amber-400"
                  : "text-gray-500 dark:text-gray-400"
              }`}
              title={`${formatBytes(contentSize)} / ${formatBytes(
                MAX_MARKDOWN_CARD_SIZE
              )}`}
            >
              <svg
                className="w-3.5 h-3.5"
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M4 7v10c0 2.21 3.582 4 8 4s8-1.79 8-4V7M4 7c0 2.21 3.582 4 8 4s8-1.79 8-4M4 7c0-2.21 3.582-4 8-4s8 1.79 8 4m0 5c0 2.21-3.582 4-8 4s-8-1.79-8-4"
                />
              </svg>
              {formatBytes(contentSize)} / {formatBytes(MAX_MARKDOWN_CARD_SIZE)}
            </div>
          </div>
          <div className="flex gap-3">
            <button
              onClick={onClose}
              className="px-4 py-2 text-sm font-medium text-gray-700 dark:text-gray-300 bg-gray-100 dark:bg-gray-700 rounded-lg hover:bg-gray-200 dark:hover:bg-gray-600 transition-colors"
            >
              Cancel
            </button>
            <button
              onClick={handleSave}
              disabled={isOverLimit}
              className={`px-4 py-2 text-sm font-medium text-white rounded-lg transition-colors ${
                isOverLimit
                  ? "bg-gray-400 cursor-not-allowed"
                  : "bg-blue-500 hover:bg-blue-600"
              }`}
            >
              Save Card
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}
