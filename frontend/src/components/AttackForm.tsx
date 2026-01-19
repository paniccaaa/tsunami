import { useState, useMemo } from 'react';
import type { AttackConfig } from '../types/api';

interface AttackFormProps {
  onStart: (config: AttackConfig) => void;
  onStop: () => void;
  isRunning: boolean;
  isLoading: boolean;
}

const HTTP_METHODS = ['GET', 'POST', 'PUT', 'DELETE', 'PATCH'];

// Validation patterns
const RATE_REGEX = /^\d+\/\d+(ms|s|m|h)$/;
const DURATION_REGEX = /^(\d+)(ms|s|m|h)$/;

// Tooltip component
function Tooltip({ text }: { text: string }) {
  const [show, setShow] = useState(false);

  return (
    <div className="relative inline-block ml-1">
      <button
        type="button"
        className="w-4 h-4 rounded-full bg-gray-600 text-gray-300 text-xs hover:bg-gray-500 focus:outline-none"
        onMouseEnter={() => setShow(true)}
        onMouseLeave={() => setShow(false)}
        onClick={() => setShow(!show)}
      >
        ?
      </button>
      {show && (
        <div className="absolute z-10 w-64 p-2 mt-1 text-sm text-gray-200 bg-gray-700 rounded-md shadow-lg -left-28 top-6">
          {text}
        </div>
      )}
    </div>
  );
}

// Label with tooltip
function Label({ text, tooltip }: { text: string; tooltip: string }) {
  return (
    <label className="flex items-center text-sm font-medium text-gray-300 mb-1">
      {text}
      <Tooltip text={tooltip} />
    </label>
  );
}

