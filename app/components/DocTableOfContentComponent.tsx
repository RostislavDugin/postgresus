"use client";

import { useEffect, useState, useRef } from "react";

interface Heading {
  id: string;
  text: string;
  level: number;
}

export default function DocTableOfContentComponent() {
  const [headings, setHeadings] = useState<Heading[]>([]);
  const [activeId, setActiveId] = useState<string>("");
  const observerRef = useRef<IntersectionObserver | null>(null);

  useEffect(() => {
    // Use a timeout to avoid cascading renders
    const timeoutId = setTimeout(() => {
      // Get all h1, h2, h3 headings from the page
      const elements = Array.from(
        document.querySelectorAll("article h1, article h2, article h3")
      );

      if (elements.length === 0) return;

      const headingData: Heading[] = elements.map((element) => ({
        id: element.id,
        text: element.textContent || "",
        level: parseInt(element.tagName.substring(1)),
      }));

      setHeadings(headingData);

      // Set up intersection observer to track active heading
      observerRef.current = new IntersectionObserver(
        (entries) => {
          entries.forEach((entry) => {
            if (entry.isIntersecting) {
              setActiveId(entry.target.id);
            }
          });
        },
        { rootMargin: "-100px 0px -80% 0px" }
      );

      elements.forEach((element) => {
        observerRef.current?.observe(element);
      });
    }, 0);

    return () => {
      clearTimeout(timeoutId);
      if (observerRef.current) {
        observerRef.current.disconnect();
      }
    };
  }, []);

  const handleClick = (id: string) => {
    const element = document.getElementById(id);
    if (element) {
      element.scrollIntoView({ behavior: "smooth", block: "start" });
    }
  };

  if (headings.length === 0) {
    return null;
  }

  return (
    <aside className="hidden w-64 border-l border-gray-200 bg-white xl:block">
      <div className="sticky top-0 h-screen overflow-y-auto p-6">
        <h3 className="mb-4 text-sm font-semibold text-gray-900">
          On This Page
        </h3>
        <nav>
          <ul className="space-y-2 text-sm">
            {headings.map((heading) => (
              <li
                key={heading.id}
                style={{ paddingLeft: `${(heading.level - 1) * 0.75}rem` }}
              >
                <button
                  onClick={() => handleClick(heading.id)}
                  className={`block w-full text-left transition-colors cursor-pointer hover:text-blue-600 ${
                    activeId === heading.id
                      ? "font-medium text-blue-600"
                      : "text-gray-600"
                  }`}
                >
                  {heading.text}
                </button>
              </li>
            ))}
          </ul>
        </nav>
      </div>
    </aside>
  );
}
