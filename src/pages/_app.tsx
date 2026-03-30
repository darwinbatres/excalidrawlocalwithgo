import "@/styles/globals.css";
import type { AppProps } from "next/app";
import { AuthProvider } from "@/contexts/AuthContext";
import { AppProvider } from "@/contexts/AppContext";
import { ErrorBoundary } from "@/components/ErrorBoundary";
import { Toaster } from "sonner";

export default function App({ Component, pageProps }: AppProps) {
  return (
    <AuthProvider>
      <ErrorBoundary>
        <AppProvider>
          <Component {...pageProps} />
          <Toaster
            position="bottom-right"
            toastOptions={{
              duration: 4000,
              className: "sonner-toast",
              style: {
                background: "#FAFAFA",
                border: "1px solid #E5E5E5",
                borderRadius: "12px",
                boxShadow: "0 4px 12px rgba(0, 0, 0, 0.08)",
                padding: "16px",
                color: "#171717",
              },
              descriptionClassName: "sonner-description",
            }}
          />
        </AppProvider>
      </ErrorBoundary>
    </AuthProvider>
  );
}
