"use client";

import { useEffect } from "react";
import { useRouter } from "next/navigation";

export default function PostgresusVsWalGRedirect() {
  const router = useRouter();

  useEffect(() => {
    router.replace("/databasus-vs-wal-g");
  }, [router]);

  return (
    <div
      style={{
        display: "flex",
        alignItems: "center",
        justifyContent: "center",
        minHeight: "100vh",
        fontFamily: "system-ui, sans-serif",
      }}
    >
      <p>Redirecting...</p>
    </div>
  );
}
