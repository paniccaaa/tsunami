import { useMemo } from 'react';
import {
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  PieChart,
  Pie,
  Cell,
  BarChart,
  Bar,
  AreaChart,
  Area,
} from 'recharts';
import type { MetricsPayload } from '../types/api';

interface MetricsDashboardProps {
  metrics: MetricsPayload | null;
  metricsHistory: MetricsPayload[];
  isRunning: boolean;
}

const STATUS_COLORS: Record<string, string> = {
  '200': '#22c55e',
  '201': '#22c55e',
  '204': '#22c55e',
  '301': '#eab308',
  '302': '#eab308',
  '400': '#f97316',
  '401': '#f97316',
  '403': '#f97316',
  '404': '#f97316',
  '500': '#ef4444',
  '502': '#ef4444',
  '503': '#ef4444',
  '0': '#6b7280',
};

const ERROR_COLORS: Record<string, string> = {
  timeout: '#ef4444',
  connection_refused: '#f97316',
  dns: '#eab308',
  tls: '#a855f7',
  other: '#6b7280',
};

function getStatusColor(code: number): string {
  return STATUS_COLORS[String(code)] || (code < 300 ? '#22c55e' : code < 400 ? '#eab308' : code < 500 ? '#f97316' : '#ef4444');
}

function formatBytes(bytes: number): string {
  if (bytes >= 1024 * 1024 * 1024) {
    return `${(bytes / (1024 * 1024 * 1024)).toFixed(2)} GB`;
  } else if (bytes >= 1024 * 1024) {
    return `${(bytes / (1024 * 1024)).toFixed(2)} MB`;
  } else if (bytes >= 1024) {
    return `${(bytes / 1024).toFixed(2)} KB`;
  }
  return `${bytes} B`;
}

function MetricCard({ label, value, subvalue, highlight }: { label: string; value: string | number; subvalue?: string; highlight?: 'green' | 'red' | 'yellow' }) {
  const highlightClass = highlight === 'green' ? 'text-green-400' : highlight === 'red' ? 'text-red-400' : highlight === 'yellow' ? 'text-yellow-400' : 'text-white';
  return (
    <div className="bg-gray-800 rounded-lg p-4">
      <div className="text-sm text-gray-400 mb-1">{label}</div>
      <div className={`text-2xl font-bold ${highlightClass}`}>{value}</div>
      {subvalue && <div className="text-sm text-gray-500">{subvalue}</div>}
    </div>
  );
}

function ProgressBar({ progress, elapsed, duration }: { progress: number; elapsed: string; duration: string }) {
  const isInfinite = progress < 0;
  const displayProgress = isInfinite ? 0 : Math.min(progress, 100);

  return (
    <div className="bg-gray-800 rounded-lg p-4">
      <div className="flex justify-between text-sm text-gray-400 mb-2">
        <span>Progress</span>
        <span>{elapsed} {!isInfinite && `/ ${duration}`}</span>
      </div>
      <div className="w-full bg-gray-700 rounded-full h-3">
        {isInfinite ? (
          <div className="bg-blue-500 h-3 rounded-full animate-pulse" style={{ width: '100%' }} />
        ) : (
          <div
            className="bg-blue-500 h-3 rounded-full transition-all duration-500"
            style={{ width: `${displayProgress}%` }}
          />
        )}
      </div>
      <div className="text-center text-sm text-gray-400 mt-1">
        {isInfinite ? 'Infinite duration (stop manually)' : `${displayProgress.toFixed(1)}%`}
      </div>
    </div>
  );
}

