export const NAME_LIST_TOKEN_SEPARATORS = [',', '\n', '\r', '\t'];

export const normalizeNameList = (rawNames: string[]): string[] => {
  const normalizedNames: string[] = [];

  for (const rawName of rawNames) {
    for (const name of rawName.split(new RegExp(`[${NAME_LIST_TOKEN_SEPARATORS.join('')}]`))) {
      const trimmedName = name.trim();

      if (trimmedName && !normalizedNames.includes(trimmedName)) {
        normalizedNames.push(trimmedName);
      }
    }
  }

  return normalizedNames;
};
