"use client";

import Image from "next/image";
import Link from "next/link";

export default function DocsNavbarComponent() {
  return (
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
  );
}
