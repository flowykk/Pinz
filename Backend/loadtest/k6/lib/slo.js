// Thresholds. Запросы помечаются tag class через requestParams().
// Единственный SLO из ТЗ (п. 4.2): trip_pipeline_duration_ms p99 < 15s — end-to-end
// создание путешествия. Остальные пороги — рабочие, для регрессии между прогонами.
// sync расслаблен до реалий: текущая кластеризация без ML делает p95 ~700-900мс.
export const slo = {
  http_req_failed:                          ['rate<0.01'],
  'http_req_duration{class:auth}':          ['p(95)<150', 'p(99)<300'],
  'http_req_duration{class:read}':          ['p(95)<200', 'p(99)<400'],
  'http_req_duration{class:feed}':          ['p(95)<500', 'p(99)<1200'],
  'http_req_duration{class:write}':         ['p(95)<400', 'p(99)<800'],
  'http_req_duration{class:sync}':          ['p(95)<1500', 'p(99)<3000'],
  trip_pipeline_duration_ms:                ['p(95)<8000', 'p(99)<15000'],
  ws_event_latency_ms:                      ['p(95)<200', 'p(99)<500'],
};

export function requestParams({className, token, contentType} = {}) {
  const headers = {};
  if (token) headers['Authorization'] = `Bearer ${token}`;
  if (contentType) headers['Content-Type'] = contentType;
  return {
    headers,
    tags: className ? {class: className} : undefined,
    timeout: '30s',
  };
}
