"use client";

/**
 * RichTextCard - Read-only Tiptap viewer for displaying rich text cards on the Excalidraw canvas.
 *
 * This component renders Tiptap JSON content in a styled container that appears as an
 * embeddable element on the canvas. It supports:
 * - All Tiptap formatting (bold, italic, lists, tables, code, etc.)
 * - Dark/light theme awareness from Excalidraw
 * - Search highlighting with indigo ring when content matches search
 * - Double-click to open the RichTextCardEditor modal
 *
 * Search highlighting works by checking if the associated search text element
 * (linked via searchTextElementId) is in the current searchMatches array.
 */

import React, { useCallback, useMemo, useEffect } from "react";
import { useEditor, EditorContent } from "@tiptap/react";
import StarterKit from "@tiptap/starter-kit";
import Link from "@tiptap/extension-link";
import Highlight from "@tiptap/extension-highlight";
import TaskList from "@tiptap/extension-task-list";
import TaskItem from "@tiptap/extension-task-item";
import TextAlign from "@tiptap/extension-text-align";
import Underline from "@tiptap/extension-underline";
import { TextStyle } from "@tiptap/extension-text-style";
import Color from "@tiptap/extension-color";
import { Table } from "@tiptap/extension-table";
import { TableRow } from "@tiptap/extension-table-row";
import { TableCell } from "@tiptap/extension-table-cell";
import { TableHeader } from "@tiptap/extension-table-header";

interface RichTextCardProps {
  element: {
    id: string;
    customData?: {
      richTextContent?: string; // JSON string of Tiptap content
      searchTextElementId?: string;
    };
    width: number;
    height: number;
  };
  appState: {
    theme?: string;
    searchMatches?: readonly { id: string; focus: boolean }[] | null;
  };
  onEdit?: (elementId: string, content: string) => void;
}

// Default content for new rich text cards
export const DEFAULT_RICH_TEXT_CONTENT = {
  type: "doc",
  content: [
    {
      type: "heading",
      attrs: { level: 2 },
      content: [{ type: "text", text: "Rich Text Card" }],
    },
    {
      type: "paragraph",
      content: [
        {
          type: "text",
          text: "Double-click to edit this card. This editor supports:",
        },
      ],
    },
    {
      type: "bulletList",
      content: [
        {
          type: "listItem",
          content: [
            {
              type: "paragraph",
              content: [
                { type: "text", marks: [{ type: "bold" }], text: "Bold" },
                { type: "text", text: ", " },
                { type: "text", marks: [{ type: "italic" }], text: "italic" },
                { type: "text", text: ", and " },
                {
                  type: "text",
                  marks: [{ type: "underline" }],
                  text: "underline",
                },
                { type: "text", text: " text" },
              ],
            },
          ],
        },
        {
          type: "listItem",
          content: [
            {
              type: "paragraph",
              content: [{ type: "text", text: "Headings (H1, H2, H3)" }],
            },
          ],
        },
        {
          type: "listItem",
          content: [
            {
              type: "paragraph",
              content: [{ type: "text", text: "Bullet and numbered lists" }],
            },
          ],
        },
        {
          type: "listItem",
          content: [
            {
              type: "paragraph",
              content: [{ type: "text", text: "Task lists with checkboxes" }],
            },
          ],
        },
        {
          type: "listItem",
          content: [
            {
              type: "paragraph",
              content: [
                { type: "text", text: "Code blocks and " },
                {
                  type: "text",
                  marks: [{ type: "code" }],
                  text: "inline code",
                },
              ],
            },
          ],
        },
        {
          type: "listItem",
          content: [
            {
              type: "paragraph",
              content: [{ type: "text", text: "Links and blockquotes" }],
            },
          ],
        },
        {
          type: "listItem",
          content: [
            {
              type: "paragraph",
              content: [
                {
                  type: "text",
                  marks: [{ type: "highlight" }],
                  text: "Highlighted text",
                },
              ],
            },
          ],
        },
      ],
    },
    {
      type: "taskList",
      content: [
        {
          type: "taskItem",
          attrs: { checked: true },
          content: [
            {
              type: "paragraph",
              content: [{ type: "text", text: "Create a rich text card" }],
            },
          ],
        },
        {
          type: "taskItem",
          attrs: { checked: false },
          content: [
            {
              type: "paragraph",
              content: [{ type: "text", text: "Add your content" }],
            },
          ],
        },
      ],
    },
    {
      type: "blockquote",
      content: [
        {
          type: "paragraph",
          content: [
            { type: "text", text: "💡 " },
            {
              type: "text",
              marks: [{ type: "italic" }],
              text: "Pro tip: Use keyboard shortcuts like Ctrl+B for bold, Ctrl+I for italic!",
            },
          ],
        },
      ],
    },
  ],
};

