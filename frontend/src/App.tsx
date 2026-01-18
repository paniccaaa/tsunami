import { useState, useEffect, useCallback, useRef } from 'react';
import { AttackForm } from './components/AttackForm';
import { MetricsDashboard } from './components/MetricsDashboard';
import { ResultsView } from './components/ResultsView';
import { startAttack, stopAttack, getStatus, getResults } from './services/api';
import { MetricsWebSocket } from './services/websocket';
import type { AttackConfig, MetricsPayload, ResultsResponse, SessionStatus } from './types/api';

function App() {
  const [status, setStatus] = useState<SessionStatus>('idle');
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [metrics, setMetrics] = useState<MetricsPayload | null>(null);
  const [metricsHistory, setMetricsHistory] = useState<MetricsPayload[]>([]);
  const [results, setResults] = useState<ResultsResponse | null>(null);
  const wsRef = useRef<MetricsWebSocket | null>(null);

  const handleMetricsUpdate = useCallback((data: MetricsPayload) => {
    setMetrics(data);
    setMetricsHistory((prev) => [...prev.slice(-120), data]); // Keep last 60 seconds (120 * 500ms)
  }, []);

  const handleStatusUpdate = useCallback((type: string) => {
    if (type === 'completed') {
      setStatus('completed');
      fetchResults();
    } else if (type === 'stopped') {
      setStatus('stopped');
      fetchResults();
    } else if (type === 'error') {
      setStatus('error');
    }
  }, []);

  const fetchResults = async () => {
    try {
      const data = await getResults();
      setResults(data);
    } catch (err) {
      console.error('Failed to fetch results:', err);
    }
  };

  useEffect(() => {
    // Check initial status
    getStatus().then((data) => {
      setStatus(data.status as SessionStatus);
      if (data.metrics) {
        setMetrics(data.metrics);
      }
    }).catch(console.error);
  }, []);

  useEffect(() => {
    // Connect WebSocket when running
    if (status === 'running') {
      wsRef.current = new MetricsWebSocket(handleMetricsUpdate, handleStatusUpdate);
      wsRef.current.connect();
    }

    return () => {
      wsRef.current?.disconnect();
    };
  }, [status, handleMetricsUpdate, handleStatusUpdate]);

  const handleStart = async (config: AttackConfig) => {
    setIsLoading(true);
    setError(null);
    setResults(null);
    setMetrics(null);
    setMetricsHistory([]);

    try {
      await startAttack(config);
      setStatus('running');
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to start attack');
    } finally {
      setIsLoading(false);
    }
  };

  const handleStop = async () => {
    setIsLoading(true);
    setError(null);

    try {
      await stopAttack();
      setStatus('stopped');
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to stop attack');
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <div className="min-h-screen bg-gray-900">
      <header className="bg-gray-800 border-b border-gray-700">
        <div className="max-w-7xl mx-auto px-4 py-4">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-3">
              <svg className="w-8 h-8 text-blue-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 10V3L4 14h7v7l9-11h-7z" />
              </svg>
              <h1 className="text-2xl font-bold text-white">Tsunami</h1>
            </div>
            <div className="text-sm text-gray-400">
              HTTP Load Testing Tool
            </div>
          </div>
        </div>
      </header>

      <main className="max-w-7xl mx-auto px-4 py-8 space-y-8">
        {error && (
          <div className="bg-red-900/50 border border-red-500 text-red-200 px-4 py-3 rounded-lg">
            <div className="flex items-center gap-2">
              <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
              </svg>
              <span>{error}</span>
            </div>
          </div>
        )}

        <AttackForm
          onStart={handleStart}
          onStop={handleStop}
          isRunning={status === 'running'}
          isLoading={isLoading}
        />

        <MetricsDashboard
          metrics={metrics}
          metricsHistory={metricsHistory}
          isRunning={status === 'running'}
        />

        {(status === 'completed' || status === 'stopped') && (
          <ResultsView results={results} />
        )}
      </main>

      <footer className="bg-gray-800 border-t border-gray-700 mt-auto">
        <div className="max-w-7xl mx-auto px-4 py-4">
          <div className="text-center text-sm text-gray-500">
            Tsunami HTTP Load Testing Tool
          </div>
        </div>
      </footer>
    </div>
  );
}

export default App;
