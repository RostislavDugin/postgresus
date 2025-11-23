import { useEffect, useState } from 'react';

/**
 * This hook detects if the current device is mobile (screen width <= 768px)
 * and adjusts dynamically when the window is resized.
 *
 * @returns isMobile boolean
 */
export function useIsMobile(): boolean {
  const [isMobile, setIsMobile] = useState<boolean>(false);

  useEffect(() => {
    const updateIsMobile = () => {
      setIsMobile(window.innerWidth <= 768);
    };

    updateIsMobile(); // Set initial value
    window.addEventListener('resize', updateIsMobile);

    return () => {
      window.removeEventListener('resize', updateIsMobile);
    };
  }, []);

  return isMobile;
}
