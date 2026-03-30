"use client";

import React, { useEffect, useState } from "react";
import mermaid from "mermaid";
import DOMPurify from "dompurify";

let mermaidInitialized = false;

export function initMermaid() {
  if (mermaidInitialized || typeof window === "undefined") return;
  mermaid.initialize({
    startOnLoad: false,
    theme: "neutral",
    securityLevel: "strict",
    fontFamily: "inherit",
  });
  mermaidInitialized = true;
}

interface MermaidRendererProps {
  code: string;
  className?: string;
}

export function MermaidRenderer({ code, className }: MermaidRendererProps) {
  const [svg, setSvg] = useState<string>("");
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    initMermaid();

    const renderDiagram = async () => {
      if (!code.trim()) return;

      try {
        const id = `mermaid-${Math.random().toString(36).slice(2, 11)}`;
        const { svg: renderedSvg } = await mermaid.render(id, code);
        setSvg(renderedSvg);
        setError(null);
      } catch (err) {
        setError(err instanceof Error ? err.message : "Mermaid render error");
        setSvg("");
      }
    };

    renderDiagram();
  }, [code]);

  if (error) {
    return (
      <div className="bg-red-50 border border-red-200 rounded p-2 text-red-600 text-sm">
        <strong>Diagram Error:</strong> {error}
      </div>
    );
  }

  if (!svg) {
    return <div className="text-gray-400 text-sm">Loading diagram...</div>;
  }

  return (
    <div
      className={className || "mermaid-container overflow-auto"}
      dangerouslySetInnerHTML={{
        __html: DOMPurify.sanitize(svg, { USE_PROFILES: { svg: true } }),
      }}
    />
  );
}
