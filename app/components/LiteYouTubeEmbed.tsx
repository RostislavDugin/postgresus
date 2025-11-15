"use client";

import { useState } from "react";

interface LiteYouTubeEmbedProps {
  videoId: string;
  title: string;
  thumbnailSrc: string;
}

export default function LiteYouTubeEmbed({
  videoId,
  title,
  thumbnailSrc,
}: LiteYouTubeEmbedProps) {
  const [isIframeLoaded, setIsIframeLoaded] = useState(false);

  const handleClick = () => {
    setIsIframeLoaded(true);
  };

  return (
    <div
      className="relative w-full overflow-hidden rounded-lg shadow-lg"
      style={{ paddingBottom: "56.25%" }}
    >
      {!isIframeLoaded ? (
        <>
          <img
            src={thumbnailSrc}
            alt={title}
            loading="lazy"
            className="absolute left-0 top-0 h-full w-full object-cover"
          />
          <button
            onClick={handleClick}
            className="absolute cursor-pointer left-0 top-0 z-10 flex h-full w-full items-center justify-center transition-all hover:bg-opacity-20"
            aria-label={`Play ${title}`}
          >
            <div className="flex items-center justify-center rounded-xl bg-red-600 px-5 py-2 transition-transform hover:scale-105 sm:px-5 sm:py-2">
              <svg
                className="h-6 w-6 text-white sm:h-8 sm:w-8"
                fill="currentColor"
                viewBox="0 0 24 24"
              >
                <path d="M8 5v14l11-7z" />
              </svg>
            </div>
          </button>
        </>
      ) : (
        <iframe
          src={`https://www.youtube.com/embed/${videoId}?autoplay=1`}
          title={title}
          allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture"
          allowFullScreen
          className="absolute left-0 top-0 h-full w-full"
        />
      )}
    </div>
  );
}
