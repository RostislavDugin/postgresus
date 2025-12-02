import { useEffect, useState } from 'react';

/**
 * This hook detects if the current device is mobile (screen width <= 768px)
 * and adjusts dynamically when the window is resized.
 *
 * @returns isMobile boolean
 */
export function useIsMobile(): boolean {
  // Initialize with actual value to avoid race conditions
  const [isMobile, setIsMobile] = useState<boolean>(() => window.innerWidth <= 768);

  useEffect(() => {
    const updateIsMobile = () => {
      setIsMobile(window.innerWidth <= 768);
    };

    window.addEventListener('resize', updateIsMobile);

    return () => {
      window.removeEventListener('resize', updateIsMobile);
    };
  }, []);

  return isMobile;
}
