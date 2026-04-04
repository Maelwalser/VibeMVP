---
name: frontend-realtime-client
description: Real-time client patterns for React — WebSocket hook with reconnect, offline buffer, optimistic updates, SSE, Pusher, Ably, type-safe message contracts, and RealtimeProvider context.
origin: vibemenu
---

# Frontend Real-Time Client Patterns

React patterns for WebSocket, SSE, Pusher, and Ably. Covers reconnection, offline buffering, optimistic updates, and type-safe message contracts.

## When to Activate

- Manifest `frontend.tech.realtime_strategy` is set (WebSocket, SSE, Pusher, Ably)
- Building collaborative features, live dashboards, notifications, or chat
- Contracts include WebSocket endpoints or event-driven messages
- Pages use live data that updates without user-initiated refresh

---

## `useWebSocket` Hook with Automatic Reconnect

A production-ready hook with exponential backoff reconnection. Never use `new WebSocket()` directly in components.

```tsx
// hooks/useWebSocket.ts
import { useCallback, useEffect, useRef, useState } from 'react'

type Status = 'connecting' | 'open' | 'closed'

interface UseWebSocketOptions {
  onMessage?: (event: MessageEvent) => void
  onOpen?: () => void
  onClose?: () => void
  maxReconnectDelay?: number  // ms, default 30000
}

export function useWebSocket(url: string, options: UseWebSocketOptions = {}) {
  const wsRef = useRef<WebSocket | null>(null)
  const [status, setStatus] = useState<Status>('connecting')
  const reconnectAttempt = useRef(0)
  const reconnectTimer = useRef<ReturnType<typeof setTimeout> | null>(null)
  const mountedRef = useRef(true)
  const { onMessage, onOpen, onClose, maxReconnectDelay = 30_000 } = options

  const connect = useCallback(() => {
    if (!mountedRef.current) return

    const ws = new WebSocket(url)
    wsRef.current = ws

    ws.onopen = () => {
      if (!mountedRef.current) return
      setStatus('open')
      reconnectAttempt.current = 0
      onOpen?.()
    }

    ws.onmessage = (event) => {
      if (!mountedRef.current) return
      onMessage?.(event)
    }

    ws.onclose = () => {
      if (!mountedRef.current) return
      setStatus('closed')
      onClose?.()

      // Exponential backoff: 1s, 2s, 4s, 8s... capped at maxReconnectDelay
      const delay = Math.min(
        1000 * 2 ** reconnectAttempt.current,
        maxReconnectDelay
      )
      reconnectAttempt.current++
      reconnectTimer.current = setTimeout(connect, delay)
    }

    ws.onerror = () => {
      // onclose fires after onerror — reconnect logic lives there
    }
  }, [url, onMessage, onOpen, onClose, maxReconnectDelay])

  useEffect(() => {
    mountedRef.current = true
    connect()
    return () => {
      mountedRef.current = false
      if (reconnectTimer.current) clearTimeout(reconnectTimer.current)
      wsRef.current?.close()
    }
  }, [connect])

  const send = useCallback((data: string | ArrayBuffer | Blob) => {
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      wsRef.current.send(data)
      return true
    }
    return false
  }, [])

  return { ws: wsRef.current, status, send }
}
```

---

## Offline Message Buffer

Queue outgoing messages while disconnected. Drain the buffer automatically on reconnect.

```tsx
// hooks/useBufferedWebSocket.ts
import { useCallback, useRef } from 'react'
import { useWebSocket } from './useWebSocket'

export function useBufferedWebSocket(url: string) {
  const buffer = useRef<string[]>([])

  const { ws, status, send: rawSend } = useWebSocket(url, {
    onOpen: () => {
      // Drain buffer on reconnect
      const pending = buffer.current.splice(0)
      pending.forEach((msg) => rawSend(msg))
    },
  })

  const send = useCallback((msg: string) => {
    if (ws?.readyState === WebSocket.OPEN) {
      rawSend(msg)
    } else {
      // Queue for later delivery
      buffer.current.push(msg)
    }
  }, [ws, rawSend])

  return { status, send, bufferSize: buffer.current.length }
}
```