export function MetricsDashboard({ metrics, metricsHistory, isRunning }: MetricsDashboardProps) {
  const rpsData = useMemo(() => {
    return metricsHistory.map((m, i) => ({
      time: i * 0.5,
      rps: Math.round(m.current_rps * 100) / 100,
      target: m.target_rps,
    }));
  }, [metricsHistory]);

  const statusCodeData = useMemo(() => {
    if (!metrics?.status_codes) return [];
    return Object.entries(metrics.status_codes).map(([code, count]) => ({
      name: code === '0' ? 'Error' : code,
      value: count,
      color: getStatusColor(Number(code)),
    }));
  }, [metrics?.status_codes]);

  const latencyData = useMemo(() => {
    if (!metrics?.latency_percentiles) return [];
    const parseLatency = (s: string) => {
      const match = s.match(/(\d+)/);
      return match ? parseInt(match[1], 10) : 0;
    };
    return [
      { name: 'Min', value: parseLatency(metrics.min_latency || '0ms') },
      { name: 'P50', value: parseLatency(metrics.latency_percentiles.p50) },
      { name: 'P90', value: parseLatency(metrics.latency_percentiles.p90) },
      { name: 'P95', value: parseLatency(metrics.latency_percentiles.p95) },
      { name: 'P99', value: parseLatency(metrics.latency_percentiles.p99) },
      { name: 'Max', value: parseLatency(metrics.max_latency || '0ms') },
    ];
  }, [metrics?.latency_percentiles, metrics?.min_latency, metrics?.max_latency]);

  const errorData = useMemo(() => {
    if (!metrics?.error_breakdown) return [];
    const breakdown = metrics.error_breakdown;
    const data = [];
    if (breakdown.timeout > 0) data.push({ name: 'Timeout', value: breakdown.timeout, color: ERROR_COLORS.timeout });
    if (breakdown.connection_refused > 0) data.push({ name: 'Conn Refused', value: breakdown.connection_refused, color: ERROR_COLORS.connection_refused });
    if (breakdown.dns > 0) data.push({ name: 'DNS', value: breakdown.dns, color: ERROR_COLORS.dns });
    if (breakdown.tls > 0) data.push({ name: 'TLS', value: breakdown.tls, color: ERROR_COLORS.tls });
    if (breakdown.other > 0) data.push({ name: 'Other', value: breakdown.other, color: ERROR_COLORS.other });
    return data;
  }, [metrics?.error_breakdown]);

  const latencyHistoryData = useMemo(() => {
    if (!metrics?.latency_history || metrics.latency_history.length === 0) return [];
    return metrics.latency_history.map((point) => ({
      time: point.time.toFixed(1),
      latency: point.latency,
    }));
  }, [metrics?.latency_history]);

  // Calculate RPS gap
  const rpsGap = metrics ? metrics.current_rps - metrics.target_rps : 0;
  const rpsGapPercent = metrics && metrics.target_rps > 0 ? (rpsGap / metrics.target_rps) * 100 : 0;
  const rpsGapColor = Math.abs(rpsGapPercent) < 5 ? 'green' : rpsGapPercent < -10 ? 'red' : 'yellow';

  if (!metrics && !isRunning) {
    return (
      <div className="bg-gray-800 rounded-lg p-8 text-center">
        <div className="text-gray-400 text-lg">No metrics available</div>
        <div className="text-gray-500 text-sm mt-2">Start an attack to see real-time metrics</div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h2 className="text-xl font-bold text-white">
          {isRunning ? 'Live Metrics' : 'Metrics'}
        </h2>
        {isRunning && (
          <div className="flex items-center gap-2">
            <div className="w-2 h-2 bg-green-500 rounded-full animate-pulse" />
            <span className="text-sm text-gray-400">Running</span>
          </div>
        )}
      </div>

      {/* Progress Bar */}
      {metrics && (
        <ProgressBar
          progress={metrics.progress}
          elapsed={metrics.elapsed_time || '0s'}
          duration={metrics.duration || '0s'}
        />
      )}

      {/* Main Metrics Cards */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
        <MetricCard
          label="Total Requests"
          value={metrics?.total_requests?.toLocaleString() || 0}
        />
        <MetricCard
          label="Success Rate"
          value={
            metrics?.total_requests
              ? `${((metrics.successes / metrics.total_requests) * 100).toFixed(1)}%`
              : '0%'
          }
          subvalue={`${metrics?.successes || 0} / ${metrics?.failures || 0}`}
          highlight={metrics && metrics.failures > 0 ? 'red' : 'green'}
        />
        <MetricCard
          label="Current RPS"
          value={metrics?.current_rps?.toFixed(1) || '0.0'}
          subvalue={`Target: ${metrics?.target_rps || 0}`}
        />
        <MetricCard
          label="RPS Gap"
          value={`${rpsGap >= 0 ? '+' : ''}${rpsGap.toFixed(1)}`}
          subvalue={`${rpsGapPercent >= 0 ? '+' : ''}${rpsGapPercent.toFixed(1)}%`}
          highlight={rpsGapColor}
        />
      </div>

      {/* Latency Cards */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
        <MetricCard
          label="Avg Latency"
          value={metrics?.average_latency || '0ms'}
        />
        <MetricCard
          label="Min Latency"
          value={metrics?.min_latency || '0ms'}
          highlight="green"
        />
        <MetricCard
          label="Max Latency"
          value={metrics?.max_latency || '0ms'}
          highlight="red"
        />
        <MetricCard
          label="P99 Latency"
          value={metrics?.latency_percentiles?.p99 || '0ms'}
          highlight="yellow"
        />
      </div>

      {/* Data Transfer Cards */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
        <MetricCard
          label="Bytes Sent"
          value={formatBytes(metrics?.bytes_sent || 0)}
        />
        <MetricCard
          label="Bytes Received"
          value={formatBytes(metrics?.bytes_received || 0)}
        />
        <MetricCard
          label="Total Transfer"
          value={formatBytes((metrics?.bytes_sent || 0) + (metrics?.bytes_received || 0))}
        />
        <MetricCard
          label="Bandwidth"
          value={(() => {
            if (!metrics?.elapsed_time) return '0 B/s';
            const match = metrics.elapsed_time.match(/(\d+)/);
            const seconds = match ? parseInt(match[1], 10) : 1;
            const totalBytes = (metrics.bytes_sent || 0) + (metrics.bytes_received || 0);
            return `${formatBytes(totalBytes / Math.max(seconds, 1))}/s`;
          })()}
        />
      </div>

      {/* Charts Row 1 */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <div className="bg-gray-800 rounded-lg p-4">
          <h3 className="text-lg font-semibold text-white mb-4">RPS Over Time (vs Target)</h3>
          <div className="h-64">
            <ResponsiveContainer width="100%" height="100%">
              <LineChart data={rpsData}>
                <CartesianGrid strokeDasharray="3 3" stroke="#374151" />
                <XAxis
                  dataKey="time"
                  stroke="#9ca3af"
                  tickFormatter={(v) => `${v}s`}
                />
                <YAxis stroke="#9ca3af" />
                <Tooltip
                  contentStyle={{ backgroundColor: '#1f2937', border: 'none' }}
                  labelStyle={{ color: '#9ca3af' }}
                />
                <Line
                  type="monotone"
                  dataKey="target"
                  stroke="#6b7280"
                  strokeWidth={1}
                  strokeDasharray="5 5"
                  dot={false}
                  name="Target RPS"
                />
                <Line
                  type="monotone"
                  dataKey="rps"
                  stroke="#3b82f6"
                  strokeWidth={2}
                  dot={false}
                  name="Actual RPS"
                />
              </LineChart>
            </ResponsiveContainer>
          </div>
        </div>

        <div className="bg-gray-800 rounded-lg p-4">
          <h3 className="text-lg font-semibold text-white mb-4">Latency Over Time</h3>
          <div className="h-64">
            {latencyHistoryData.length > 0 ? (
              <ResponsiveContainer width="100%" height="100%">
                <AreaChart data={latencyHistoryData}>
                  <CartesianGrid strokeDasharray="3 3" stroke="#374151" />
                  <XAxis
                    dataKey="time"
                    stroke="#9ca3af"
                    tickFormatter={(v) => `${v}s`}
                  />
                  <YAxis stroke="#9ca3af" tickFormatter={(v) => `${v}ms`} />
                  <Tooltip
                    contentStyle={{ backgroundColor: '#1f2937', border: 'none' }}
                    labelStyle={{ color: '#9ca3af' }}
                    formatter={(value: number) => [`${value}ms`, 'Latency']}
                  />
                  <Area
                    type="monotone"
                    dataKey="latency"
                    stroke="#8b5cf6"
                    fill="#8b5cf680"
                    strokeWidth={2}
                  />
                </AreaChart>
              </ResponsiveContainer>
            ) : (
              <div className="flex items-center justify-center h-full text-gray-500">
                No latency history yet
              </div>
            )}
          </div>
        </div>
      </div>

      {/* Charts Row 2 */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <div className="bg-gray-800 rounded-lg p-4">
          <h3 className="text-lg font-semibold text-white mb-4">Status Codes</h3>
          <div className="h-64">
            {statusCodeData.length > 0 ? (
              <ResponsiveContainer width="100%" height="100%">
                <PieChart>
                  <Pie
                    data={statusCodeData}
                    dataKey="value"
                    nameKey="name"
                    cx="50%"
                    cy="50%"
                    outerRadius={80}
                    label={({ name, percent }) => `${name} (${(percent * 100).toFixed(0)}%)`}
                  >
                    {statusCodeData.map((entry, index) => (
                      <Cell key={`cell-${index}`} fill={entry.color} />
                    ))}
                  </Pie>
                  <Tooltip
                    contentStyle={{ backgroundColor: '#1f2937', border: 'none' }}
                  />
                </PieChart>
              </ResponsiveContainer>
            ) : (
              <div className="flex items-center justify-center h-full text-gray-500">
                No data yet
              </div>
            )}
          </div>
        </div>

        <div className="bg-gray-800 rounded-lg p-4">
          <h3 className="text-lg font-semibold text-white mb-4">Error Breakdown</h3>
          <div className="h-64">
            {errorData.length > 0 ? (
              <ResponsiveContainer width="100%" height="100%">
                <PieChart>
                  <Pie
                    data={errorData}
                    dataKey="value"
                    nameKey="name"
                    cx="50%"
                    cy="50%"
                    outerRadius={80}
                    label={({ name, value }) => `${name}: ${value}`}
                  >
                    {errorData.map((entry, index) => (
                      <Cell key={`cell-${index}`} fill={entry.color} />
                    ))}
                  </Pie>
                  <Tooltip
                    contentStyle={{ backgroundColor: '#1f2937', border: 'none' }}
                  />
                </PieChart>
              </ResponsiveContainer>
            ) : (
              <div className="flex items-center justify-center h-full text-gray-500 text-center px-4">
                {metrics?.failures
                  ? `No network errors. ${metrics.failures} request(s) failed with HTTP error codes (see Status Codes).`
                  : 'No errors'}
              </div>
            )}
          </div>
        </div>
      </div>

      {/* Latency Percentiles Bar Chart */}
      <div className="bg-gray-800 rounded-lg p-4">
        <h3 className="text-lg font-semibold text-white mb-4">Latency Distribution (Min → Max)</h3>
        <div className="h-48">
          {latencyData.length > 0 ? (
            <ResponsiveContainer width="100%" height="100%">
              <BarChart data={latencyData}>
                <CartesianGrid strokeDasharray="3 3" stroke="#374151" />
                <XAxis dataKey="name" stroke="#9ca3af" />
                <YAxis stroke="#9ca3af" tickFormatter={(v) => `${v}ms`} />
                <Tooltip
                  contentStyle={{ backgroundColor: '#1f2937', border: 'none' }}
                  labelStyle={{ color: '#9ca3af' }}
                  formatter={(value: number) => [`${value}ms`, 'Latency']}
                />
                <Bar dataKey="value" fill="#8b5cf6" radius={[4, 4, 0, 0]} />
              </BarChart>
            </ResponsiveContainer>
          ) : (
            <div className="flex items-center justify-center h-full text-gray-500">
              No latency data yet
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
