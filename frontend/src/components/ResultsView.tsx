import type { ResultsResponse } from '../types/api';
import { downloadResults } from '../services/api';

interface ResultsViewProps {
  results: ResultsResponse | null;
}

export function ResultsView({ results }: ResultsViewProps) {
  if (!results) {
    return null;
  }

  const handleDownload = () => {
    downloadResults();
  };

  return (
    <div className="bg-gray-800 rounded-lg p-6 space-y-6">
      <div className="flex items-center justify-between">
        <h2 className="text-xl font-bold text-white">Test Results</h2>
        <button
          onClick={handleDownload}
          className="px-4 py-2 bg-green-600 text-white rounded-md hover:bg-green-500 focus:outline-none focus:ring-2 focus:ring-green-500 transition-colors flex items-center gap-2"
        >
          <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
          </svg>
          Download JSON
        </button>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        <div className="space-y-4">
          <h3 className="text-lg font-semibold text-white">Configuration</h3>
          <div className="bg-gray-700 rounded-lg p-4 space-y-2 text-sm">
            <div className="flex justify-between">
              <span className="text-gray-400">URL:</span>
              <span className="text-white font-mono">{results.config.url}</span>
            </div>
            <div className="flex justify-between">
              <span className="text-gray-400">Method:</span>
              <span className="text-white">{results.config.method}</span>
            </div>
            <div className="flex justify-between">
              <span className="text-gray-400">Rate:</span>
              <span className="text-white">{results.config.rate}</span>
            </div>
            <div className="flex justify-between">
              <span className="text-gray-400">Duration:</span>
              <span className="text-white">{results.config.duration}</span>
            </div>
            <div className="flex justify-between">
              <span className="text-gray-400">Workers:</span>
              <span className="text-white">{results.config.workers}</span>
            </div>
            <div className="flex justify-between">
              <span className="text-gray-400">Connections:</span>
              <span className="text-white">{results.config.connections}</span>
            </div>
          </div>
        </div>

        <div className="space-y-4">
          <h3 className="text-lg font-semibold text-white">Summary</h3>
          <div className="bg-gray-700 rounded-lg p-4 space-y-2 text-sm">
            <div className="flex justify-between">
              <span className="text-gray-400">Total Requests:</span>
              <span className="text-white font-bold">{results.summary.total_requests.toLocaleString()}</span>
            </div>
            <div className="flex justify-between">
              <span className="text-gray-400">Successful:</span>
              <span className="text-green-400 font-bold">{results.summary.successful_requests.toLocaleString()}</span>
            </div>
            <div className="flex justify-between">
              <span className="text-gray-400">Failed:</span>
              <span className="text-red-400 font-bold">{results.summary.failed_requests.toLocaleString()}</span>
            </div>
            <div className="flex justify-between">
              <span className="text-gray-400">Elapsed Time:</span>
              <span className="text-white">{results.summary.total_elapsed_time}</span>
            </div>
            <div className="flex justify-between">
              <span className="text-gray-400">Avg Latency:</span>
              <span className="text-white">{results.summary.average_latency}</span>
            </div>
            <div className="flex justify-between">
              <span className="text-gray-400">Throughput:</span>
              <span className="text-blue-400 font-bold">{results.summary.throughput_rps.toFixed(2)} req/s</span>
            </div>
          </div>
        </div>

        {results.latency_percentiles && (
          <div className="space-y-4">
            <h3 className="text-lg font-semibold text-white">Latency Percentiles</h3>
            <div className="bg-gray-700 rounded-lg p-4 space-y-2 text-sm">
              <div className="flex justify-between">
                <span className="text-gray-400">P50:</span>
                <span className="text-white">{results.latency_percentiles.p50}</span>
              </div>
              <div className="flex justify-between">
                <span className="text-gray-400">P90:</span>
                <span className="text-white">{results.latency_percentiles.p90}</span>
              </div>
              <div className="flex justify-between">
                <span className="text-gray-400">P95:</span>
                <span className="text-white">{results.latency_percentiles.p95}</span>
              </div>
              <div className="flex justify-between">
                <span className="text-gray-400">P99:</span>
                <span className="text-white">{results.latency_percentiles.p99}</span>
              </div>
            </div>
          </div>
        )}

        <div className="space-y-4">
          <h3 className="text-lg font-semibold text-white">Status Codes</h3>
          <div className="bg-gray-700 rounded-lg p-4 space-y-2 text-sm">
            {Object.entries(results.status_codes).map(([code, count]) => {
              const percentage = ((count / results.summary.total_requests) * 100).toFixed(1);
              const isSuccess = Number(code) >= 200 && Number(code) < 300;
              const isError = Number(code) >= 400 || code === '0';
              return (
                <div key={code} className="flex justify-between items-center">
                  <span className="text-gray-400">
                    {code === '0' ? 'Error (timeout/connection)' : `HTTP ${code}`}:
                  </span>
                  <span className={`font-bold ${isSuccess ? 'text-green-400' : isError ? 'text-red-400' : 'text-yellow-400'}`}>
                    {count.toLocaleString()} ({percentage}%)
                  </span>
                </div>
              );
            })}
          </div>
        </div>
      </div>

      <div className="text-sm text-gray-500 text-right">
        Test completed at: {new Date(results.timestamp).toLocaleString()}
      </div>
    </div>
  );
}
