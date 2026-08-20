import type { NextConfig } from "next";

// /api/* хүсэлтүүд Go API руу дамжина — нэг origin, CORS хэрэггүй.
const API_URL = process.env.API_URL || "http://127.0.0.1:8084";

const nextConfig: NextConfig = {
  async rewrites() {
    return [{ source: "/api/:path*", destination: `${API_URL}/api/:path*` }];
  },
};

export default nextConfig;
