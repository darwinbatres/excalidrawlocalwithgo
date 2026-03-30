import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  // Enable strict mode
  reactStrictMode: true,

  // Standalone output for Docker deployment
  output: "standalone",

  // Optimize images — no remote patterns needed (all images served from our own API)
  images: {
    remotePatterns: [],
  },

  // Security headers (industry standard)
  headers: async () => [
    {
      source: "/(.*)",
      headers: [
        {
          key: "X-DNS-Prefetch-Control",
          value: "on",
        },
        {
          key: "X-Frame-Options",
          value: "SAMEORIGIN",
        },
        {
          key: "X-Content-Type-Options",
          value: "nosniff",
        },
        {
          key: "Referrer-Policy",
          value: "strict-origin-when-cross-origin",
        },
        {
          key: "Permissions-Policy",
          value: "camera=(), microphone=(), geolocation=()",
        },
      ],
    },
  ],

  // Environment variables exposed to the browser
  env: {
    NEXT_PUBLIC_APP_URL:
      process.env.NEXT_PUBLIC_APP_URL || "http://localhost:3021",
  },

  // Proxy API requests to Go backend (local dev only — in Docker, Caddy handles this)
  rewrites: async () => [
    {
      source: "/api/v1/:path*",
      destination: `${process.env.NEXT_PUBLIC_API_URL || "http://localhost:8090"}/api/v1/:path*`,
    },
  ],

  // Turbopack config (Next.js 16 default bundler)
  turbopack: {},

  // Webpack fallback config for Excalidraw compatibility
  webpack: (config) => {
    config.resolve.fallback = {
      ...config.resolve.fallback,
      fs: false,
      path: false,
    };
    return config;
  },
};

export default nextConfig;
