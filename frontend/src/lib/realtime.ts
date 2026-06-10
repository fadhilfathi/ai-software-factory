/**
 * Real-time WebSocket client with auto-reconnect and event dispatch.
 *
 * Connects to the backend WebSocket endpoint, reconnects on disconnect
 * with exponential backoff, and dispatches typed events to subscribers.
 */
import { useCallback, useEffect, useRef, useState } from "react";

// ─── Types ───────────────────────────────────────────────────────────────────

export type RealtimeEvent =
  | { type: "agent_status"; agent_id: string; status: string }
  | { type: "task_update"; task_id: string; project_id: string; status: string }
  | { type: "project_update"; project_id: string; status: string }
  | { type: "activity"; activity: { id: string; agent_type: string; text: string; created_at: string } }
  | { type: "notification"; notification: { type: string; title: string; message: string } }
  | { type: "pong" }
  | { type: string; [key: string]: unknown };

type EventHandler = (event: RealtimeEvent) => void;

// ─── Connection state ────────────────────────────────────────────────────────

export type ConnectionState = "disconnected" | "connecting" | "connected" | "reconnecting";

// ─── Client ──────────────────────────────────────────────────────────────────

const DEFAULT_URL = process.env.NEXT_PUBLIC_WS_URL || "ws://localhost:8080/v1/ws";
const MAX_RECONNECT_DELAY = 30_000; // 30s cap
const INITIAL_RECONNECT_DELAY = 1_000; // 1s

class RealtimeClient {
  private ws: WebSocket | null = null;
  private url: string;
  private handlers = new Map<string, Set<EventHandler>>();
  private _state: ConnectionState = "disconnected";
  private reconnectAttempts = 0;
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null;
  private pingInterval: ReturnType<typeof setInterval> | null = null;
  private intentionalClose = false;
  private onStateChange: ((state: ConnectionState) => void) | null = null;
  private accessTokenProvider: (() => string | null) | null = null;

  constructor(url: string = DEFAULT_URL) {
    this.url = url;
  }

  /** Register a state change listener. */
  setStateListener(fn: (state: ConnectionState) => void) {
    this.onStateChange = fn;
  }

  /** Provide auth token for the connection handshake. */
  setAccessTokenProvider(fn: () => string | null) {
    this.accessTokenProvider = fn;
  }

  get state(): ConnectionState {
    return this._state;
  }

  private setState(s: ConnectionState) {
    this._state = s;
    this.onStateChange?.(s);
  }

  /** Subscribe to a specific event type (or '*' for all). */
  on(eventType: string, handler: EventHandler): () => void {
    if (!this.handlers.has(eventType)) {
      this.handlers.set(eventType, new Set());
    }
    this.handlers.get(eventType)!.add(handler);
    return () => {
      this.handlers.get(eventType)?.delete(handler);
    };
  }

  /** Subscribe to multiple event types at once. */
  onMany(eventTypes: string[], handler: EventHandler): () => void {
    const unsubs = eventTypes.map((t) => this.on(t, handler));
    return () => unsubs.forEach((u) => u());
  }

  connect() {
    if (this.ws?.readyState === WebSocket.OPEN || this.ws?.readyState === WebSocket.CONNECTING) return;
    this.intentionalClose = false;
    this.setState("connecting");

    const token = this.accessTokenProvider?.();
    const wsUrl = token ? `${this.url}?token=${encodeURIComponent(token)}` : this.url;

    try {
      this.ws = new WebSocket(wsUrl);
    } catch {
      this.scheduleReconnect();
      return;
    }

    this.ws.onopen = () => {
      this.reconnectAttempts = 0;
      this.setState("connected");
      this.startPing();
    };

    this.ws.onmessage = (msg) => {
      try {
        const event = JSON.parse(msg.data) as RealtimeEvent;
        this.dispatch(event.type, event);
        // Also dispatch to wildcard listeners
        this.dispatch("*", event);
      } catch {
        // Ignore malformed messages
      }
    };

    this.ws.onclose = () => {
      this.stopPing();
      if (!this.intentionalClose) {
        this.setState("reconnecting");
        this.scheduleReconnect();
      } else {
        this.setState("disconnected");
      }
    };

    this.ws.onerror = () => {
      // onclose will fire after this, triggering reconnect
    };
  }

  disconnect() {
    this.intentionalClose = true;
    this.stopPing();
    this.clearReconnect();
    this.ws?.close();
    this.ws = null;
    this.setState("disconnected");
  }

  send(data: unknown) {
    if (this.ws?.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify(data));
    }
  }

  private dispatch(type: string, event: RealtimeEvent) {
    this.handlers.get(type)?.forEach((h) => {
      try {
        h(event);
      } catch {
        // Don't let one bad handler break others
      }
    });
  }

  private startPing() {
    this.stopPing();
    this.pingInterval = setInterval(() => {
      this.send({ type: "ping" });
    }, 25_000);
  }

  private stopPing() {
    if (this.pingInterval) {
      clearInterval(this.pingInterval);
      this.pingInterval = null;
    }
  }

  private scheduleReconnect() {
    this.clearReconnect();
    const delay = Math.min(
      INITIAL_RECONNECT_DELAY * Math.pow(2, this.reconnectAttempts),
      MAX_RECONNECT_DELAY,
    );
    this.reconnectAttempts++;
    this.reconnectTimer = setTimeout(() => this.connect(), delay);
  }

  private clearReconnect() {
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer);
      this.reconnectTimer = null;
    }
  }
}

// Singleton
let globalClient: RealtimeClient | null = null;

export function getRealtimeClient(): RealtimeClient {
  if (!globalClient) {
    globalClient = new RealtimeClient();
  }
  return globalClient;
}

// ─── React Hook ──────────────────────────────────────────────────────────────

interface UseRealtimeOptions {
  /** Event types to subscribe to. Subscribe to '*' for all events. */
  events?: string[];
  /** If true, automatically connect on mount (default: true). */
  autoConnect?: boolean;
}

interface UseRealtimeResult {
  /** Current connection state. */
  state: ConnectionState;
  /** Manually connect. */
  connect: () => void;
  /** Manually disconnect. */
  disconnect: () => void;
  /** Send a raw message. */
  send: (data: unknown) => void;
  /** Subscribe to an event. Returns unsubscribe function. */
  on: (eventType: string, handler: EventHandler) => () => void;
}

export function useRealtime({ events, autoConnect = true }: UseRealtimeOptions = {}): UseRealtimeResult {
  const client = getRealtimeClient();
  const [state, setState] = useState<ConnectionState>(client.state);

  // Track connection state changes
  useEffect(() => {
    client.setStateListener(setState);
    return () => client.setStateListener(() => {});
  }, [client]);

  // Auto-connect
  useEffect(() => {
    if (autoConnect) {
      client.connect();
    }
    return () => {
      // Don't disconnect on unmount — let the singleton live
    };
  }, [client, autoConnect]);

  const connect = useCallback(() => client.connect(), [client]);
  const disconnect = useCallback(() => client.disconnect(), [client]);
  const send = useCallback((data: unknown) => client.send(data), [client]);

  return { state, connect, disconnect, send, on: client.on.bind(client) };
}
