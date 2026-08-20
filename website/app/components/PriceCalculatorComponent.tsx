"use client";

import { useState } from "react";
import styles from "./PriceCalculatorComponent.module.css";

const PRICE_PER_GB_USD = 0.45;
const MIN_GB_AMOUNT = 20;

function distributeGfs(total: number): {
  daily: number;
  weekly: number;
  monthly: number;
  yearly: number;
} {
  const daily = Math.min(7, total);
  const weekly = Math.min(4, Math.max(0, total - daily));
  const monthly = Math.min(12, Math.max(0, total - daily - weekly));
  const yearly = Math.max(0, total - daily - weekly - monthly);
  return { daily, weekly, monthly, yearly };
}

function buildBackupSizeSteps(): number[] {
  const values: number[] = [];
  for (let i = 1; i <= 100; i++) values.push(i);
  for (let i = 110; i <= 200; i += 10) values.push(i);
  return values;
}

function buildStorageSizeSteps(): number[] {
  const values: number[] = [];
  for (let i = 20; i <= 100; i++) values.push(i);
  for (let i = 110; i <= 1000; i += 10) values.push(i);
  for (let i = 1100; i <= 5000; i += 100) values.push(i);
  for (let i = 6000; i <= 10000; i += 1000) values.push(i);
  return values;
}

const BACKUP_SIZE_STEPS = buildBackupSizeSteps();
const STORAGE_SIZE_STEPS = buildStorageSizeSteps();

function formatSize(gb: number): string {
  if (gb >= 1000) {
    const tb = gb / 1000;
    return tb % 1 === 0 ? `${tb} TB` : `${tb.toFixed(1)} TB`;
  }
  return `${gb} GB`;
}

function sliderBackground(pos: number, max: number): React.CSSProperties {
  const pct = (pos / max) * 100;
  return {
    background: `linear-gradient(to right, #155dfc ${pct}%, #1f2937 ${pct}%)`,
  };
}

const DB_SIZE_COMMANDS = [
  {
    label: "PostgreSQL",
    code: `SELECT pg_size_pretty(pg_database_size(current_database()));`,
  },
  {
    label: "MySQL / MariaDB",
    code: `SELECT table_schema AS 'Database',
  ROUND(SUM(data_length + index_length) / 1024 / 1024, 2) AS 'Size (MB)'
FROM information_schema.tables
GROUP BY table_schema;`,
  },
  {
    label: "MongoDB",
    code: `db.stats(1024 * 1024)  // size in MB`,
  },
];

