"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";
import { importRepository } from "@/lib/api";

export default function ImportRepositoryPage() {
  const router = useRouter();
  const [formData, setFormData] = useState({
    name: "",
    description: "",
    clone_url: "",
    username: "",
    password: "",
    is_private: false,
    mirror: false,
  });
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  const handleChange = (
    e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>,
  ) => {
    const { name, value, type } = e.target;
    if (type === "checkbox") {
      const checked = (e.target as HTMLInputElement).checked;
      setFormData((prev) => ({ ...prev, [name]: checked }));
    } else {
      setFormData((prev) => ({ ...prev, [name]: value }));
    }
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);

    // Validate repository name
    if (!formData.name.trim()) {
      setError("Repository name is required");
      return;
    }

    // Validate clone URL
    if (!formData.clone_url.trim()) {
      setError("Clone URL is required");
      return;
    }

    // Validate URL format
    try {
      new URL(formData.clone_url);
    } catch {
      setError("Invalid clone URL format");
      return;
    }

    // Validate repository name format
    const validNameRegex = /^[a-zA-Z0-9._-]+$/;
    if (!validNameRegex.test(formData.name)) {
      setError(
        "Repository name can only contain letters, numbers, hyphens, underscores, and periods",
      );
      return;
    }

    if (formData.name.length > 100) {
      setError("Repository name must be 100 characters or less");
      return;
    }

    setLoading(true);

    try {
      const repo = await importRepository({
        name: formData.name,
        clone_url: formData.clone_url,
        description: formData.description || undefined,
        username: formData.username || undefined,
        password: formData.password || undefined,
        is_private: formData.is_private,
        mirror: formData.mirror,
      });

      // Redirect to the new repository
      router.push(`/${repo.owner}/${repo.name}`);
    } catch (err) {
      setError(
        err instanceof Error ? err.message : "Failed to import repository",
      );
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="container mx-auto py-10 px-4 max-w-2xl">
      <div className="mb-8">
        <h1 className="text-3xl font-bold text-base">
          Import a repository
        </h1>
        <p className="mt-2 text-muted">
          Import an existing Git repository from GitHub, GitLab, Bitbucket, or
          any other Git hosting service.
        </p>
      </div>

      <form className="space-y-6" onSubmit={handleSubmit}>
        {error && (
          <div className="bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 text-red-600 dark:text-red-400 px-4 py-3 rounded-md text-sm">
            {error}
          </div>
        )}

        <div>
          <label
            htmlFor="clone_url"
            className="block text-sm font-medium text-base"
          >
            Clone URL <span className="text-red-500">*</span>
          </label>
          <input
            id="clone_url"
            name="clone_url"
            type="url"
            required
            value={formData.clone_url}
            onChange={handleChange}
            className="mt-1 block w-full px-3 py-2 border border-base rounded-md shadow-sm bg-panel text-base focus:outline-none focus:ring-2 focus:ring-accent focus:border-transparent"
            placeholder="https://github.com/username/repository.git"
          />
          <p className="mt-1 text-xs text-muted">
            The Git URL of the repository you want to import.
          </p>
        </div>

        <div>
          <label
            htmlFor="name"
            className="block text-sm font-medium text-base"
          >
            Repository name <span className="text-red-500">*</span>
          </label>
          <input
            id="name"
            name="name"
            type="text"
            required
            value={formData.name}
            onChange={handleChange}
            className="mt-1 block w-full px-3 py-2 border border-base rounded-md shadow-sm bg-panel text-base focus:outline-none focus:ring-2 focus:ring-accent focus:border-transparent"
            placeholder="my-imported-project"
            maxLength={100}
            pattern="^[a-zA-Z0-9._-]+$"
          />
          <p className="mt-1 text-xs text-muted">
            Use letters, numbers, hyphens, underscores, or periods. Max 100
            characters.
          </p>
        </div>

        <div>
          <label
            htmlFor="description"
            className="block text-sm font-medium text-base"
          >
            Description <span className="text-muted">(optional)</span>
          </label>
          <textarea
            id="description"
            name="description"
            rows={3}
            value={formData.description}
            onChange={handleChange}
            className="mt-1 block w-full px-3 py-2 border border-base rounded-md shadow-sm bg-panel text-base focus:outline-none focus:ring-2 focus:ring-accent focus:border-transparent resize-none"
            placeholder="A short description of your repository"
            maxLength={500}
          />
          <p className="mt-1 text-xs text-muted">Max 500 characters.</p>
        </div>

        <div className="border border-base rounded-md p-4 bg-panel space-y-4">
          <h3 className="text-sm font-medium text-base">
            Authentication <span className="text-muted">(optional)</span>
          </h3>
          <p className="text-xs text-muted">
            If the repository is private, provide authentication credentials.
          </p>

          <div>
            <label
              htmlFor="username"
              className="block text-sm font-medium text-base"
            >
              Username
            </label>
            <input
              id="username"
              name="username"
              type="text"
              value={formData.username}
              onChange={handleChange}
              className="mt-1 block w-full px-3 py-2 border border-base rounded-md shadow-sm bg-panel text-base focus:outline-none focus:ring-2 focus:ring-accent focus:border-transparent"
              placeholder="git-username"
            />
          </div>

          <div>
            <label
              htmlFor="password"
              className="block text-sm font-medium text-base"
            >
              Password / Token
            </label>
            <input
              id="password"
              name="password"
              type="password"
              value={formData.password}
              onChange={handleChange}
              className="mt-1 block w-full px-3 py-2 border border-base rounded-md shadow-sm bg-panel text-base focus:outline-none focus:ring-2 focus:ring-accent focus:border-transparent"
              placeholder="Personal access token or password"
            />
            <p className="mt-1 text-xs text-muted">
              For GitHub/GitLab, use a personal access token instead of your
              password.
            </p>
          </div>
        </div>

        <div className="border border-base rounded-md p-4 bg-panel space-y-3">
          <div className="flex items-start gap-3">
            <input
              id="is_private"
              name="is_private"
              type="checkbox"
              checked={formData.is_private}
              onChange={handleChange}
              className="mt-1 h-4 w-4 text-accent border-base rounded focus:ring-accent"
            />
            <div>
              <label
                htmlFor="is_private"
                className="block text-sm font-medium text-base cursor-pointer"
              >
                Private repository
              </label>
              <p className="text-xs text-muted mt-1">
                Only you and collaborators you explicitly add will be able to
                see this repository.
              </p>
            </div>
          </div>

          <div className="flex items-start gap-3">
            <input
              id="mirror"
              name="mirror"
              type="checkbox"
              checked={formData.mirror}
              onChange={handleChange}
              className="mt-1 h-4 w-4 text-accent border-base rounded focus:ring-accent"
            />
            <div>
              <label
                htmlFor="mirror"
                className="block text-sm font-medium text-base cursor-pointer"
              >
                Mirror repository
              </label>
              <p className="text-xs text-muted mt-1">
                Create a mirror that stays synchronized with the source
                repository. All refs (branches and tags) will be copied.
              </p>
            </div>
          </div>
        </div>

        <div className="flex items-center justify-between pt-4 border-t border-base">
          <Link
            href="/"
            className="text-sm text-muted hover:text-accent hover:underline"
          >
            Cancel
          </Link>
          <button
            type="submit"
            disabled={loading}
            className="px-4 py-2 border border-transparent rounded-md shadow-sm text-sm font-medium text-white bg-blue-600 hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {loading ? "Importing..." : "Import repository"}
          </button>
        </div>
      </form>
    </div>
  );
}
