"use client";

import { useState } from "react";

interface CopyButtonProps {
  text: string;
  className?: string;
}

export function CopyButton({ text, className = "" }: CopyButtonProps) {
  const [copied, setCopied] = useState(false);

  const copyToClipboard = () => {
    if (!navigator.clipboard) {
      const textArea = document.createElement("textarea");
      textArea.value = text;
      document.body.appendChild(textArea);
      textArea.select();
      document.execCommand("copy");
      document.body.removeChild(textArea);
      showCopyFeedback();
      return;
    }

    navigator.clipboard
      .writeText(text)
      .then(() => {
        showCopyFeedback();
      })
      .catch((err) => {
        console.error("Failed to copy text: ", err);
        const textArea = document.createElement("textarea");
        textArea.value = text;
        document.body.appendChild(textArea);
        textArea.select();
        document.execCommand("copy");
        document.body.removeChild(textArea);
        showCopyFeedback();
      });
  };

  const showCopyFeedback = () => {
    setCopied(true);
    setTimeout(() => {
      setCopied(false);
    }, 2000);
  };

  return (
    <button
      onClick={copyToClipboard}
      className={`rounded px-2 py-1 text-xs text-white transition-colors ${
        copied ? "bg-green-500" : "bg-blue-600 hover:bg-blue-700"
      } ${className}`}
    >
      {copied ? "Copied!" : "Copy"}
    </button>
  );
}
