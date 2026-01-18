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

function getStatusColor(code: number): string {
  return STATUS_COLORS[String(code)] || (code < 300 ? '#22c55e' : code < 400 ? '#eab308' : code < 500 ? '#f97316' : '#ef4444');
}

function MetricCard({ label, value, subvalue }: { label: string; value: string | number; subvalue?: string }) {
  return (
    <div className="bg-gray-800 rounded-lg p-4">
      <div className="text-sm text-gray-400 mb-1">{label}</div>
      <div className="text-2xl font-bold text-white">{value}</div>
      {subvalue && <div className="text-sm text-gray-500">{subvalue}</div>}
    </div>
  );
}

export function MetricsDashboard({ metrics, metricsHistory, isRunning }: MetricsDashboardProps) {
  const rpsData = useMemo(() => {
    return metricsHistory.map((m, i) => ({
      time: i * 0.5,
      rps: Math.round(m.current_rps * 100) / 100,
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
      { name: 'P50', value: parseLatency(metrics.latency_percentiles.p50) },
      { name: 'P90', value: parseLatency(metrics.latency_percentiles.p90) },
      { name: 'P95', value: parseLatency(metrics.latency_percentiles.p95) },
      { name: 'P99', value: parseLatency(metrics.latency_percentiles.p99) },
    ];
  }, [metrics?.latency_percentiles]);

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

      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
        <MetricCard
          label="Total Requests"
          value={metrics?.total_requests?.toLocaleString() || 0}
        />
        <MetricCard
          label="Current RPS"
          value={metrics?.current_rps?.toFixed(1) || '0.0'}
        />
        <MetricCard
          label="Success Rate"
          value={
            metrics?.total_requests
              ? `${((metrics.successes / metrics.total_requests) * 100).toFixed(1)}%`
              : '0%'
          }
          subvalue={`${metrics?.successes || 0} / ${metrics?.failures || 0}`}
        />
        <MetricCard
          label="Avg Latency"
          value={metrics?.average_latency || '0ms'}
        />
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <div className="bg-gray-800 rounded-lg p-4">
          <h3 className="text-lg font-semibold text-white mb-4">RPS Over Time</h3>
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
                  dataKey="rps"
                  stroke="#3b82f6"
                  strokeWidth={2}
                  dot={false}
                />
              </LineChart>
            </ResponsiveContainer>
          </div>
        </div>

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

        <div className="bg-gray-800 rounded-lg p-4 lg:col-span-2">
          <h3 className="text-lg font-semibold text-white mb-4">Latency Percentiles</h3>
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
    </div>
  );
}
