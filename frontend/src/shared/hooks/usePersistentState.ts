import { useEffect, useState } from 'react';

function readPersistedValue<T>(storageKey: string, fallbackValue: T): T {
  try {
    const raw = localStorage.getItem(storageKey);
    if (!raw) {
      return fallbackValue;
    }

    return JSON.parse(raw) as T;
  } catch {
    return fallbackValue;
  }
}

// Drop-in useState replacement that mirrors its value into localStorage under storageKey,
// so the value survives reloads. Use only for small, JSON-serializable UI state.
export function usePersistentState<T>(storageKey: string, fallbackValue: T) {
  const [value, setValue] = useState<T>(() => readPersistedValue(storageKey, fallbackValue));

  useEffect(() => {
    try {
      localStorage.setItem(storageKey, JSON.stringify(value));
    } catch {
      // Ignore quota or serialization errors - persistence is best-effort.
    }
  }, [storageKey, value]);

  return [value, setValue] as const;
}
