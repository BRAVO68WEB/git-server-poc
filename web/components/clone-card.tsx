'use client';

import { useState } from 'react';

export default function CloneCard({ httpUrl, sshUrl }: { httpUrl: string; sshUrl: string }) {
  const [method, setMethod] = useState<'http' | 'ssh'>('http');
  const [copied, setCopied] = useState(false);

  const url = method === 'http' ? httpUrl : sshUrl;

  const handleCopy = () => {
    navigator.clipboard.writeText(url);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  return (
    <div className="bg-white p-4 rounded-lg shadow-sm border border-gray-200 mb-6">
      <div className="flex items-center justify-between mb-2">
        <h3 className="font-semibold text-gray-900">Clone this repository</h3>
        <div className="flex space-x-2 text-sm">
          <button
            onClick={() => setMethod('http')}
            className={`px-3 py-1 rounded-full ${
              method === 'http' ? 'bg-blue-100 text-blue-700' : 'text-gray-600 hover:bg-gray-100'
            }`}
          >
            HTTP
          </button>
          <button
            onClick={() => setMethod('ssh')}
            className={`px-3 py-1 rounded-full ${
              method === 'ssh' ? 'bg-blue-100 text-blue-700' : 'text-gray-600 hover:bg-gray-100'
            }`}
          >
            SSH
          </button>
        </div>
      </div>
      <div className="flex">
        <input
          type="text"
          readOnly
          value={url}
          className="flex-1 p-2 border border-gray-300 rounded-l-md bg-gray-50 text-sm font-mono text-gray-600 focus:outline-none"
        />
        <button
          onClick={handleCopy}
          className="px-4 py-2 bg-gray-100 border border-l-0 border-gray-300 rounded-r-md hover:bg-gray-200 text-sm font-medium text-gray-700 transition-colors"
        >
          {copied ? 'Copied!' : 'Copy'}
        </button>
      </div>
    </div>
  );
}
