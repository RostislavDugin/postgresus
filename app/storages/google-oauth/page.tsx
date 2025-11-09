"use client";

import { useEffect } from "react";
import type { Metadata } from "next";

export const metadata: Metadata = {
  robots: "noindex, nofollow",
};

export default function GoogleOAuthPage() {
  useEffect(() => {
    // Get current URL parameters
    const urlParams = new URLSearchParams(window.location.search);

    // Extract state and code from current URL
    const state = urlParams.get("state");
    const code = urlParams.get("code");

    if (state && code) {
      try {
        // Parse the StorageOauthDto from the state parameter
        const oauthDto = JSON.parse(decodeURIComponent(state));

        // Update the authCode field with the received code
        oauthDto.authCode = code;

        // Construct the redirect URL with the updated DTO
        const redirectUrl = `${
          oauthDto.redirectUrl
        }?oauthDto=${encodeURIComponent(JSON.stringify(oauthDto))}`;

        // Redirect to the constructed URL
        window.location.href = redirectUrl;
      } catch (error) {
        console.error("Error parsing state parameter:", error);
      }
    } else {
      console.error("Missing state or code parameter");
    }
  }, []);

  return (
    <div
      style={{
        textAlign: "center",
        padding: "50px",
        fontFamily: "Arial, sans-serif",
      }}
    >
      <h2>Processing OAuth...</h2>
      <p>Please wait while we redirect you back to the application.</p>
    </div>
  );
}
