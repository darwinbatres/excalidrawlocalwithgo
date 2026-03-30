import React from "react";
import { Header } from "./Header";

interface LayoutProps {
  children: React.ReactNode;
  fullScreen?: boolean;
}

export function Layout({ children, fullScreen = false }: LayoutProps) {
  if (fullScreen) {
    return <div className="h-screen flex flex-col">{children}</div>;
  }

  return (
    <div className="min-h-screen bg-gray-50 dark:bg-gray-950">
      <Header />
      <main>{children}</main>
    </div>
  );
}