**Buffer limits:** Add a max buffer size to avoid unbounded memory growth during long disconnections:
```tsx
const MAX_BUFFER = 100
if (buffer.current.length >= MAX_BUFFER) {
  buffer.current.shift() // drop oldest message
}
buffer.current.push(msg)
```

---

## Type-Safe Message Schema

Define the message contract in a shared file. Import it in both the backend TypeScript handler and the frontend hook.

```ts
// contracts/messages.ts — shared between backend and frontend
export type ServerMessage =
  | { type: 'ITEM_CREATED'; data: Item }
  | { type: 'ITEM_UPDATED'; data: Item }
  | { type: 'ITEM_DELETED'; id: string }
  | { type: 'ERROR'; code: string; message: string }
  | { type: 'PING' }

export type ClientMessage =
  | { type: 'SUBSCRIBE'; channel: string }
  | { type: 'UNSUBSCRIBE'; channel: string }
  | { type: 'UPDATE_ITEM'; id: string; data: Partial<Item> }

// Type-safe parse — use at the socket boundary
export function parseServerMessage(raw: string): ServerMessage | null {
  try {
    const parsed = JSON.parse(raw) as unknown
    if (typeof parsed !== 'object' || parsed === null) return null
    if (!('type' in parsed)) return null
    return parsed as ServerMessage // narrow further with zod if needed
  } catch {
    return null
  }
}
```

```tsx
// Usage in component
const { status } = useWebSocket(url, {
  onMessage: (event) => {
    const msg = parseServerMessage(event.data)
    if (!msg) return
    switch (msg.type) {
      case 'ITEM_CREATED':
        setItems((prev) => [...prev, msg.data])
        break
      case 'ITEM_UPDATED':
        setItems((prev) => prev.map((i) => (i.id === msg.data.id ? msg.data : i)))
        break
      case 'ITEM_DELETED':
        setItems((prev) => prev.filter((i) => i.id !== msg.id))
        break
    }
  },
})
```

---

## Optimistic Updates

Apply changes to local state immediately — don't wait for server confirmation.

```tsx
// hooks/useOptimisticItems.ts
function useOptimisticItems(send: (msg: string) => void) {
  const [items, setItems] = useState<Item[]>([])
  // Track pending updates for rollback
  const pending = useRef<Map<string, Item>>(new Map())

  function updateItem(id: string, data: Partial<Item>) {
    // Save original for rollback
    const original = items.find((i) => i.id === id)
    if (original) pending.current.set(id, original)

    // Optimistic update — immediate UI feedback
    setItems((prev) => prev.map((i) => (i.id === id ? { ...i, ...data } : i)))

    // Send to server
    send(JSON.stringify({ type: 'UPDATE_ITEM', id, data }))
  }

  function handleServerMessage(msg: ServerMessage) {
    switch (msg.type) {
      case 'ITEM_UPDATED':
        // Server confirmed — clear pending, finalize with server data
        pending.current.delete(msg.data.id)
        setItems((prev) => prev.map((i) => (i.id === msg.data.id ? msg.data : i)))
        break
      case 'ERROR':
        // Rollback optimistic update on error
        const itemId = msg.code // encode item ID in error code field
        const original = pending.current.get(itemId)
        if (original) {
          setItems((prev) => prev.map((i) => (i.id === original.id ? original : i)))
          pending.current.delete(itemId)
        }
        break
    }
  }

  return { items, setItems, updateItem, handleServerMessage }
}
```

**Rule:** Never show optimistic state for destructive actions (delete, payment, transfer). Only use optimistic updates for reversible changes.

---

## Pusher JS SDK

Use when you need managed infrastructure with presence channels and private channel auth.

```tsx
// lib/pusher.ts — singleton client
import Pusher from 'pusher-js'

export const pusherClient = new Pusher(process.env.NEXT_PUBLIC_PUSHER_KEY!, {
  cluster: process.env.NEXT_PUBLIC_PUSHER_CLUSTER ?? 'eu',
  authEndpoint: '/api/pusher/auth',  // required for private/presence channels
})

// hooks/usePusherChannel.ts
import { useEffect, useRef } from 'react'
import { pusherClient } from '@/lib/pusher'

export function usePusherChannel<T>(
  channelName: string,
  eventName: string,
  handler: (data: T) => void
) {
  const handlerRef = useRef(handler)
  handlerRef.current = handler

  useEffect(() => {
    const channel = pusherClient.subscribe(channelName)
    channel.bind(eventName, (data: T) => handlerRef.current(data))

    return () => {
      channel.unbind(eventName)
      pusherClient.unsubscribe(channelName)
    }
  }, [channelName, eventName])
}
```

