"use client";

import { useEffect } from "react";
import { useRouter } from "next/navigation";

export default function PostgresusVsPgbackrestRedirect() {
  const router = useRouter();

  useEffect(() => {
    router.replace("/databasus-vs-pgbackrest");
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
