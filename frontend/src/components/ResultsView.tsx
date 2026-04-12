import { useTranslation } from 'react-i18next';
import type { ResultsResponse } from '../types/api';
import { downloadResults } from '../services/api';

const GRPC_STATUS_NAMES: Record<number, string> = {
  0: 'OK', 1: 'CANCELLED', 2: 'UNKNOWN', 3: 'INVALID_ARGUMENT',
  4: 'DEADLINE_EXCEEDED', 5: 'NOT_FOUND', 6: 'ALREADY_EXISTS',
  7: 'PERMISSION_DENIED', 8: 'RESOURCE_EXHAUSTED', 9: 'FAILED_PRECONDITION',
  10: 'ABORTED', 11: 'OUT_OF_RANGE', 12: 'UNIMPLEMENTED', 13: 'INTERNAL',
  14: 'UNAVAILABLE', 15: 'DATA_LOSS', 16: 'UNAUTHENTICATED',
};

interface ResultsViewProps {
  results: ResultsResponse | null;
  protocol?: 'http' | 'grpc';
}

export function ResultsView({ results, protocol }: ResultsViewProps) {
  const isGRPC = protocol === 'grpc';
  const { t } = useTranslation();

  if (!results) {
    return null;
  }

  const handleDownload = () => {
    downloadResults();
  };

  return (
    <div className="bg-gray-800 rounded-lg p-6 space-y-6">
      <div className="flex items-center justify-between">
        <h2 className="text-xl font-bold text-white">{t('results.title')}</h2>
        <button
          onClick={handleDownload}
          className="px-4 py-2 bg-green-600 text-white rounded-md hover:bg-green-500 focus:outline-none focus:ring-2 focus:ring-green-500 transition-colors flex items-center gap-2"
        >
          <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
          </svg>
          {t('results.downloadJson')}
        </button>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        <div className="space-y-4">
          <h3 className="text-lg font-semibold text-white">{t('results.configuration')}</h3>
          <div className="bg-gray-700 rounded-lg p-4 space-y-2 text-sm">
            <div className="flex justify-between">
              <span className="text-gray-400">{t('results.url')}</span>
              <span className="text-white font-mono">{results.config.url}</span>
            </div>
            <div className="flex justify-between">
              <span className="text-gray-400">{t('results.method')}</span>
              <span className="text-white">{results.config.method}</span>
            </div>
            <div className="flex justify-between">
              <span className="text-gray-400">{t('results.rate')}</span>
              <span className="text-white">{results.config.rate}</span>
            </div>
            <div className="flex justify-between">
              <span className="text-gray-400">{t('results.duration')}</span>
              <span className="text-white">{results.config.duration}</span>
            </div>
            <div className="flex justify-between">
              <span className="text-gray-400">{t('results.workers')}</span>
              <span className="text-white">{results.config.workers}</span>
            </div>
            <div className="flex justify-between">
              <span className="text-gray-400">{t('results.connections')}</span>
              <span className="text-white">{results.config.connections}</span>
            </div>
          </div>
        </div>

        <div className="space-y-4">
          <h3 className="text-lg font-semibold text-white">{t('results.summary')}</h3>
          <div className="bg-gray-700 rounded-lg p-4 space-y-2 text-sm">
            <div className="flex justify-between">
              <span className="text-gray-400">{t('results.totalRequests')}</span>
              <span className="text-white font-bold">{results.summary.total_requests.toLocaleString()}</span>
            </div>
            <div className="flex justify-between">
              <span className="text-gray-400">{t('results.successful')}</span>
              <span className="text-green-400 font-bold">{results.summary.successful_requests.toLocaleString()}</span>
            </div>
            <div className="flex justify-between">
              <span className="text-gray-400">{t('results.failed')}</span>
              <span className="text-red-400 font-bold">{results.summary.failed_requests.toLocaleString()}</span>
            </div>
            <div className="flex justify-between">
              <span className="text-gray-400">{t('results.elapsedTime')}</span>
              <span className="text-white">{results.summary.total_elapsed_time}</span>
            </div>
            <div className="flex justify-between">
              <span className="text-gray-400">{t('results.avgLatency')}</span>
              <span className="text-white">{results.summary.average_latency}</span>
            </div>
            <div className="flex justify-between">
              <span className="text-gray-400">{t('results.throughput')}</span>
              <span className="text-blue-400 font-bold">{results.summary.throughput_rps.toFixed(2)} req/s</span>
            </div>
          </div>
        </div>

        {results.latency_percentiles && (
          <div className="space-y-4">
            <h3 className="text-lg font-semibold text-white">{t('results.latencyPercentiles')}</h3>
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
          <h3 className="text-lg font-semibold text-white">{t('results.statusCodes')}</h3>
          <div className="bg-gray-700 rounded-lg p-4 space-y-2 text-sm">
            {Object.entries(results.status_codes).map(([code, count]) => {
              const percentage = ((count / results.summary.total_requests) * 100).toFixed(1);
              const num = Number(code);

              let label: string;
              let colorClass: string;

              if (isGRPC) {
                label = `${GRPC_STATUS_NAMES[num] ?? `CODE_${num}`} (${num})`;
                colorClass = num === 0 ? 'text-green-400' : num === 4 || num === 14 ? 'text-red-400' : 'text-yellow-400';
              } else {
                const isSuccess = num >= 200 && num < 300;
                const isError = num >= 400 || num === 0;
                label = num === 0 ? t('results.errorLabel') : `HTTP ${code}`;
                colorClass = isSuccess ? 'text-green-400' : isError ? 'text-red-400' : 'text-yellow-400';
              }

              return (
                <div key={code} className="flex justify-between items-center">
                  <span className="text-gray-400">{label}:</span>
                  <span className={`font-bold ${colorClass}`}>
                    {count.toLocaleString()} ({percentage}%)
                  </span>
                </div>
              );
            })}
          </div>
        </div>
      </div>

      <div className="text-sm text-gray-500 text-right">
        {t('results.completedAt')} {new Date(results.timestamp).toLocaleString()}
      </div>
    </div>
  );
}
