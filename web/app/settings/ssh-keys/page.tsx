"use client";

import { useState, useEffect } from "react";
import { useRouter } from "next/navigation";
import {
  listSSHKeys,
  addSSHKey,
  deleteSSHKey,
  isAuthenticated,
} from "@/lib/api";
import { SSHKeyInfo } from "@/lib/types";

export default function SSHKeysPage() {
  const router = useRouter();
  const [keys, setKeys] = useState<SSHKeyInfo[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);

  // Add key form state
  const [showAddForm, setShowAddForm] = useState(false);
  const [addingKey, setAddingKey] = useState(false);
  const [formData, setFormData] = useState({
    title: "",
    key: "",
  });

  // Delete confirmation state
  const [deletingKeyId, setDeletingKeyId] = useState<string | null>(null);
  const [keyToDelete, setKeyToDelete] = useState<SSHKeyInfo | null>(null);

  useEffect(() => {
    // Check if user is authenticated
    if (!isAuthenticated()) {
      router.push("/auth/register");
      return;
    }

    fetchKeys();
  }, [router]);

  async function fetchKeys() {
    try {
      setLoading(true);
      setError(null);
      const response = await listSSHKeys();
      setKeys(response.keys || []);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load SSH keys");
    } finally {
      setLoading(false);
    }
  }

  const handleAddKey = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);
    setSuccess(null);

    // Validate form
    if (!formData.title.trim()) {
      setError("Title is required");
      return;
    }

    if (!formData.key.trim()) {
      setError("SSH public key is required");
      return;
    }

    // Basic SSH key format validation
    const keyPattern = /^(ssh-rsa|ssh-ed25519|ecdsa-sha2-\S+|ssh-dss)\s+\S+/;
    if (!keyPattern.test(formData.key.trim())) {
      setError(
        "Invalid SSH key format. Key should start with ssh-rsa, ssh-ed25519, ecdsa-sha2-*, or ssh-dss"
      );
      return;
    }

    setAddingKey(true);

    try {
      await addSSHKey({
        title: formData.title.trim(),
        key: formData.key.trim(),
      });

      setSuccess("SSH key added successfully");
      setFormData({ title: "", key: "" });
      setShowAddForm(false);
      await fetchKeys();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to add SSH key");
    } finally {
      setAddingKey(false);
    }
  };

  const handleDeleteKey = async () => {
    if (!keyToDelete) return;

    setDeletingKeyId(keyToDelete.id);
    setError(null);
    setSuccess(null);

    try {
      await deleteSSHKey(keyToDelete.id);
      setSuccess("SSH key deleted successfully");
      setKeyToDelete(null);
      await fetchKeys();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to delete SSH key");
    } finally {
      setDeletingKeyId(null);
    }
  };

  const formatDate = (dateString: string) => {
    return new Date(dateString).toLocaleDateString("en-US", {
      year: "numeric",
      month: "short",
      day: "numeric",
    });
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center py-12">
        <div className="text-muted">Loading SSH keys...</div>
      </div>
    );
  }

  return (
    <div className="max-w-3xl space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-xl font-semibold text-base">SSH Keys</h2>
          <p className="text-sm text-muted mt-1">
            SSH keys allow you to establish a secure connection to your Git
            repositories without entering your password.
          </p>
        </div>
        {!showAddForm && (
          <button
            onClick={() => setShowAddForm(true)}
            className="px-4 py-2 bg-green-600 hover:bg-green-700 text-white text-sm font-medium rounded-md shadow-sm transition-colors focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-green-500"
          >
            Add SSH Key
          </button>
        )}
      </div>

      {error && (
        <div className="bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 text-red-600 dark:text-red-400 px-4 py-3 rounded-md text-sm">
          {error}
        </div>
      )}

      {success && (
        <div className="bg-green-50 dark:bg-green-900/20 border border-green-200 dark:border-green-800 text-green-600 dark:text-green-400 px-4 py-3 rounded-md text-sm">
          {success}
        </div>
      )}

      {/* Add SSH Key Form */}
      {showAddForm && (
        <form
          onSubmit={handleAddKey}
          className="border border-base rounded-md bg-panel"
        >
          <div className="px-4 py-3 border-b border-base">
            <h3 className="font-medium text-base">Add new SSH Key</h3>
          </div>

          <div className="p-4 space-y-4">
            <div>
              <label
                htmlFor="title"
                className="block text-sm font-medium text-base"
              >
                Title <span className="text-red-500">*</span>
              </label>
              <input
                id="title"
                name="title"
                type="text"
                required
                value={formData.title}
                onChange={(e) =>
                  setFormData((prev) => ({ ...prev, title: e.target.value }))
                }
                className="mt-1 block w-full px-3 py-2 border border-base rounded-md shadow-sm bg-base text-base focus:outline-none focus:ring-2 focus:ring-accent focus:border-transparent"
                placeholder="My MacBook Pro"
                maxLength={100}
              />
              <p className="mt-1 text-xs text-muted">
                A descriptive name for this key (e.g., &quot;Work Laptop&quot;, &quot;Home
                PC&quot;)
              </p>
            </div>

            <div>
              <label
                htmlFor="key"
                className="block text-sm font-medium text-base"
              >
                Public Key <span className="text-red-500">*</span>
              </label>
              <textarea
                id="key"
                name="key"
                rows={5}
                required
                value={formData.key}
                onChange={(e) =>
                  setFormData((prev) => ({ ...prev, key: e.target.value }))
                }
                className="mt-1 block w-full px-3 py-2 border border-base rounded-md shadow-sm bg-base text-base font-mono text-sm focus:outline-none focus:ring-2 focus:ring-accent focus:border-transparent resize-none"
                placeholder="ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAI... user@example.com"
              />
              <p className="mt-1 text-xs text-muted">
                Paste your public SSH key. Starts with &quot;ssh-rsa&quot;, &quot;ssh-ed25519&quot;,
                &quot;ecdsa-sha2-*&quot;, or &quot;ssh-dss&quot;.
              </p>
            </div>

            <div className="bg-base border border-base rounded-md p-3">
              <h4 className="text-sm font-medium text-base mb-2">
                How to generate an SSH key:
              </h4>
              <ol className="text-xs text-muted space-y-1 list-decimal list-inside">
                <li>
                  Open a terminal and run:{" "}
                  <code className="bg-panel px-1 py-0.5 rounded text-accent">
                    ssh-keygen -t ed25519 -C &quot;your_email@example.com&quot;
                  </code>
                </li>
                <li>Press Enter to accept the default file location</li>
                <li>Enter a secure passphrase (optional but recommended)</li>
                <li>
                  Copy your public key:{" "}
                  <code className="bg-panel px-1 py-0.5 rounded text-accent">
                    cat ~/.ssh/id_ed25519.pub
                  </code>
                </li>
                <li>Paste the output above</li>
              </ol>
            </div>
          </div>

          <div className="px-4 py-3 border-t border-base flex justify-end gap-3">
            <button
              type="button"
              onClick={() => {
                setShowAddForm(false);
                setFormData({ title: "", key: "" });
              }}
              className="px-4 py-2 border border-base rounded-md text-sm font-medium text-base hover:bg-base focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-accent"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={addingKey}
              className="px-4 py-2 bg-green-600 hover:bg-green-700 text-white text-sm font-medium rounded-md shadow-sm transition-colors focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-green-500 disabled:opacity-50 disabled:cursor-not-allowed"
            >
              {addingKey ? "Adding..." : "Add SSH Key"}
            </button>
          </div>
        </form>
      )}

      {/* SSH Keys List */}
      <div className="border border-base rounded-md">
        <div className="px-4 py-3 border-b border-base bg-panel">
          <h3 className="font-medium text-base">
            Your SSH Keys ({keys.length})
          </h3>
        </div>

        {keys.length === 0 ? (
          <div className="p-8 text-center">
            <svg
              xmlns="http://www.w3.org/2000/svg"
              width="48"
              height="48"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              strokeWidth="1"
              strokeLinecap="round"
              strokeLinejoin="round"
              className="mx-auto text-muted mb-4"
            >
              <path d="M21 2l-2 2m-7.61 7.61a5.5 5.5 0 1 1-7.778 7.778 5.5 5.5 0 0 1 7.777-7.777zm0 0L15.5 7.5m0 0l3 3L22 7l-3-3m-3.5 3.5L19 4" />
            </svg>
            <p className="text-muted">No SSH keys added yet.</p>
            <p className="text-sm text-muted mt-1">
              Add an SSH key to connect to your repositories securely.
            </p>
            {!showAddForm && (
              <button
                onClick={() => setShowAddForm(true)}
                className="mt-4 px-4 py-2 bg-green-600 hover:bg-green-700 text-white text-sm font-medium rounded-md shadow-sm transition-colors"
              >
                Add your first SSH key
              </button>
            )}
          </div>
        ) : (
          <ul className="divide-y divide-base">
            {keys.map((key) => (
              <li key={key.id} className="p-4 flex items-start justify-between">
                <div className="min-w-0 flex-1">
                  <div className="flex items-center gap-2">
                    <svg
                      xmlns="http://www.w3.org/2000/svg"
                      width="16"
                      height="16"
                      viewBox="0 0 24 24"
                      fill="none"
                      stroke="currentColor"
                      strokeWidth="2"
                      strokeLinecap="round"
                      strokeLinejoin="round"
                      className="text-accent shrink-0"
                    >
                      <path d="M21 2l-2 2m-7.61 7.61a5.5 5.5 0 1 1-7.778 7.778 5.5 5.5 0 0 1 7.777-7.777zm0 0L15.5 7.5m0 0l3 3L22 7l-3-3m-3.5 3.5L19 4" />
                    </svg>
                    <span className="font-medium text-base">{key.title}</span>
                    {key.key_type && (
                      <span className="text-xs px-2 py-0.5 rounded-full bg-base border border-base text-muted">
                        {key.key_type}
                      </span>
                    )}
                  </div>

                  <div className="mt-1 text-sm text-muted font-mono truncate">
                    {key.fingerprint}
                  </div>

                  <div className="mt-2 flex items-center gap-4 text-xs text-muted">
                    <span>Added {formatDate(key.created_at)}</span>
                    {key.last_used_at && (
                      <span>Last used {formatDate(key.last_used_at)}</span>
                    )}
                  </div>
                </div>

                <button
                  onClick={() => setKeyToDelete(key)}
                  className="ml-4 p-2 text-muted hover:text-red-500 hover:bg-red-50 dark:hover:bg-red-900/20 rounded-md transition-colors"
                  title="Delete SSH key"
                >
                  <svg
                    xmlns="http://www.w3.org/2000/svg"
                    width="16"
                    height="16"
                    viewBox="0 0 24 24"
                    fill="none"
                    stroke="currentColor"
                    strokeWidth="2"
                    strokeLinecap="round"
                    strokeLinejoin="round"
                  >
                    <path d="M3 6h18" />
                    <path d="M19 6v14c0 1-1 2-2 2H7c-1 0-2-1-2-2V6" />
                    <path d="M8 6V4c0-1 1-2 2-2h4c1 0 2 1 2 2v2" />
                    <line x1="10" y1="11" x2="10" y2="17" />
                    <line x1="14" y1="11" x2="14" y2="17" />
                  </svg>
                </button>
              </li>
            ))}
          </ul>
        )}
      </div>

      {/* Usage Instructions */}
      <div className="border border-base rounded-md bg-panel">
        <div className="px-4 py-3 border-b border-base">
          <h3 className="font-medium text-base">Using SSH with Git</h3>
        </div>
        <div className="p-4 text-sm text-muted space-y-3">
          <p>
            Once you&apos;ve added your SSH key, you can clone repositories using SSH:
          </p>
          <code className="block bg-base border border-base rounded-md p-3 text-accent font-mono text-xs overflow-x-auto">
            git clone ssh://git@localhost:2222/username/repo.git
          </code>
          <p>
            You can also configure your SSH client by adding to{" "}
            <code className="bg-base px-1 py-0.5 rounded">~/.ssh/config</code>:
          </p>
          <pre className="block bg-base border border-base rounded-md p-3 text-accent font-mono text-xs overflow-x-auto">
{`Host git-server
  HostName localhost
  Port 2222
  User git
  IdentityFile ~/.ssh/id_ed25519`}
          </pre>
        </div>
      </div>

      {/* Delete Confirmation Modal */}
      {keyToDelete && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
          <div className="bg-panel border border-base rounded-lg shadow-xl max-w-md w-full mx-4 p-6">
            <h3 className="text-lg font-semibold text-base mb-4">
              Delete SSH Key?
            </h3>
            <p className="text-sm text-muted mb-4">
              Are you sure you want to delete the SSH key{" "}
              <strong>&quot;{keyToDelete.title}&quot;</strong>? You will no longer be able
              to use this key to authenticate.
            </p>

            <div className="flex justify-end gap-3">
              <button
                type="button"
                onClick={() => setKeyToDelete(null)}
                className="px-4 py-2 border border-base rounded-md text-sm font-medium text-base hover:bg-base focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-accent"
              >
                Cancel
              </button>
              <button
                type="button"
                onClick={handleDeleteKey}
                disabled={deletingKeyId === keyToDelete.id}
                className="px-4 py-2 border border-transparent rounded-md text-sm font-medium text-white bg-red-600 hover:bg-red-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-red-500 disabled:opacity-50 disabled:cursor-not-allowed"
              >
                {deletingKeyId === keyToDelete.id ? "Deleting..." : "Delete Key"}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
