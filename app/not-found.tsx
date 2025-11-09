import type { Metadata } from "next";
import Link from "next/link";
import Image from "next/image";

export const metadata: Metadata = {
  title: "404 - Page Not Found | Postgresus",
  description: "The page you're looking for doesn't exist.",
  robots: "noindex, nofollow",
};

export default function NotFound() {
  return (
    <div className="flex min-h-screen flex-col">
      {/* Navbar */}
      <nav className="flex h-[60px] w-full justify-center border-b border-gray-200 bg-white sm:h-[70px] md:h-[80px]">
        <div className="flex min-w-0 grow items-center px-4 sm:px-6 md:px-10">
          <Link href="/" className="flex items-center">
            <Image
              src="/logo.svg"
              alt="Postgresus logo"
              width={30}
              height={30}
              className="shrink-0 sm:h-[40px] sm:w-[40px] md:h-[50px] md:w-[50px]"
              priority
            />

            <div className="ml-2 select-none text-lg font-bold sm:ml-3 sm:text-xl md:ml-4 md:text-2xl">
              Postgresus
            </div>
          </Link>

          <div className="ml-auto mr-4 hidden gap-3 sm:mr-6 md:mr-10 lg:flex lg:gap-5">
            <a className="hover:opacity-70" href="/docs">
              Docs
            </a>
            <a
              className="hover:opacity-70"
              href="https://t.me/postgresus_community"
              target="_blank"
              rel="noopener noreferrer"
            >
              Community
            </a>
          </div>

          <a
            className="ml-auto lg:ml-0"
            href="https://github.com/RostislavDugin/postgresus"
            target="_blank"
            rel="noopener noreferrer"
          >
            <div className="flex items-center rounded-lg border border-gray-200 bg-[#f5f7f9] px-2 py-1 hover:bg-gray-100 md:px-4 md:py-2">
              <Image
                src="/images/index/github.svg"
                className="mr-1 h-4 w-4 sm:mr-2 md:mr-3"
                alt="GitHub icon"
                width={16}
                height={16}
                priority
              />
              <span className="text-sm sm:text-base">
                Star on GitHub
                <span className="hidden sm:inline">
                  , it&apos;s really important ❤️
                </span>
              </span>
            </div>
          </a>
        </div>
      </nav>

      {/* 404 Content */}
      <div className="flex grow flex-col items-center justify-center bg-linear-to-b from-white to-gray-50 px-6 py-12 text-center">
        <div className="mb-4">
          <h1 className="text-7xl font-bold text-blue-600 md:text-8xl">404</h1>
        </div>

        <h2 className="mb-3 text-2xl font-bold text-gray-800 md:text-3xl">
          Page Not Found
        </h2>

        <p className="mb-6 max-w-md text-base text-gray-600">
          The page you&apos;re looking for doesn&apos;t exist or has been moved.
        </p>

        <div className="flex flex-col gap-3 sm:flex-row">
          <Link
            href="/"
            className="rounded-lg bg-blue-600 px-5 py-2.5 text-sm font-semibold text-white transition-colors hover:bg-blue-700 md:px-6 md:py-3 md:text-base"
          >
            Go to Homepage
          </Link>

          <Link
            href="/installation"
            className="rounded-lg border-2 border-blue-600 bg-white px-5 py-2.5 text-sm font-semibold text-blue-600 transition-colors hover:bg-blue-50 md:px-6 md:py-3 md:text-base"
          >
            View Documentation
          </Link>
        </div>
      </div>

      {/* Footer */}
      <footer className="border-t border-gray-200 bg-white py-8 text-center text-sm text-gray-600">
        <p>
          © {new Date().getFullYear()} Postgresus. Open source PostgreSQL backup
          tool.
        </p>
      </footer>
    </div>
  );
}
