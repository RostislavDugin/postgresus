"use client";

import Script from "next/script";
import { PADDLE_CLIENT_TOKEN, PADDLE_SANDBOX } from "../sponsorship/tiers";

// Minimal shape of the Paddle Billing v2 global we rely on.
interface PaddleApi {
  Environment?: { set: (env: "sandbox" | "production") => void };
  Initialize: (options: { token: string }) => void;
}

/**
 * Initialises Paddle Billing once paddle.js has loaded. Renders no UI of its own.
 *
 * After Initialize(), Paddle auto-binds any element with class="paddle_button" and a
 * data-items attribute, so the server-rendered tier buttons open the overlay without any
 * per-card React handler. The sponsorship page renders this only when PADDLE_ENABLED.
 */
export default function PaddleInitComponent() {
  return (
    <Script
      src="https://cdn.paddle.com/paddle/v2/paddle.js"
      strategy="afterInteractive"
      onLoad={() => {
        const paddle = (window as unknown as { Paddle?: PaddleApi }).Paddle;
        if (!paddle) return;
        if (PADDLE_SANDBOX) paddle.Environment?.set("sandbox");
        paddle.Initialize({ token: PADDLE_CLIENT_TOKEN });
      }}
    />
  );
}
