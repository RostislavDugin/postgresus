"use client";

import { useState } from "react";
import LiteYouTubeEmbed from "./LiteYouTubeEmbed";

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
  title: string;
  code: string | ScriptVariant[];
  codeBlocks?: CodeBlock[];
  language: string;
  description: string;
};

const installationMethods: Record<InstallMethod, Installation> = {
  "Automated Script": {
    label: "Automated script",
    title: "Automated script (recommended)",
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
    title: "Docker",
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
    title: "Docker Compose",
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
    title: "Helm (Kubernetes)",
    language: "bash",
    description:
      "For Kubernetes deployments, install directly from the OCI registry. Choose your preferred access method: ClusterIP with port-forward for development, LoadBalancer for cloud environments, or Ingress for domain-based access.",
    code: "",
    codeBlocks: [
      {
        label: "With ClusterIP + port-forward (development)",
        code: `helm install postgresus oci://ghcr.io/rostislavdugin/charts/postgresus \\
  -n postgresus --create-namespace

kubectl port-forward svc/postgresus-service 4005:4005 -n postgresus
# Access at http://localhost:4005`,
      },
      {
        label: "With LoadBalancer (cloud environments)",
        code: `helm install postgresus oci://ghcr.io/rostislavdugin/charts/postgresus \\
  -n postgresus --create-namespace \\
  --set service.type=LoadBalancer

kubectl get svc postgresus-service -n postgresus
# Access at http://<EXTERNAL-IP>:4005`,
      },
      {
        label: "With Ingress (domain-based access)",
        code: `helm install postgresus oci://ghcr.io/rostislavdugin/charts/postgresus \\
  -n postgresus --create-namespace \\
  --set ingress.enabled=true \\
  --set ingress.hosts[0].host=backup.example.com`,
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
      <div className="mb-6 flex flex-wrap gap-2 justify-center">
        {methods.map((method) => (
          <button
            key={method}
            onClick={() => handleMethodChange(method)}
            className={`cursor-pointer rounded-lg px-3 py-1.5 text-sm font-medium transition-colors sm:px-4 sm:py-2 md:px-6 md:py-2 md:text-base ${
              selectedMethod === method
                ? "bg-blue-600 text-white border border-[#155dfc]"
                : "border border-[#ffffff20] hover:border-[#155dfc] hover:bg-blue-600 hover:text-white"
            }`}
          >
            {installationMethods[method].label}
          </button>
        ))}
      </div>

      <div className="flex flex-col lg:flex-row gap-8 lg:gap-10 mt-8 lg:mt-20">
        <div className="w-full lg:w-[50%]">
          <div className="text-xl md:text-2xl font-bold mb-3 md:mb-4">
            {currentInstallation.title}
          </div>

          {/* Description */}
          <div className="mb-4 md:mb-5 max-w-[550px] text-gray-400 text-sm md:text-base">
            {currentInstallation.description}
          </div>

          {/* Script variants tabs (only for Automated Script) */}
          {hasVariants && (
            <div className="mb-4 flex flex-wrap gap-2">
              {(currentInstallation.code as ScriptVariant[]).map(
                (variant, index) => (
                  <button
                    key={index}
                    onClick={() => handleVariantChange(index)}
                    className={`cursor-pointer rounded-lg px-3 py-1.5 text-sm sm:px-4 sm:py-2 font-medium transition-colors ${
                      selectedVariant === index
                        ? "bg-[#2C2F35] text-white border border-[#2C2F35]"
                        : "border border-[#ffffff20] hover:border-[#2C2F35] hover:bg-[#2C2F35] hover:text-white"
                    }`}
                  >
                    {variant.label}
                  </button>
                )
              )}
            </div>
          )}

          {/* Multiple code blocks (for Helm) */}
          {hasCodeBlocks ? (
            <div className="space-y-4">
              {currentInstallation.codeBlocks!.map((block, index) => (
                <div key={index}>
                  <p className="mb-2 text-gray-400 text-sm md:text-base">
                    {block.label}
                  </p>
                  <div className="relative">
                    <pre className="rounded-lg p-3 md:p-4 sm:pr-14 md:pr-16 text-sm border border-[#ffffff20] overflow-x-auto">
                      <code className="block whitespace-pre-wrap wrap-break-word">
                        {block.code}
                      </code>
                    </pre>
                    <button
                      onClick={() => handleCopyBlock(block.code, index)}
                      className={`absolute right-2 top-2 rounded px-2 py-1 text-sm text-white transition-colors ${
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
            <div className="relative border border-[#ffffff20] max-w-full lg:max-w-[530px] rounded-lg p-2 sm:pr-14 md:pr-16">
              <pre className="rounded-lg p-3 md:p-4 sm:pr-14 md:pr-16 text-sm overflow-x-auto">
                <code className="block whitespace-pre-wrap wrap-break-word">
                  {getCurrentCode()}
                </code>
              </pre>

              <button
                onClick={handleCopy}
                className={`absolute right-2 top-2 rounded px-2 py-1 text-sm text-white transition-colors border border-[#ffffff20] ${
                  copied ? "bg-green-500" : "bg-blue-600 hover:bg-blue-700"
                }`}
              >
                {copied ? "Copied!" : "Copy"}
              </button>
            </div>
          )}

          <a
            href="/installation"
            className="inline-flex items-center gap-1 mt-4 md:mt-5 text-blue-400 hover:text-blue-600 text-sm md:text-base"
          >
            Read more about installation
            <svg
              width="18"
              height="18"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              strokeWidth="2"
              strokeLinecap="round"
              strokeLinejoin="round"
            >
              <path d="M5 12h14M12 5l7 7-7 7" />
            </svg>
          </a>
        </div>

        <div className="w-full lg:w-[50%]">
          <div className="flex-1 relative">
            <LiteYouTubeEmbed
              videoId="KaNLPkuu03M"
              title="How to install Postgresus"
              thumbnailSrc="/images/index/how-to-install-preview.svg"
            />
          </div>
        </div>
      </div>
    </div>
  );
}