```tsx
// Server-side auth endpoint (Next.js App Router)
// app/api/pusher/auth/route.ts
import Pusher from 'pusher'
import { NextRequest } from 'next/server'

const pusherServer = new Pusher({
  appId: process.env.PUSHER_APP_ID!,
  key: process.env.PUSHER_KEY!,
  secret: process.env.PUSHER_SECRET!,
  cluster: process.env.PUSHER_CLUSTER!,
  useTLS: true,
})

export async function POST(req: NextRequest) {
  const { socket_id, channel_name } = await req.json()
  const user = await getAuthUser(req)  // validate session
  const authResponse = pusherServer.authorizeChannel(socket_id, channel_name, {
    user_id: user.id,
    user_info: { name: user.name },
  })
  return Response.json(authResponse)
}
```

---

## Ably JS SDK

Use Ably for guaranteed message delivery, message history, and edge network distribution.

```tsx
// lib/ably.ts
import * as Ably from 'ably'

// Token auth is more secure than key auth in browser clients
export const ablyClient = new Ably.Realtime({
  authUrl: '/api/ably/token',  // server-generated tokens
  authMethod: 'POST',
})

ablyClient.connection.on('connected', () => console.log('Ably connected'))
ablyClient.connection.on('disconnected', () => console.warn('Ably disconnected'))

// hooks/useAblyChannel.ts
export function useAblyChannel<T>(
  channelName: string,
  eventName: string,
  handler: (data: T) => void
) {
  const handlerRef = useRef(handler)
  handlerRef.current = handler

  useEffect(() => {
    const channel = ablyClient.channels.get(channelName)
    channel.subscribe(eventName, (msg) => handlerRef.current(msg.data as T))

    return () => {
      channel.unsubscribe(eventName)
      channel.detach()
    }
  }, [channelName, eventName])
}
```

```tsx
// Token auth endpoint (Next.js)
// app/api/ably/token/route.ts
import * as Ably from 'ably'

const ablyServer = new Ably.Rest({ key: process.env.ABLY_API_KEY! })

export async function POST(req: NextRequest) {
  const user = await getAuthUser(req)
  const tokenRequest = await ablyServer.auth.createTokenRequest({
    clientId: user.id,
    capability: { [`user:${user.id}`]: ['subscribe', 'publish'] },
  })
  return Response.json(tokenRequest)
}
```

---

## Server-Sent Events (SSE)

One-way push from server to client. Simpler than WebSocket — use it for read-only streams (notifications, progress, live feeds).

```tsx
// hooks/useSSE.ts
import { useEffect, useRef, useState } from 'react'

interface UseSSEOptions<T> {
  onMessage: (data: T) => void
  onError?: (err: Event) => void
}

export function useSSE<T>(url: string, options: UseSSEOptions<T>) {
  const [status, setStatus] = useState<'connecting' | 'open' | 'closed'>('connecting')
  const { onMessage, onError } = options

  useEffect(() => {
    const es = new EventSource(url, { withCredentials: true })

    es.onopen = () => setStatus('open')

    // Default message event
    es.onmessage = (e) => {
      try {
        onMessage(JSON.parse(e.data) as T)
      } catch {
        // ignore parse errors
      }
    }

    // Named event types
    es.addEventListener('notification', (e) => {
      onMessage(JSON.parse((e as MessageEvent).data) as T)
    })

    es.onerror = (e) => {
      setStatus('closed')
      onError?.(e)
      // EventSource auto-reconnects after onerror — no manual retry needed
    }

    return () => {
      es.close()
      setStatus('closed')
    }
  }, [url]) // eslint-disable-line react-hooks/exhaustive-deps

  return { status }
}
```