export default function RichTextCard({
  element,
  appState,
  onEdit,
}: RichTextCardProps) {
  const isDark = appState.theme === "dark";

  // Parse content from customData
  const customData = element.customData;
  const rawContent = customData?.richTextContent;

  const content = useMemo(() => {
    if (rawContent) {
      try {
        return JSON.parse(rawContent);
      } catch {
        return DEFAULT_RICH_TEXT_CONTENT;
      }
    }
    return DEFAULT_RICH_TEXT_CONTENT;
  }, [rawContent]);

  // Create a stable key for content changes to help with re-renders
  const contentKey = useMemo(() => {
    return rawContent ? rawContent.substring(0, 100) : "default";
  }, [rawContent]);

  // Check if this card or its search text element is highlighted by search
  // First try customData, then derive from element ID for backwards compatibility
  const searchTextElementId =
    customData?.searchTextElementId ||
    (element.id.startsWith("rt-") ? `rtsearch-${element.id}` : undefined);
  const isSearchHighlighted = Boolean(
    searchTextElementId &&
      appState.searchMatches?.some((match) => match.id === searchTextElementId)
  );

  // Configure read-only editor for display
  const editor = useEditor({
    extensions: [
      StarterKit.configure({
        heading: { levels: [1, 2, 3] },
      }),
      Link.configure({
        openOnClick: false,
        HTMLAttributes: {
          class:
            "text-indigo-600 dark:text-indigo-400 underline cursor-pointer",
        },
      }),
      Highlight.configure({
        HTMLAttributes: {
          class: "bg-yellow-200 dark:bg-yellow-900/50 px-1 rounded",
        },
      }),
      TaskList,
      TaskItem.configure({
        nested: true,
      }),
      TextAlign.configure({
        types: ["heading", "paragraph"],
      }),
      Underline,
      TextStyle,
      Color,
      Table.configure({
        resizable: false,
      }),
      TableRow,
      TableCell,
      TableHeader,
    ],
    content,
    editable: false,
    immediatelyRender: false, // SSR-safe
    editorProps: {
      attributes: {
        class: "prose prose-sm dark:prose-invert max-w-none",
      },
    },
  });

  // Update editor content when the element's content changes
  useEffect(() => {
    if (editor && content) {
      // Only update if the content actually differs
      const currentContent = editor.getJSON();
      if (JSON.stringify(currentContent) !== JSON.stringify(content)) {
        editor.commands.setContent(content);
      }
    }
  }, [editor, content]);

  const handleDoubleClick = useCallback(() => {
    if (onEdit) {
      onEdit(element.id, JSON.stringify(content));
    }
  }, [element.id, content, onEdit]);

  return (
    <div
      onDoubleClick={handleDoubleClick}
      className={`
        w-full h-full overflow-auto rounded-lg shadow-sm
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
        rich-text-card
      `}
      style={{
        fontFamily: "system-ui, -apple-system, sans-serif",
      }}
    >
      <EditorContent
        key={contentKey}
        editor={editor}
        className="rich-text-content p-4"
      />
    </div>
  );
}
