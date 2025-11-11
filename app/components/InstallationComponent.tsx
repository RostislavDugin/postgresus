"use client";

import { useState } from "react";

type InstallMethod = "Automated Script" | "Docker Run" | "Docker Compose";

type Installation = {
  label: string;
  code: string;
  language: string;
  description: string;
};

const installationMethods: Record<InstallMethod, Installation> = {
  "Automated Script": {
    label: "Automated script (recommended)",
    language: "bash",
    description:
      "The installation script will install Docker with Docker Compose (if not already installed), set up Postgresus, and configure automatic startup on system reboot.",
    code: `sudo apt-get install -y curl && \\
sudo curl -sSL https://raw.githubusercontent.com/RostislavDugin/postgresus/refs/heads/main/install-postgresus.sh | sudo bash`,
  },
  "Docker Run": {
    label: "Docker",
    language: "bash",
    description:
      "The easiest way to run Postgresus. This single command will start Postgresus, store all data in ./postgresus-data directory, and automatically restart on system reboot.",
    code: `docker run -d \\
  --name postgresus \\
  -p 4005:4005 \\
  -v ./postgresus-data:/postgresus-data \\
  --restart unless-stopped \\
  rostislavdugin/postgresus:latest`,
  },
  "Docker Compose": {
    label: "Docker Compose",
    language: "yaml",
    description:
      "Create a docker-compose.yml file with the following configuration, then run: docker compose up -d",
    code: `services:
  postgresus:
    container_name: postgresus
    image: rostislavdugin/postgresus:latest
    ports:
      - "4005:4005"
    volumes:
      - ./postgresus-data:/postgresus-data
    restart: unless-stopped`,
  },
};

const methods: InstallMethod[] = [
  "Automated Script",
  "Docker Run",
  "Docker Compose",
];

export default function InstallationComponent() {
  const [selectedMethod, setSelectedMethod] =
    useState<InstallMethod>("Automated Script");
  const [copied, setCopied] = useState(false);

  const currentInstallation = installationMethods[selectedMethod];

  const handleMethodChange = (method: InstallMethod) => {
    setSelectedMethod(method);
    setCopied(false);
  };

  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(currentInstallation.code);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch (err) {
      console.error("Failed to copy:", err);
    }
  };

  return (
    <div className="mx-auto w-full">
      {/* Installation methods tabs */}
      <div className="mb-6 flex flex-wrap gap-2">
        {methods.map((method) => (
          <button
            key={method}
            onClick={() => handleMethodChange(method)}
            className={`cursor-pointer rounded-lg px-4 py-2 text-sm font-medium transition-colors md:px-6 md:py-2 md:text-base ${
              selectedMethod === method
                ? "bg-blue-600 text-white"
                : "bg-gray-100 text-gray-700 hover:bg-gray-200"
            }`}
          >
            {installationMethods[method].label}
          </button>
        ))}
      </div>

      {/* Description */}
      <div className="mb-4 text-base md:text-lg max-w-[600px]">
        {currentInstallation.description}
      </div>

      {/* Code block with copy button */}
      <div className="relative max-w-[600px]">
        <pre className="rounded-lg bg-gray-100 p-4 pr-16 text-sm">
          <code className="block whitespace-pre-wrap wrap-break-word">
            {currentInstallation.code}
          </code>
        </pre>

        <button
          onClick={handleCopy}
          className={`absolute right-2 top-2 rounded px-2 py-1 text-xs text-white transition-colors ${
            copied ? "bg-green-500" : "bg-blue-600 hover:bg-blue-700"
          }`}
        >
          {copied ? "Copied!" : "Copy"}
        </button>
      </div>
    </div>
  );
}
