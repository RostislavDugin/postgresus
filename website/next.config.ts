import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  output: "export", // Enable static exports for GitHub Pages
  trailingSlash: true, // Ensures proper routing on GitHub Pages
  images: {
    unoptimized: true, // Required for static export
  },
};

export default nextConfig;
