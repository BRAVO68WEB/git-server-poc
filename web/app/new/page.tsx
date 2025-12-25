"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";
import { createRepository } from "@/lib/api";

export default function NewRepositoryPage() {
  const router = useRouter();
  const [formData, setFormData] = useState({
    name: "",
    description: "",
    is_private: false,
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
      const repo = await createRepository({
        name: formData.name,
        description: formData.description || undefined,
        is_private: formData.is_private,
      });

      // Redirect to the new repository
      router.push(`/${repo.owner}/${repo.name}`);
    } catch (err) {
      setError(
        err instanceof Error ? err.message : "Failed to create repository",
      );
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="container mx-auto py-10 px-4 max-w-2xl">
      <div className="mb-8">
        <h1 className="text-3xl font-bold text-base">Create a new repository</h1>
        <p className="mt-2 text-muted">
          A repository contains all project files, including the revision
          history.
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
            placeholder="my-awesome-project"
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

        <div className="border border-base rounded-md p-4 bg-panel">
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
            className="px-4 py-2 border border-transparent rounded-md shadow-sm text-sm font-medium text-white bg-green-600 hover:bg-green-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-green-500 disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {loading ? "Creating..." : "Create repository"}
          </button>
        </div>
      </form>
    </div>
  );
}
