"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { useState, useEffect } from "react";

interface NavItem {
  title: string;
  href: string;
  children?: NavItem[];
}

const navItems: NavItem[] = [
  {
    title: "Installation",
    href: "/installation",
  },
  {
    title: "Storages",
    href: "/storages",
    children: [
      { title: "Google Drive", href: "/storages/google-drive" },
      { title: "Cloudflare R2", href: "/storages/cloudflare-r2" },
    ],
  },
  {
    title: "Notifiers",
    href: "/notifiers",
    children: [
      { title: "Slack", href: "/notifiers/slack" },
      { title: "Microsoft Teams", href: "/notifiers/teams" },
    ],
  },
  {
    title: "Access management",
    href: "/access-management",
  },
  {
    title: "Reset password",
    href: "/password",
  },
  {
    title: "Security",
    href: "/security",
  },
  {
    title: "FAQ",
    href: "/faq",
  },
  {
    title: "Contribute",
    href: "/contribute",
    children: [
      { title: "How to add storage", href: "/contribute/how-to-add-storage" },
      { title: "How to add notifier", href: "/contribute/how-to-add-notifier" },
    ],
  },
  {
    title: "Comparisons",
    href: "/pgdump-alternative",
    children: [
      { title: "pg_dump alternative", href: "/pgdump-alternative" },
      { title: "Postgresus vs Barman", href: "/postgresus-vs-barman" },
      { title: "Postgresus vs PgBackWeb", href: "/postgresus-vs-pgbackweb" },
      { title: "Postgresus vs pgBackRest", href: "/postgresus-vs-pgbackrest" },
      { title: "Postgresus vs WAL-G", href: "/postgresus-vs-wal-g" },
    ],
  },
];

