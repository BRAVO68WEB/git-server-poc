"use client";

import Link from "next/link";
import { useParams, usePathname } from "next/navigation";

export default function SettingsLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  const params = useParams();
  const pathname = usePathname();
  const username = params.username as string;
  const repo = params.repo as string;

  const tabs = [
    {
      name: "General",
      href: `/${username}/${repo}/settings`,
      path: `/${username}/${repo}/settings`,
    },
    {
      name: "Mirror",
      href: `/${username}/${repo}/settings/mirror`,
      path: `/${username}/${repo}/settings/mirror`,
    },
  ];

  const isActive = (tabPath: string) => {
    return pathname === tabPath;
  };

  return (
    <div className="container mx-auto py-10 px-4">
      <div className="mb-8">
        {/*<Link
          href={`/${username}/${repo}`}
          className="text-sm text-accent hover:underline mb-4 inline-block"
        >
          ← Back to Repository
        </Link>*/}
        <h1 className="text-2xl font-bold">Repository Settings</h1>
        <p className="mt-2 text-muted">
          Manage settings for {username}/{repo}
        </p>
      </div>

      {/* Tabs */}
      <div className="border-b border-base mb-8">
        <nav className="-mb-px flex space-x-8" aria-label="Tabs">
          {tabs.map((tab) => (
            <Link
              key={tab.name}
              href={tab.href}
              className={`
                whitespace-nowrap py-4 px-1 border-b-2 font-medium text-sm
                ${
                  isActive(tab.path)
                    ? "border-accent text-accent"
                    : "border-transparent text-muted hover:text-base hover:border-base"
                }
              `}
            >
              {tab.name}
            </Link>
          ))}
        </nav>
      </div>

      {/* Tab Content */}
      <div className="max-w-2xl">{children}</div>
    </div>
  );
}