export function PriceCalculatorComponent() {
  const [backupSliderPos, setBackupSliderPos] = useState(0);
  const [storageSliderPos, setStorageSliderPos] = useState(0);
  const [copiedIndex, setCopiedIndex] = useState<number | null>(null);

  const singleBackupSizeGb = BACKUP_SIZE_STEPS[backupSliderPos];

  const minStoragePosIndex = STORAGE_SIZE_STEPS.findIndex(
    (s) => s >= singleBackupSizeGb,
  );
  const minStoragePos =
    minStoragePosIndex === -1
      ? STORAGE_SIZE_STEPS.length - 1
      : minStoragePosIndex;
  const effectiveStoragePos = Math.max(storageSliderPos, minStoragePos);
  const storageSizeGb = STORAGE_SIZE_STEPS[effectiveStoragePos];

  const backupsCompressionRatio = 10;

  const price = storageSizeGb * PRICE_PER_GB_USD;
  const approximateDbSize = singleBackupSizeGb * backupsCompressionRatio;
  const backupsFit = Math.floor(storageSizeGb / singleBackupSizeGb);

  return (
    <div className="mx-auto w-full max-w-[700px]">
      <div className="relative border border-[#ffffff20] rounded-xl p-5 md:p-6">
        {/* Decorative glow */}
        <div className="absolute top-0 right-0 w-[250px] h-[250px] bg-[#155dfc]/5 rounded-full blur-3xl pointer-events-none" />

        <div className="relative">
          <div className="md:grid md:grid-cols-2 md:gap-6">
            {/* Left column: sliders */}
            <div>
              {/* Storage size slider */}
              <div className="mb-5">
                <div className="flex items-baseline mb-2">
                  <label className=" md:text-base font-medium min-w-[125px]">
                    Storage size
                  </label>

                  <span className=" md:text-base font-bold text-blue-500">
                    {formatSize(storageSizeGb)}
                  </span>
                </div>

                <input
                  type="range"
                  className={styles.calcSlider}
                  style={sliderBackground(
                    effectiveStoragePos,
                    STORAGE_SIZE_STEPS.length - 1,
                  )}
                  min={0}
                  max={STORAGE_SIZE_STEPS.length - 1}
                  value={effectiveStoragePos}
                  onChange={(e) => setStorageSliderPos(Number(e.target.value))}
                />

                <div className="flex justify-between mt-1.5  text-gray-500">
                  <span>{formatSize(MIN_GB_AMOUNT)}</span>
                  <span>10 TB</span>
                </div>
              </div>

              {/* Single backup size slider */}
              <div className="mb-4">
                <div className="flex items-baseline mb-1">
                  <label className=" md:text-base font-medium min-w-[210px]">
                    If your single backup size is
                  </label>

                  <span className=" md:text-base font-bold text-blue-500">
                    {formatSize(singleBackupSizeGb)}
                  </span>
                </div>

                <p className="mb-2 text-gray-400 flex items-center">
                  <span className="flex items-center gap-1 min-w-[200px]">
                    Approximate DB size{" "}
                    <span className="relative inline-block group">
                      <svg
                        width="14"
                        height="14"
                        viewBox="0 0 24 24"
                        fill="none"
                        stroke="currentColor"
                        strokeWidth="2"
                        strokeLinecap="round"
                        strokeLinejoin="round"
                        className="inline -mt-0.5 text-gray-500 cursor-help"
                      >
                        <circle cx="12" cy="12" r="10" />
                        <path d="M12 16v-4M12 8h.01" />
                      </svg>
                      <span className="pointer-events-none absolute bottom-full left-1/2 -translate-x-1/2 mb-2 w-52 rounded-lg bg-[#1f2937] border border-[#ffffff20] px-3 py-2  text-gray-300 opacity-0 group-hover:opacity-100 transition-opacity">
                        Estimated with ~10x compression ratio typical for
                        database backups. Can differ based on the database type,
                        structure, and content.
                      </span>
                    </span>
                  </span>

                  <span className="text-gray-200 font-medium">
                    ~{formatSize(approximateDbSize)}
                  </span>
                </p>

                {/* DB size commands */}
                <details className="mb-2 group">
                  <summary className=" text-gray-500 cursor-pointer hover:text-gray-400 transition-colors list-none flex items-center gap-1.5">
                    <svg
                      width="12"
                      height="12"
                      viewBox="0 0 24 24"
                      fill="none"
                      stroke="currentColor"
                      strokeWidth="2"
                      strokeLinecap="round"
                      strokeLinejoin="round"
                      className="transition-transform group-open:rotate-90"
                    >
                      <path d="M9 18l6-6-6-6" />
                    </svg>
                    How to check DB size?
                  </summary>

                  <div className="mt-2 space-y-1.5">
                    {DB_SIZE_COMMANDS.map((cmd, index) => (
                      <div key={index}>
                        <p className=" text-gray-400 mb-1">{cmd.label}</p>
                        <div className="relative">
                          <pre className="rounded-lg bg-[#1f2937] border border-[#ffffff20] px-2.5 py-1.5 pr-16  overflow-x-auto">
                            <code className="block whitespace-pre text-gray-300">
                              {cmd.code}
                            </code>
                          </pre>
                          <button
                            onClick={async () => {
                              try {
                                await navigator.clipboard.writeText(cmd.code);
                                setCopiedIndex(index);
                                setTimeout(() => setCopiedIndex(null), 2000);
                              } catch {}
                            }}
                            className={`absolute right-2 top-2 rounded px-2 py-0.5  text-white transition-colors border border-[#ffffff20] ${
                              copiedIndex === index
                                ? "bg-green-500"
                                : "bg-blue-600 hover:bg-blue-700"
                            }`}
                          >
                            {copiedIndex === index ? "Copied!" : "Copy"}
                          </button>
                        </div>
                      </div>
                    ))}
                  </div>
                </details>

                <input
                  type="range"
                  className={styles.calcSlider}
                  style={sliderBackground(
                    backupSliderPos,
                    BACKUP_SIZE_STEPS.length - 1,
                  )}
                  min={0}
                  max={BACKUP_SIZE_STEPS.length - 1}
                  value={backupSliderPos}
                  onChange={(e) => setBackupSliderPos(Number(e.target.value))}
                />

                <div className="flex justify-between mt-1.5  text-gray-500">
                  <span>1 GB</span>
                  <span>200 GB</span>
                </div>
              </div>
            </div>

            {/* Right column: retention */}
            <div>
              {/* Retention estimation (GFS) */}
              <div>
                {(() => {
                  const gfs = distributeGfs(backupsFit);
                  return (
                    <div className="space-y-1.5">
                      <div className="bg-[#1f2937]/50 border border-[#ffffff20] rounded-lg px-3 py-2 text-center">
                        <p className=" text-gray-500 mb-0.5">Total backups</p>
                        <p className="text-lg font-bold text-gray-200">
                          {backupsFit}
                        </p>
                      </div>

                      <div className="flex items-center gap-3 my-1">
                        <div className="flex-1 h-px bg-[#ffffff20]" />
                        <span className="text-sm text-gray-500">or</span>
                        <div className="flex-1 h-px bg-[#ffffff20]" />
                      </div>

                      <p className=" text-gray-400 mb-2 md:text-sm">
                        Keeps recent backups frequently, older ones less often —
                        broad time at the lowest cost. It is enough to keep the
                        following amount of backups in GFS:
                      </p>

                      <div className="grid grid-cols-2 gap-1.5">
                        {(
                          [
                            ["Daily", gfs.daily],
                            ["Weekly", gfs.weekly],
                            ["Monthly", gfs.monthly],
                            ["Yearly", gfs.yearly],
                          ] as const
                        ).map(([label, value]) => (
                          <div
                            key={label}
                            className="bg-[#1f2937]/50 border border-[#ffffff20] rounded-lg px-2 py-1.5 text-center"
                          >
                            <p className="text-xs text-gray-500">{label}</p>
                            <p className="text-base font-bold text-gray-200">
                              {value}
                            </p>
                          </div>
                        ))}
                      </div>
                    </div>
                  );
                })()}

                <p className=" text-gray-400 mt-2 md:text-sm mb-3">
                  You can fine-tune retention values (change daily count, keep
                  only monthly, keep N latest, etc.)
                </p>
              </div>
            </div>
          </div>

          {/* Price display */}
          <div className="border-t border-[#ffffff20] pt-5 text-center">
            <p className=" text-gray-400 mb-2">Monthly price (per DB)</p>
            <p className="text-3xl md:text-4xl font-bold">
              ${price.toFixed(2)}
              <span className="text-lg md:text-xl font-medium text-gray-400">
                /mo
              </span>
            </p>
          </div>
        </div>
      </div>
    </div>
  );
}
