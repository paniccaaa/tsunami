import type { AttackConfig, StartResponse, StopResponse, StatusResponse, ResultsResponse } from '../types/api';

const API_URL = import.meta.env.VITE_API_URL || '';

export async function startAttack(config: AttackConfig): Promise<StartResponse> {
  const response = await fetch(`${API_URL}/api/attack/start`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(config),
  });

  if (!response.ok) {
    const error = await response.json();
    throw new Error(error.message || 'Failed to start attack');
  }

  return response.json();
}

export async function stopAttack(): Promise<StopResponse> {
  const response = await fetch(`${API_URL}/api/attack/stop`, {
    method: 'POST',
  });

  if (!response.ok) {
    const error = await response.json();
    throw new Error(error.message || 'Failed to stop attack');
  }

  return response.json();
}

export async function getStatus(): Promise<StatusResponse> {
  const response = await fetch(`${API_URL}/api/attack/status`);

  if (!response.ok) {
    const error = await response.json();
    throw new Error(error.message || 'Failed to get status');
  }

  return response.json();
}

export async function getResults(): Promise<ResultsResponse> {
  const response = await fetch(`${API_URL}/api/attack/results`);

  if (!response.ok) {
    const error = await response.json();
    throw new Error(error.message || 'Failed to get results');
  }

  return response.json();
}

export function downloadResults(): void {
  window.open(`${API_URL}/api/attack/results/download`, '_blank');
}
