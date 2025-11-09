import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  async redirects() {
    return [
      // Old storage URLs to new structure
      {
        source: "/cloudflare-r2-storage",
        destination: "/storages/cloudflare-r2",
        permanent: true,
      },
      {
        source: "/google-drive-storage",
        destination: "/storages/google-drive",
        permanent: true,
      },
      // Old notifier URLs to new structure
      {
        source: "/notifier-slack",
        destination: "/notifiers/slack",
        permanent: true,
      },
      {
        source: "/notifier-teams",
        destination: "/notifiers/teams",
        permanent: true,
      },
      // Blog removed permanently
      {
        source: "/blog",
        destination: "/gone",
        permanent: true,
      },
    ];
  },
};

export default nextConfig;
