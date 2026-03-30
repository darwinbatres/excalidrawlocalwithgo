"use client";

import React, { useCallback } from "react";
import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";
import { MermaidRenderer } from "./MermaidRenderer";

interface MarkdownCardProps {
  element: {
    id: string;
    customData?: {
      markdown?: string;
      searchTextElementId?: string;
    };
    width: number;
    height: number;
  };
  appState: {
    theme?: string;
    searchMatches?: readonly { id: string; focus: boolean }[] | null;
  };
  onEdit?: (elementId: string, markdown: string) => void;
}

export default function MarkdownCard({
  element,
  appState,
  onEdit,
}: MarkdownCardProps) {
  const markdown =
    element.customData?.markdown || "*Double-click to add content*";
  const isDark = appState.theme === "dark";

  // Check if this card's search text element is highlighted by search
  // First try customData, then derive from element ID for backwards compatibility
  const searchTextElementId =
    element.customData?.searchTextElementId ||
    (element.id.startsWith("md-") ? `mdsearch-${element.id}` : undefined);
  const isSearchHighlighted = Boolean(
    searchTextElementId &&
      appState.searchMatches?.some((match) => match.id === searchTextElementId)
  );

  const handleDoubleClick = useCallback(() => {
    if (onEdit) {
      onEdit(element.id, markdown);
    }
  }, [element.id, markdown, onEdit]);

  return (
    <div
      onDoubleClick={handleDoubleClick}
      className={`
        w-full h-full overflow-auto p-4 rounded-lg shadow-sm
        ${isDark ? "bg-gray-800 text-gray-100" : "bg-white text-gray-900"}
        ${
          isSearchHighlighted
            ? "ring-4 ring-indigo-500 ring-offset-2 border-indigo-500"
            : isDark
            ? "border-gray-700"
            : "border-gray-200"
        }
        border-2 cursor-pointer select-none
        transition-all duration-200
      `}
      style={{
        fontFamily: "system-ui, -apple-system, sans-serif",
        fontSize: "14px",
        lineHeight: "1.6",
      }}
    >
      <ReactMarkdown
        remarkPlugins={[remarkGfm]}
        components={{
          // Render mermaid code blocks
          code({ className, children, ...props }) {
            const match = /language-(\w+)/.exec(className || "");
            const language = match ? match[1] : "";
            const codeContent = String(children).replace(/\n$/, "");

            if (language === "mermaid") {
              return <MermaidRenderer code={codeContent} />;
            }

            // Inline code
            if (!className) {
              return (
                <code
                  className={`px-1 py-0.5 rounded text-sm ${
                    isDark ? "bg-gray-700" : "bg-gray-100"
                  }`}
                  {...props}
                >
                  {children}
                </code>
              );
            }

            // Code block
            return (
              <pre
                className={`p-3 rounded overflow-auto text-sm ${
                  isDark ? "bg-gray-900" : "bg-gray-50"
                }`}
              >
                <code className={className} {...props}>
                  {children}
                </code>
              </pre>
            );
          },
          // Style headings
          h1: ({ children }) => (
            <h1 className="text-2xl font-bold mb-4 pb-2 border-b border-current/20">
              {children}
            </h1>
          ),
          h2: ({ children }) => (
            <h2 className="text-xl font-semibold mb-3 mt-4">{children}</h2>
          ),
          h3: ({ children }) => (
            <h3 className="text-lg font-medium mb-2 mt-3">{children}</h3>
          ),
          // Style paragraphs
          p: ({ children }) => <p className="mb-3">{children}</p>,
          // Style lists
          ul: ({ children }) => (
            <ul className="list-disc list-inside mb-3 space-y-1">{children}</ul>
          ),
          ol: ({ children }) => (
            <ol className="list-decimal list-inside mb-3 space-y-1">
              {children}
            </ol>
          ),
          // Style blockquotes
          blockquote: ({ children }) => (
            <blockquote
              className={`border-l-4 pl-4 italic my-3 ${
                isDark
                  ? "border-gray-600 text-gray-300"
                  : "border-gray-300 text-gray-600"
              }`}
            >
              {children}
            </blockquote>
          ),
          // Style tables
          table: ({ children }) => (
            <div className="overflow-auto mb-3">
              <table
                className={`min-w-full border-collapse ${
                  isDark ? "border-gray-600" : "border-gray-300"
                }`}
              >
                {children}
              </table>
            </div>
          ),
          th: ({ children }) => (
            <th
              className={`border px-3 py-2 text-left font-semibold ${
                isDark
                  ? "border-gray-600 bg-gray-700"
                  : "border-gray-300 bg-gray-100"
              }`}
            >
              {children}
            </th>
          ),
          td: ({ children }) => (
            <td
              className={`border px-3 py-2 ${
                isDark ? "border-gray-600" : "border-gray-300"
              }`}
            >
              {children}
            </td>
          ),
          // Style links
          a: ({ href, children }) => (
            <a
              href={href}
              target="_blank"
              rel="noopener noreferrer"
              className="text-blue-500 hover:underline"
            >
              {children}
            </a>
          ),
          // Style horizontal rules
          hr: () => (
            <hr
              className={`my-4 ${
                isDark ? "border-gray-600" : "border-gray-300"
              }`}
            />
          ),
          // Style checkboxes (GFM task lists)
          input: ({ type, checked }) => {
            if (type === "checkbox") {
              return (
                <input
                  type="checkbox"
                  checked={checked}
                  readOnly
                  className="mr-2 pointer-events-none"
                />
              );
            }
            return <input type={type} />;
          },
        }}
      >
        {markdown}
      </ReactMarkdown>
    </div>
  );
}

// Export type for external use
export type { MarkdownCardProps };
