import ws from 'k6/ws';
import {Trend} from 'k6/metrics';

export const wsEventLatencyMs = new Trend('ws_event_latency_ms', true);
export const tripPipelineDurationMs = new Trend('trip_pipeline_duration_ms', true);

// waitForEvent открывает WS, ждёт событие нужного type, возвращает {ok, durationMs}.
// timeoutMs — максимальное время ожидания.
export function waitForEvent({url, token, eventType, timeoutMs}) {
  let received = false;
  let durationMs = 0;
  const startedAt = Date.now();

  const wsUrl = token ? `${url}?token=${encodeURIComponent(token)}` : url;
  const params = token ? {headers: {Authorization: `Bearer ${token}`}} : {};

  const res = ws.connect(wsUrl, params, function (socket) {
    socket.setTimeout(function () { socket.close(); }, timeoutMs || 30000);
    socket.on('message', function (raw) {
      try {
        const msg = JSON.parse(raw);
        if (msg.type === eventType) {
          received = true;
          durationMs = Date.now() - startedAt;
          if (msg.published_at) {
            const pub = Date.parse(msg.published_at);
            if (!Number.isNaN(pub)) wsEventLatencyMs.add(Date.now() - pub);
          }
          socket.close();
        }
      } catch (_) { /* ignore non-JSON frames */ }
    });
    socket.on('error', function () { socket.close(); });
  });
  return {ok: received && res && res.status === 101, durationMs};
}