export default function DocsSidebarComponent() {
  const pathname = usePathname();
  const [isMobileMenuOpen, setIsMobileMenuOpen] = useState(false);
  const [manuallyToggledSections, setManuallyToggledSections] = useState<
    Set<string>
  >(new Set());

  const isActive = (href: string) => {
    // Normalize paths by removing trailing slashes for comparison
    const normalizedPathname =
      pathname.endsWith("/") && pathname !== "/"
        ? pathname.slice(0, -1)
        : pathname;
    const normalizedHref =
      href.endsWith("/") && href !== "/" ? href.slice(0, -1) : href;
    return normalizedPathname === normalizedHref;
  };

  const isParentActive = (item: NavItem) => {
    if (item.children) {
      const normalizedPathname =
        pathname.endsWith("/") && pathname !== "/"
          ? pathname.slice(0, -1)
          : pathname;
      return item.children.some((child) => {
        const normalizedChildHref =
          child.href.endsWith("/") && child.href !== "/"
            ? child.href.slice(0, -1)
            : child.href;
        return normalizedPathname === normalizedChildHref;
      });
    }
    return false;
  };

  // Determine if a section should be expanded
  const isSectionExpanded = (href: string) => {
    // If manually toggled, respect that
    if (manuallyToggledSections.has(href)) {
      return true;
    }
    // Auto-expand if a child page is active
    const item = navItems.find((i) => i.href === href);
    if (item && isParentActive(item)) {
      return true;
    }
    return false;
  };

  // Manage body overflow when mobile menu is open
  useEffect(() => {
    if (isMobileMenuOpen) {
      document.body.style.overflow = "hidden";
    } else {
      document.body.style.overflow = "";
    }

    // Cleanup on unmount
    return () => {
      document.body.style.overflow = "";
    };
  }, [isMobileMenuOpen]);

  const toggleSection = (href: string, hasChildren: boolean) => {
    if (!hasChildren) return;

    setManuallyToggledSections((prev) => {
      const newSet = new Set(prev);
      if (newSet.has(href)) {
        newSet.delete(href);
      } else {
        newSet.add(href);
      }
      return newSet;
    });
  };

  const renderSidebarContent = () => (
    <nav className="space-y-0.5">
      {navItems.map((item) => (
        <div key={item.href}>
          <div className="flex items-center">
            <Link
              href={item.href}
              onClick={() => setIsMobileMenuOpen(false)}
              className={`flex-1 rounded-md px-2 py-1.5 text-sm transition-colors relative pl-4 ${
                isActive(item.href)
                  ? "text-white font-medium"
                  : "text-gray-400 hover:text-white"
              }`}
            >
              <span
                className={`absolute left-0 top-0 bottom-0 w-0.5 rounded-full transition-all duration-200 ${
                  isActive(item.href)
                    ? "bg-blue-600 opacity-100"
                    : "bg-transparent opacity-0"
                }`}
              />
              {item.title}
            </Link>
            {item.children && (
              <button
                onClick={() => toggleSection(item.href, !!item.children)}
                className={`ml-1 rounded-md p-1 transition-all duration-200 ${
                  isActive(item.href)
                    ? "text-white"
                    : "text-gray-500 hover:text-gray-300"
                }`}
                aria-label={`Toggle ${item.title} section`}
              >
                <svg
                  className={`h-3.5 w-3.5 transition-transform duration-200 ${
                    isSectionExpanded(item.href) ? "rotate-90" : ""
                  }`}
                  fill="none"
                  stroke="currentColor"
                  viewBox="0 0 24 24"
                  strokeWidth={2}
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    d="M9 5l7 7-7 7"
                  />
                </svg>
              </button>
            )}
          </div>
          {item.children && (
            <div
              className={`overflow-hidden transition-all duration-200 ease-in-out ${
                isSectionExpanded(item.href)
                  ? "max-h-96 opacity-100"
                  : "max-h-0 opacity-0"
              }`}
            >
              <div className="ml-3 mt-0.5 space-y-0.5 border-l border-[#ffffff20] pl-3">
                {item.children.map((child) => (
                  <Link
                    key={child.href}
                    href={child.href}
                    onClick={() => setIsMobileMenuOpen(false)}
                    className={`block rounded-md px-2 py-1.5 text-sm transition-colors relative pl-4 ${
                      isActive(child.href)
                        ? "text-white font-medium"
                        : "text-gray-400 hover:text-white"
                    }`}
                  >
                    <span
                      className={`absolute left-0 top-0 bottom-0 w-0.5 rounded-full transition-all duration-200 ${
                        isActive(child.href)
                          ? "bg-blue-500 opacity-100"
                          : "bg-transparent opacity-0"
                      }`}
                    />
                    {child.title}
                  </Link>
                ))}
              </div>
            </div>
          )}
        </div>
      ))}
    </nav>
  );

  return (
    <>
      {/* Mobile Menu Button */}
      <button
        onClick={() => setIsMobileMenuOpen(!isMobileMenuOpen)}
        className="fixed bottom-4 right-4 z-50 flex h-14 w-14 items-center justify-center rounded-full bg-blue-600 text-white shadow-lg hover:bg-blue-700 lg:hidden"
        aria-label="Toggle navigation menu"
      >
        {isMobileMenuOpen ? (
          <svg
            className="h-6 w-6"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M6 18L18 6M6 6l12 12"
            />
          </svg>
        ) : (
          <svg
            className="h-6 w-6"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M4 6h16M4 12h16M4 18h16"
            />
          </svg>
        )}
      </button>

      {/* Mobile Menu Overlay */}
      {isMobileMenuOpen && (
        <div
          className="fixed inset-0 z-40 backdrop-blur-md bg-black/50 lg:hidden"
          onClick={() => setIsMobileMenuOpen(false)}
        />
      )}

      {/* Mobile Menu */}
      <aside
        className={`fixed bottom-0 left-0 right-0 z-40 max-h-[80vh] overflow-y-auto rounded-t-2xl border-t border-[#ffffff20] bg-[#0F1115] p-6 shadow-2xl transition-transform duration-300 lg:hidden ${
          isMobileMenuOpen ? "translate-y-0" : "translate-y-full"
        }`}
      >
        <div className="mb-4 flex items-center justify-between">
          <h2 className="text-lg font-semibold text-white">Navigation</h2>
          <button
            onClick={() => setIsMobileMenuOpen(false)}
            className="rounded-lg p-2 text-gray-400 hover:bg-[#1f2937]"
            aria-label="Close menu"
          >
            <svg
              className="h-5 w-5"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M6 18L18 6M6 6l12 12"
              />
            </svg>
          </button>
        </div>
        {renderSidebarContent()}
      </aside>

      {/* Desktop Sidebar */}
      <aside className="hidden w-64 border-r border-[#ffffff20] bg-[#0F1115] lg:block">
        <div className="sticky top-0 h-screen overflow-y-auto p-6">
          {renderSidebarContent()}
        </div>
      </aside>
    </>
  );
}
