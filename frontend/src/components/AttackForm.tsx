import { useState } from 'react';
import type { AttackConfig } from '../types/api';

interface AttackFormProps {
  onStart: (config: AttackConfig) => void;
  onStop: () => void;
  isRunning: boolean;
  isLoading: boolean;
}

const HTTP_METHODS = ['GET', 'POST', 'PUT', 'DELETE', 'PATCH'];

export function AttackForm({ onStart, onStop, isRunning, isLoading }: AttackFormProps) {
  const [url, setUrl] = useState('');
  const [method, setMethod] = useState('GET');
  const [body, setBody] = useState('');
  const [headers, setHeaders] = useState<string[]>([]);
  const [newHeader, setNewHeader] = useState('');
  const [rate, setRate] = useState('100/1s');
  const [duration, setDuration] = useState('30s');
  const [timeout, setTimeout] = useState('10s');
  const [workers, setWorkers] = useState(50);
  const [connections, setConnections] = useState(100);

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    onStart({
      url,
      method,
      body: body || undefined,
      headers: headers.length > 0 ? headers : undefined,
      rate,
      duration,
      timeout,
      workers,
      connections,
    });
  };

  const addHeader = () => {
    if (newHeader.includes(':')) {
      setHeaders([...headers, newHeader]);
      setNewHeader('');
    }
  };

  const removeHeader = (index: number) => {
    setHeaders(headers.filter((_, i) => i !== index));
  };

  return (
    <form onSubmit={handleSubmit} className="bg-gray-800 rounded-lg p-6 space-y-4">
      <h2 className="text-xl font-bold text-white mb-4">Attack Configuration</h2>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        <div className="md:col-span-2">
          <label className="block text-sm font-medium text-gray-300 mb-1">Target URL *</label>
          <input
            type="url"
            value={url}
            onChange={(e) => setUrl(e.target.value)}
            placeholder="https://example.com/api"
            required
            disabled={isRunning}
            className="w-full px-3 py-2 bg-gray-700 border border-gray-600 rounded-md text-white placeholder-gray-400 focus:outline-none focus:ring-2 focus:ring-blue-500 disabled:opacity-50"
          />
        </div>

        <div>
          <label className="block text-sm font-medium text-gray-300 mb-1">HTTP Method</label>
          <select
            value={method}
            onChange={(e) => setMethod(e.target.value)}
            disabled={isRunning}
            className="w-full px-3 py-2 bg-gray-700 border border-gray-600 rounded-md text-white focus:outline-none focus:ring-2 focus:ring-blue-500 disabled:opacity-50"
          >
            {HTTP_METHODS.map((m) => (
              <option key={m} value={m}>{m}</option>
            ))}
          </select>
        </div>

        <div>
          <label className="block text-sm font-medium text-gray-300 mb-1">Rate (requests/time)</label>
          <input
            type="text"
            value={rate}
            onChange={(e) => setRate(e.target.value)}
            placeholder="100/1s"
            disabled={isRunning}
            className="w-full px-3 py-2 bg-gray-700 border border-gray-600 rounded-md text-white placeholder-gray-400 focus:outline-none focus:ring-2 focus:ring-blue-500 disabled:opacity-50"
          />
        </div>

        <div>
          <label className="block text-sm font-medium text-gray-300 mb-1">Duration</label>
          <input
            type="text"
            value={duration}
            onChange={(e) => setDuration(e.target.value)}
            placeholder="30s"
            disabled={isRunning}
            className="w-full px-3 py-2 bg-gray-700 border border-gray-600 rounded-md text-white placeholder-gray-400 focus:outline-none focus:ring-2 focus:ring-blue-500 disabled:opacity-50"
          />
        </div>

        <div>
          <label className="block text-sm font-medium text-gray-300 mb-1">Timeout</label>
          <input
            type="text"
            value={timeout}
            onChange={(e) => setTimeout(e.target.value)}
            placeholder="10s"
            disabled={isRunning}
            className="w-full px-3 py-2 bg-gray-700 border border-gray-600 rounded-md text-white placeholder-gray-400 focus:outline-none focus:ring-2 focus:ring-blue-500 disabled:opacity-50"
          />
        </div>

        <div>
          <label className="block text-sm font-medium text-gray-300 mb-1">Workers</label>
          <input
            type="number"
            value={workers}
            onChange={(e) => setWorkers(Number(e.target.value))}
            min="1"
            disabled={isRunning}
            className="w-full px-3 py-2 bg-gray-700 border border-gray-600 rounded-md text-white focus:outline-none focus:ring-2 focus:ring-blue-500 disabled:opacity-50"
          />
        </div>

        <div>
          <label className="block text-sm font-medium text-gray-300 mb-1">Connections</label>
          <input
            type="number"
            value={connections}
            onChange={(e) => setConnections(Number(e.target.value))}
            min="1"
            disabled={isRunning}
            className="w-full px-3 py-2 bg-gray-700 border border-gray-600 rounded-md text-white focus:outline-none focus:ring-2 focus:ring-blue-500 disabled:opacity-50"
          />
        </div>

        {(method === 'POST' || method === 'PUT' || method === 'PATCH') && (
          <div className="md:col-span-2">
            <label className="block text-sm font-medium text-gray-300 mb-1">Request Body</label>
            <textarea
              value={body}
              onChange={(e) => setBody(e.target.value)}
              placeholder='{"key": "value"}'
              rows={3}
              disabled={isRunning}
              className="w-full px-3 py-2 bg-gray-700 border border-gray-600 rounded-md text-white placeholder-gray-400 focus:outline-none focus:ring-2 focus:ring-blue-500 disabled:opacity-50 font-mono text-sm"
            />
          </div>
        )}

        <div className="md:col-span-2">
          <label className="block text-sm font-medium text-gray-300 mb-1">Headers</label>
          <div className="flex gap-2 mb-2">
            <input
              type="text"
              value={newHeader}
              onChange={(e) => setNewHeader(e.target.value)}
              placeholder="Header-Name: value"
              disabled={isRunning}
              className="flex-1 px-3 py-2 bg-gray-700 border border-gray-600 rounded-md text-white placeholder-gray-400 focus:outline-none focus:ring-2 focus:ring-blue-500 disabled:opacity-50"
              onKeyDown={(e) => e.key === 'Enter' && (e.preventDefault(), addHeader())}
            />
            <button
              type="button"
              onClick={addHeader}
              disabled={isRunning || !newHeader.includes(':')}
              className="px-4 py-2 bg-gray-600 text-white rounded-md hover:bg-gray-500 disabled:opacity-50 disabled:cursor-not-allowed"
            >
              Add
            </button>
          </div>
          {headers.length > 0 && (
            <div className="space-y-1">
              {headers.map((header, index) => (
                <div key={index} className="flex items-center gap-2 text-sm">
                  <code className="flex-1 px-2 py-1 bg-gray-700 rounded text-gray-300">{header}</code>
                  <button
                    type="button"
                    onClick={() => removeHeader(index)}
                    disabled={isRunning}
                    className="text-red-400 hover:text-red-300 disabled:opacity-50"
                  >
                    Remove
                  </button>
                </div>
              ))}
            </div>
          )}
        </div>
      </div>

      <div className="flex gap-4 pt-4">
        {!isRunning ? (
          <button
            type="submit"
            disabled={isLoading || !url}
            className="flex-1 px-6 py-3 bg-blue-600 text-white font-semibold rounded-md hover:bg-blue-500 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 focus:ring-offset-gray-800 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
          >
            {isLoading ? 'Starting...' : 'Start Attack'}
          </button>
        ) : (
          <button
            type="button"
            onClick={onStop}
            disabled={isLoading}
            className="flex-1 px-6 py-3 bg-red-600 text-white font-semibold rounded-md hover:bg-red-500 focus:outline-none focus:ring-2 focus:ring-red-500 focus:ring-offset-2 focus:ring-offset-gray-800 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
          >
            {isLoading ? 'Stopping...' : 'Stop Attack'}
          </button>
        )}
      </div>
    </form>
  );
}
