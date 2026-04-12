import type { WSMessage, MetricsPayload } from '../types/api';

type MessageHandler = (data: MetricsPayload) => void;
type StatusHandler = (type: string, errorMessage?: string) => void;

export class MetricsWebSocket {
  private ws: WebSocket | null = null;
  private onMetrics: MessageHandler;
  private onStatus: StatusHandler;
  private reconnectAttempts = 0;
  private maxReconnectAttempts = 5;
  private reconnectDelay = 1000;

  constructor(onMetrics: MessageHandler, onStatus: StatusHandler) {
    this.onMetrics = onMetrics;
    this.onStatus = onStatus;
  }

  connect(): void {
    const wsUrl = import.meta.env.VITE_WS_URL || `ws://${window.location.host}`;
    this.ws = new WebSocket(`${wsUrl}/ws/metrics`);

    this.ws.onopen = () => {
      console.log('WebSocket connected');
      this.reconnectAttempts = 0;
    };

    this.ws.onmessage = (event) => {
      try {
        const message: WSMessage = JSON.parse(event.data);

        if (message.type === 'metrics' && message.data) {
          this.onMetrics(message.data as MetricsPayload);
        } else if (message.type === 'completed' || message.type === 'stopped') {
          this.onStatus(message.type);
          if (message.data) {
            this.onMetrics(message.data as MetricsPayload);
          }
        } else if (message.type === 'error') {
          const errorMsg = (message.data as Record<string, string>)?.error;
          this.onStatus('error', errorMsg);
        }
      } catch (err) {
        console.error('Failed to parse WebSocket message:', err);
      }
    };

    this.ws.onclose = () => {
      console.log('WebSocket disconnected');
      this.tryReconnect();
    };

    this.ws.onerror = (error) => {
      console.error('WebSocket error:', error);
    };
  }

  private tryReconnect(): void {
    if (this.reconnectAttempts < this.maxReconnectAttempts) {
      this.reconnectAttempts++;
      console.log(`Reconnecting... attempt ${this.reconnectAttempts}`);
      setTimeout(() => this.connect(), this.reconnectDelay * this.reconnectAttempts);
    }
  }

  disconnect(): void {
    if (this.ws) {
      this.ws.close();
      this.ws = null;
    }
  }

  isConnected(): boolean {
    return this.ws?.readyState === WebSocket.OPEN;
  }
}
