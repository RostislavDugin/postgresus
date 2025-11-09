import type { Metadata } from "next";
import OAuthProcessor from "./OAuthProcessor";

export const metadata: Metadata = {
  robots: "noindex, nofollow",
};

export default function GoogleOAuthPage() {
  return <OAuthProcessor />;
}
