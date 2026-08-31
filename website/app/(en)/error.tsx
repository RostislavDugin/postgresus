"use client";

export default function Error({ reset }: { error: Error; reset: () => void }) {
  return (
    <div className="flex min-h-screen items-center justify-center bg-[#0F1115]">
      <div className="text-center">
        <p className="mb-4 text-gray-400">Something went wrong.</p>
        <button
          onClick={reset}
          className="rounded-lg bg-blue-600 px-4 py-2 text-white hover:bg-blue-700"
        >
          Try again
        </button>
      </div>
    </div>
  );
}