**Server-side SSE handler (Go):**
```go
func eventsHandler(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "text/event-stream")
    w.Header().Set("Cache-Control", "no-cache")
    w.Header().Set("Connection", "keep-alive")
    w.Header().Set("X-Accel-Buffering", "no") // disable nginx buffering

    flusher, ok := w.(http.Flusher)
    if !ok {
        http.Error(w, "streaming not supported", http.StatusInternalServerError)
        return
    }

    for {
        select {
        case <-r.Context().Done():
            return
        case event := <-eventChannel:
            data, _ := json.Marshal(event)
            fmt.Fprintf(w, "event: notification\ndata: %s\n\n", data)
            flusher.Flush()
        }
    }
}
```

**SSE advantages over WebSocket:**
- HTTP/1.1 compatible (works through proxies, load balancers without WebSocket support)
- Auto-reconnects built into the browser `EventSource` API
- Supports `Last-Event-ID` for resuming missed messages after disconnect
- Simpler server implementation — no upgrade handshake

**SSE limitation:** One-way only. Use WebSocket if clients need to send data back.

---

## Connection State via React Context

Wrap the WebSocket in a context provider so any component can access connection state without prop drilling.

```tsx
// context/RealtimeContext.tsx
import { createContext, useContext, useEffect, useRef, useState, ReactNode } from 'react'
import { ServerMessage, ClientMessage, parseServerMessage } from '@/contracts/messages'

interface RealtimeContextValue {
  status: 'connecting' | 'open' | 'closed'
  send: (msg: ClientMessage) => void
  subscribe: (type: ServerMessage['type'], handler: (msg: ServerMessage) => void) => () => void
}

const RealtimeContext = createContext<RealtimeContextValue | null>(null)

export function RealtimeProvider({ children, url }: { children: ReactNode; url: string }) {
  const [status, setStatus] = useState<RealtimeContextValue['status']>('connecting')
  const wsRef = useRef<WebSocket | null>(null)
  const handlers = useRef<Map<string, Set<(msg: ServerMessage) => void>>>(new Map())

  useEffect(() => {
    const ws = new WebSocket(url)
    wsRef.current = ws
    ws.onopen = () => setStatus('open')
    ws.onclose = () => setStatus('closed')
    ws.onmessage = (e) => {
      const msg = parseServerMessage(e.data)
      if (!msg) return
      handlers.current.get(msg.type)?.forEach((h) => h(msg))
    }
    return () => ws.close()
  }, [url])

  const send = (msg: ClientMessage) => {
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      wsRef.current.send(JSON.stringify(msg))
    }
  }

  const subscribe = (type: ServerMessage['type'], handler: (msg: ServerMessage) => void) => {
    if (!handlers.current.has(type)) handlers.current.set(type, new Set())
    handlers.current.get(type)!.add(handler)
    return () => handlers.current.get(type)?.delete(handler)
  }

  return (
    <RealtimeContext.Provider value={{ status, send, subscribe }}>
      {children}
    </RealtimeContext.Provider>
  )
}

export function useRealtime() {
  const ctx = useContext(RealtimeContext)
  if (!ctx) throw new Error('useRealtime must be used within RealtimeProvider')
  return ctx
}

// In components:
// const { status, subscribe, send } = useRealtime()
// useEffect(() => subscribe('ITEM_UPDATED', (msg) => { ... }), [subscribe])
```

---

## Anti-Patterns to Avoid

- **`new WebSocket()` in component body (not in `useEffect`)**: Creates a new connection on every render.
- **Missing cleanup in `useEffect`**: Leaked WebSocket connections exhaust browser limits (typically 6 per origin).
- **`JSON.parse` without try-catch**: A malformed server message crashes the client silently.
- **Reconnecting with fixed delay**: Fixed delays cause all clients to reconnect simultaneously after a server restart. Always add jitter.
- **Storing `ws.send` result without checking `readyState`**: `send()` on a closing/closed socket throws; always guard with `ws.readyState === WebSocket.OPEN`.
- **Pusher/Ably key in source code**: `NEXT_PUBLIC_PUSHER_SECRET` would expose it. Only the publishable key goes in `NEXT_PUBLIC_*` — secrets go in server-only env vars.