// Validation error message
function ValidationError({ message }: { message: string | null }) {
  if (!message) return null;
  return <p className="text-red-400 text-xs mt-1">{message}</p>;
}

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

  // Validation
  const errors = useMemo(() => {
    const errs: Record<string, string | null> = {
      url: null,
      rate: null,
      duration: null,
      timeout: null,
      workers: null,
      connections: null,
    };

    // URL validation
    if (url && !url.startsWith('http://') && !url.startsWith('https://')) {
      errs.url = 'URL must start with http:// or https://';
    }

    // Rate validation
    if (rate && !RATE_REGEX.test(rate)) {
      errs.rate = 'Format: NUMBER/NUMBERunit (e.g., 100/1s, 50/500ms)';
    }

    // Duration validation
    if (duration && !DURATION_REGEX.test(duration)) {
      errs.duration = 'Format: NUMBERunit (e.g., 30s, 5m, 1h)';
    }

    // Timeout validation
    if (timeout && !DURATION_REGEX.test(timeout)) {
      errs.timeout = 'Format: NUMBERunit (e.g., 10s, 1m)';
    }

    // Workers validation
    if (workers !== null && workers < 1) {
      errs.workers = 'Must be at least 1';
    }

    // Connections validation
    if (connections !== null && connections < 1) {
      errs.connections = 'Must be at least 1';
    }

    return errs;
  }, [url, rate, duration, timeout, workers, connections]);

  const hasErrors = Object.values(errors).some((e) => e !== null);
  const isFormValid = url && !hasErrors;

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!isFormValid) return;

    onStart({
      url,
      method,
      body: body || undefined,
      headers: headers.length > 0 ? headers : undefined,
      rate,
      duration,
      timeout,
      workers: workers || 50,
      connections: connections || 100,
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
          <Label
            text="Target URL *"
            tooltip="The HTTP(S) endpoint to test. Must include protocol (http:// or https://)."
          />
          <input
            type="url"
            value={url}
            onChange={(e) => setUrl(e.target.value)}
            placeholder="https://example.com/api"
            required
            disabled={isRunning}
            className={`w-full px-3 py-2 bg-gray-700 border rounded-md text-white placeholder-gray-400 focus:outline-none focus:ring-2 focus:ring-blue-500 disabled:opacity-50 ${
              errors.url ? 'border-red-500' : 'border-gray-600'
            }`}
          />
          <ValidationError message={errors.url} />
        </div>

        <div>
          <Label
            text="HTTP Method"
            tooltip="HTTP method to use for requests. GET for fetching data, POST/PUT for sending data."
          />
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
          <Label
            text="Rate"
            tooltip="Request rate: REQUESTS/TIME. Examples: 100/1s (100 per second), 1000/1m (1000 per minute), 50/500ms (50 per 500ms). Units: ms, s, m, h."
          />
          <input
            type="text"
            value={rate}
            onChange={(e) => setRate(e.target.value)}
            placeholder="100/1s"
            disabled={isRunning}
            className={`w-full px-3 py-2 bg-gray-700 border rounded-md text-white placeholder-gray-400 focus:outline-none focus:ring-2 focus:ring-blue-500 disabled:opacity-50 ${
              errors.rate ? 'border-red-500' : 'border-gray-600'
            }`}
          />
          <ValidationError message={errors.rate} />
        </div>

        <div>
          <Label
            text="Duration"
            tooltip="How long to run the test. Use 0 for infinite (stop manually). Examples: 30s, 5m, 1h. Units: ms, s, m, h."
          />
          <input
            type="text"
            value={duration}
            onChange={(e) => setDuration(e.target.value)}
            placeholder="30s"
            disabled={isRunning}
            className={`w-full px-3 py-2 bg-gray-700 border rounded-md text-white placeholder-gray-400 focus:outline-none focus:ring-2 focus:ring-blue-500 disabled:opacity-50 ${
              errors.duration ? 'border-red-500' : 'border-gray-600'
            }`}
          />
          <ValidationError message={errors.duration} />
        </div>

        <div>
          <Label
            text="Timeout"
            tooltip="Maximum time to wait for each request response. If exceeded, request is marked as failed. Examples: 10s, 30s, 1m."
          />
          <input
            type="text"
            value={timeout}
            onChange={(e) => setTimeout(e.target.value)}
            placeholder="10s"
            disabled={isRunning}
            className={`w-full px-3 py-2 bg-gray-700 border rounded-md text-white placeholder-gray-400 focus:outline-none focus:ring-2 focus:ring-blue-500 disabled:opacity-50 ${
              errors.timeout ? 'border-red-500' : 'border-gray-600'
            }`}
          />
          <ValidationError message={errors.timeout} />
        </div>

        <div>
          <Label
            text="Workers"
            tooltip="Number of concurrent goroutines sending requests. More workers = more parallel requests. Recommended: 10-100 for most tests."
          />
          <input
            type="number"
            value={workers || ''}
            onChange={(e) => setWorkers(e.target.value === '' ? 0 : parseInt(e.target.value, 10) || 0)}
            min="1"
            placeholder="50"
            disabled={isRunning}
            className={`w-full px-3 py-2 bg-gray-700 border rounded-md text-white focus:outline-none focus:ring-2 focus:ring-blue-500 disabled:opacity-50 ${
              errors.workers ? 'border-red-500' : 'border-gray-600'
            }`}
          />
          <ValidationError message={errors.workers} />
        </div>

        <div>
          <Label
            text="Connections"
            tooltip="Maximum number of TCP connections to keep open. Higher values allow more parallel requests but use more resources. Recommended: equal to or higher than workers."
          />
          <input
            type="number"
            value={connections || ''}
            onChange={(e) => setConnections(e.target.value === '' ? 0 : parseInt(e.target.value, 10) || 0)}
            min="1"
            placeholder="100"
            disabled={isRunning}
            className={`w-full px-3 py-2 bg-gray-700 border rounded-md text-white focus:outline-none focus:ring-2 focus:ring-blue-500 disabled:opacity-50 ${
              errors.connections ? 'border-red-500' : 'border-gray-600'
            }`}
          />
          <ValidationError message={errors.connections} />
        </div>

        {(method === 'POST' || method === 'PUT' || method === 'PATCH') && (
          <div className="md:col-span-2">
            <Label
              text="Request Body"
              tooltip="JSON or text body to send with each request. Leave empty for no body. Content-Type defaults to application/json."
            />
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
          <Label
            text="Headers"
            tooltip="Custom HTTP headers to send with each request. Format: Header-Name: value. Press Enter or click Add to add a header."
          />
          <div className="flex gap-2 mb-2">
            <input
              type="text"
              value={newHeader}
              onChange={(e) => setNewHeader(e.target.value)}
              placeholder="Authorization: Bearer token"
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
            disabled={isLoading || !isFormValid}
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
