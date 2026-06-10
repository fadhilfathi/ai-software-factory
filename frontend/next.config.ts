import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  /* API proxy — forward /api/* to Go backend during dev */
  async rewrites() {
    return [
      {
        source: "/api/:path*",
        destination: `${process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080/v1"}/:path*`,
      },
    ];
  },

  /* Standalone output for Docker deployment */
  output: "standalone",

  /* Strict mode for React 19 concurrent features */
  reactStrictMode: true,
};

export default nextConfig;
