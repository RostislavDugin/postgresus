/* eslint-disable i18next/no-literal-string -- unit symbols stay English in every language, like the backup lists */
const BYTE_UNITS = ['B', 'KB', 'MB', 'GB', 'TB'] as const;

const BYTES_PER_UNIT_STEP = 1024;

type NumberFormatter = (value: number, options?: Intl.NumberFormatOptions) => string;

// Base 1024, the way PostgreSQL and the backup lists count, so 16 MB is one WAL segment. The
// number goes through the formatter from useLocale(), so digits are grouped by the selected
// language.
export const formatBytes = (bytes: number, formatNumber: NumberFormatter): string => {
  let unitIndex = 0;
  let unitValue = Math.abs(bytes);

  while (unitValue >= BYTES_PER_UNIT_STEP && unitIndex < BYTE_UNITS.length - 1) {
    unitValue /= BYTES_PER_UNIT_STEP;
    unitIndex++;
  }

  const signedValue = bytes < 0 ? -unitValue : unitValue;

  return `${formatNumber(signedValue, { maximumFractionDigits: 1 })} ${BYTE_UNITS[unitIndex]}`;
};
