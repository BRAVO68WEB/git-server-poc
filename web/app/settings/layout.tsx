"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";

const settingsNav = [
  {
    name: "SSH Keys",
    href: "/settings/ssh-keys",
    description: "Manage SSH keys for Git access",
  },
];

export default function SettingsLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  const pathname = usePathname();

  return (
    <div className="container mx-auto py-10 px-4">
      <div className="mb-8">
        <h1 className="text-3xl font-bold text-base">Settings</h1>
        <p className="mt-2 text-muted">
          Manage your account settings and preferences
        </p>
      </div>

      <div className="flex flex-col md:flex-row gap-8">
        {/* Sidebar Navigation */}
        <nav className="w-full md:w-64 shrink-0">
          <ul className="space-y-1">
            {settingsNav.map((item) => {
              const isActive = pathname === item.href;
              return (
                <li key={item.href}>
                  <Link
                    href={item.href}
                    className={`block px-4 py-2 rounded-md text-sm transition-colors ${
                      isActive
                        ? "bg-panel border border-base text-accent font-medium"
                        : "text-muted hover:text-base hover:bg-panel"
                    }`}
                  >
                    {item.name}
                  </Link>
                </li>
              );
            })}
          </ul>
        </nav>

        {/* Main Content */}
        <main className="flex-1 min-w-0">{children}</main>
      </div>
    </div>
  );
}
