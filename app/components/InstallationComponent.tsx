"use client";

import { useState } from "react";

type InstallMethod =
  | "Automated Script"
  | "Docker Run"
  | "Docker Compose"
  | "Helm";

type ScriptVariant = {
  label: string;
  code: string;
};

type CodeBlock = {
  label: string;
  code: string;
};

type Installation = {
  label: string;
  code: string | ScriptVariant[];
  codeBlocks?: CodeBlock[];
  language: string;
  description: string;
};

const installationMethods: Record<InstallMethod, Installation> = {
  "Automated Script": {
    label: "Automated script (recommended)",
    language: "bash",
    description:
      "The installation script will install Docker with Docker Compose (if not already installed), set up Postgresus and configure automatic startup on system reboot.",
    code: [
      {
        label: "with sudo",
        code: `sudo apt-get install -y curl && \\
sudo curl -sSL https://raw.githubusercontent.com/RostislavDugin/postgresus/refs/heads/main/install-postgresus.sh | sudo bash`,
      },
      {
        label: "without sudo",
        code: `apt-get install -y curl && \\
curl -sSL https://raw.githubusercontent.com/RostislavDugin/postgresus/refs/heads/main/install-postgresus.sh | bash`,
      },
    ],
  },
  "Docker Run": {
    label: "Docker",
    language: "bash",
    description:
      "The easiest way to run Postgresus. This single command will start Postgresus, store all data in ./postgresus-data directory and automatically restart on system reboot.",
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
  Helm: {
    label: "Helm (Kubernetes)",
    language: "bash",
    description:
      "For Kubernetes deployments, clone the repository and use the official Helm chart. This will create a StatefulSet with persistent storage and LoadBalancer service on port 80. Config uses by default LoadBalancer, but has predefined values for Ingress and HTTPRoute as well.",
    code: "",
    codeBlocks: [
      {
        label: "Clone the repository",
        code: `git clone https://github.com/RostislavDugin/postgresus.git
cd postgresus`,
      },
      {
        label: "Install the Helm chart",
        code: `helm install postgresus ./deploy/helm -n postgresus --create-namespace`,
      },
      {
        label: "Get the external IP",
        code: `kubectl get svc -n postgresus`,
      },
    ],
  },
};

const methods: InstallMethod[] = [
  "Automated Script",
  "Docker Run",
  "Docker Compose",
  "Helm",
];

export default function InstallationComponent() {
  const [selectedMethod, setSelectedMethod] =
    useState<InstallMethod>("Automated Script");
  const [selectedVariant, setSelectedVariant] = useState(0);
  const [copied, setCopied] = useState(false);
  const [copiedBlockIndex, setCopiedBlockIndex] = useState<number | null>(null);

  const currentInstallation = installationMethods[selectedMethod];
  const hasVariants =
    Array.isArray(currentInstallation.code) &&
    currentInstallation.code.length > 0;
  const hasCodeBlocks =
    currentInstallation.codeBlocks && currentInstallation.codeBlocks.length > 0;

  const handleMethodChange = (method: InstallMethod) => {
    setSelectedMethod(method);
    setSelectedVariant(0);
    setCopied(false);
    setCopiedBlockIndex(null);
  };

  const handleVariantChange = (index: number) => {
    setSelectedVariant(index);
    setCopied(false);
  };

  const getCurrentCode = () => {
    if (hasVariants) {
      return (currentInstallation.code as ScriptVariant[])[selectedVariant]
        .code;
    }
    return currentInstallation.code as string;
  };

  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(getCurrentCode());
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch (err) {
      console.error("Failed to copy:", err);
    }
  };

  const handleCopyBlock = async (code: string, index: number) => {
    try {
      await navigator.clipboard.writeText(code);
      setCopiedBlockIndex(index);
      setTimeout(() => setCopiedBlockIndex(null), 2000);
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

      {/* Script variants tabs (only for Automated Script) */}
      {hasVariants && (
        <div className="mb-4 flex flex-wrap gap-2">
          {(currentInstallation.code as ScriptVariant[]).map(
            (variant, index) => (
              <button
                key={index}
                onClick={() => handleVariantChange(index)}
                className={`cursor-pointer rounded-lg px-4 py-2 text-sm font-medium transition-colors ${
                  selectedVariant === index
                    ? "bg-blue-100 text-blue-700 ring-2 ring-blue-600"
                    : "bg-gray-50 text-gray-600 hover:bg-gray-100"
                }`}
              >
                {variant.label}
              </button>
            )
          )}
        </div>
      )}

      {/* Description */}
      <div className="mb-4 text-base md:text-lg max-w-[600px]">
        {currentInstallation.description}
      </div>

      {/* Multiple code blocks (for Helm) */}
      {hasCodeBlocks ? (
        <div className="space-y-4 max-w-[600px]">
          {currentInstallation.codeBlocks!.map((block, index) => (
            <div key={index}>
              <p className="mb-2 text-sm font-medium text-gray-700">
                {block.label}
              </p>
              <div className="relative">
                <pre className="rounded-lg bg-gray-100 p-4 pr-16 text-sm">
                  <code className="block whitespace-pre-wrap wrap-break-word">
                    {block.code}
                  </code>
                </pre>
                <button
                  onClick={() => handleCopyBlock(block.code, index)}
                  className={`absolute right-2 top-2 rounded px-2 py-1 text-xs text-white transition-colors ${
                    copiedBlockIndex === index
                      ? "bg-green-500"
                      : "bg-blue-600 hover:bg-blue-700"
                  }`}
                >
                  {copiedBlockIndex === index ? "Copied!" : "Copy"}
                </button>
              </div>
            </div>
          ))}
        </div>
      ) : (
        /* Single code block with copy button */
        <div className="relative max-w-[600px]">
          <pre className="rounded-lg bg-gray-100 p-4 pr-16 text-sm">
            <code className="block whitespace-pre-wrap wrap-break-word">
              {getCurrentCode()}
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
      )}
    </div>
  );
}
