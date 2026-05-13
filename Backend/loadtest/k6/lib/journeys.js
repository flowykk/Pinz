// User journeys S2/S3/S4/S5/S6/S8.
// S3/S4/S6 (pipeline) выходят штатно при отсутствии WS-события от ML — HTTP-ручки
// до apply нагружают backend в любом случае.
import http from 'k6/http';
import {check, sleep} from 'k6';
import {requestParams} from './slo.js';
import {waitForEvent, tripPipelineDurationMs} from './ws.js';
import {fakeMediaMetadata} from './upload.js';

const allow200_404 = http.expectedStatuses(200, 404);
const allow2xx_404 = http.expectedStatuses(
  {min: 200, max: 299}, 404,
);
const allow2xx_4xx = http.expectedStatuses(
  {min: 200, max: 299}, {min: 400, max: 499},
);

const WS_WAIT_MS = parseInt(__ENV.WS_WAIT_MS || '3000', 10);

// S2.
export function dailyUser(baseUrl, user) {
  const tok = user.accessToken;
  const list = http.get(`${baseUrl}/api/v1/trips`, requestParams({className: 'read', token: tok}));
  check(list, {'list trips 200': (r) => r.status === 200});

  for (let page = 1; page <= 3; page++) {
    const feed = http.get(`${baseUrl}/api/v1/feed?page=${page}&limit=20`, requestParams({className: 'feed', token: tok}));
    check(feed, {'feed 200': (r) => r.status === 200});
  }

  const recsParams = requestParams({className: 'feed', token: tok});
  recsParams.responseCallback = allow200_404;
  const recs = http.get(`${baseUrl}/api/v1/recommendations?city=TestCity_55_37`, recsParams);
  check(recs, {'recs 2xx/404': (r) => r.status === 200 || r.status === 404});

  const search = http.get(`${baseUrl}/api/v1/pins/search?q=test`, requestParams({className: 'feed', token: tok}));
  check(search, {'search 200': (r) => r.status === 200});
}

// S3. Возвращает tripId после apply; finalize идёт только при ML на стенде.
export function tripCreation(baseUrl, user, mediaCount = 50) {
  const tok = user.accessToken;
  const t0 = Date.now();

  const create = http.post(
    `${baseUrl}/api/v1/trips/creation/start`,
    JSON.stringify({name: `loadtest-${Date.now()}`, category: 'vacation', season: 'summer'}),
    requestParams({className: 'write', token: tok, contentType: 'application/json'}),
  );
  if (!check(create, {'trip create 2xx': (r) => r.status === 200 || r.status === 201})) return null;
  const tripId = create.json('trip_id') || create.json('id') || create.json('trip.id');
  if (!tripId) return null;

  const media = [];
  for (let i = 0; i < mediaCount; i++) media.push(fakeMediaMetadata(i));
  const grouping = http.post(
    `${baseUrl}/api/v1/trips/creation/${tripId}/media/process-grouping`,
    JSON.stringify({media}),
    requestParams({className: 'sync', token: tok, contentType: 'application/json'}),
  );
  check(grouping, {'process-grouping 200': (r) => r.status === 200});

  const applyParams = requestParams({className: 'sync', token: tok, contentType: 'application/json'});
  applyParams.responseCallback = allow2xx_4xx;
  const apply = http.post(
    `${baseUrl}/api/v1/trips/creation/${tripId}/apply-groups-and-process`,
    JSON.stringify({}),
    applyParams,
  );
  check(apply, {'apply 2xx/4xx': (r) => r.status < 500});
  if (apply.status >= 300) return tripId;

  const wsBase = baseUrl.replace(/^http/, 'ws');
  const ev = waitForEvent({
    url: `${wsBase}/api/v1/trips/creation/${tripId}/review/ws`,
    token: tok,
    eventType: 'TRIP_PROCESSING_COMPLETED',
    timeoutMs: WS_WAIT_MS,
  });
  if (!ev.ok) return tripId;

  const review = http.get(`${baseUrl}/api/v1/trips/creation/${tripId}/review`, requestParams({className: 'read', token: tok}));
  check(review, {'review 200': (r) => r.status === 200});

  const finalize = http.post(
    `${baseUrl}/api/v1/trips/creation/${tripId}/finalize`,
    JSON.stringify({pins: []}),
    requestParams({className: 'sync', token: tok, contentType: 'application/json'}),
  );
  check(finalize, {'finalize 200': (r) => r.status === 200});
  tripPipelineDurationMs.add(Date.now() - t0);
  return tripId;
}

// S4. Кооперативное добавление медиа; без ML останавливается на apply.
export function addMediaToTrip(baseUrl, user, tripId, mediaCount = 20) {
  if (!tripId) return;
  const tok = user.accessToken;

  const startParams = requestParams({className: 'write', token: tok, contentType: 'application/json'});
  startParams.responseCallback = allow2xx_4xx;
  const start = http.post(`${baseUrl}/api/v1/trips/${tripId}/media/add/start`, '{}', startParams);
  check(start, {'add-media start ok': (r) => r.status < 500});
  if (start.status >= 300) return;

  const media = [];
  for (let i = 0; i < mediaCount; i++) media.push(fakeMediaMetadata(i));
  const grouping = http.post(
    `${baseUrl}/api/v1/trips/${tripId}/media/add/process-grouping`,
    JSON.stringify({media}),
    requestParams({className: 'sync', token: tok, contentType: 'application/json'}),
  );
  check(grouping, {'add-media grouping 2xx/4xx': (r) => r.status < 500});

  const applyParams = requestParams({className: 'sync', token: tok, contentType: 'application/json'});
  applyParams.responseCallback = allow2xx_4xx;
  http.post(
    `${baseUrl}/api/v1/trips/${tripId}/media/add/apply-groups-and-process`,
    '{}',
    applyParams,
  );
  // cancel освобождает сессию (на одну поездку — одна активная сессия добавления медиа).
  http.post(`${baseUrl}/api/v1/trips/${tripId}/media/add/cancel`, null, applyParams);
}

// S6. Pin-uploads (объединённый flow); без ML останавливается на process.
export function pinUpload(baseUrl, user, tripId, mediaCount = 10) {
  if (!tripId) return;
  const tok = user.accessToken;
  const writeP = requestParams({className: 'write', token: tok, contentType: 'application/json'});
  writeP.responseCallback = allow2xx_4xx;

  const start = http.post(`${baseUrl}/api/v1/trips/${tripId}/pin-uploads/start`, '{}', writeP);
  if (start.status >= 300) return;
  const sid = start.json('session_id') || start.json('sid');
  if (!sid) return;

  const upload = http.post(
    `${baseUrl}/api/v1/trips/${tripId}/pin-uploads/${sid}/upload-urls`,
    JSON.stringify({files: [{client_id: 'cid_0', content_type: 'image/jpeg'}]}),
    writeP,
  );
  check(upload, {'pin-upload urls 2xx/4xx': (r) => r.status < 500});

  http.post(
    `${baseUrl}/api/v1/trips/${tripId}/pin-uploads/${sid}/process`,
    '{}',
    requestParams({className: 'sync', token: tok, contentType: 'application/json'}),
  );
  http.post(`${baseUrl}/api/v1/trips/${tripId}/pin-uploads/${sid}/cancel`, null, writeP);
}

// S5.
export function social(baseUrl, user, tripId) {
  const tok = user.accessToken;
  if (!tripId) {
    const feed = http.get(`${baseUrl}/api/v1/feed?page=1&limit=20`, requestParams({className: 'feed', token: tok}));
    const arr = feed.json('items') || feed.json('trips') || [];
    if (!arr.length) return;
    tripId = arr[Math.floor(Math.random() * arr.length)].id;
  }

  const like = http.post(`${baseUrl}/api/v1/trips/${tripId}/like`, null, requestParams({className: 'write', token: tok}));
  check(like, {'like 2xx': (r) => r.status >= 200 && r.status < 300});

  const fav = http.post(`${baseUrl}/api/v1/trips/${tripId}/favourite`, null, requestParams({className: 'write', token: tok}));
  check(fav, {'fav 2xx': (r) => r.status >= 200 && r.status < 300});

  sleep(0.2);

  http.del(`${baseUrl}/api/v1/trips/${tripId}/favourite`, null, requestParams({className: 'write', token: tok}));
}

// S8.
export function saveRecommendation(baseUrl, user) {
  const tok = user.accessToken;
  const recsParams = requestParams({className: 'feed', token: tok});
  recsParams.responseCallback = allow200_404;
  const recs = http.get(`${baseUrl}/api/v1/recommendations?city=TestCity_55_37`, recsParams);
  if (recs.status !== 200) return;
  const token = recs.json('snapshot_token');
  if (!token) return;
  const save = http.post(
    `${baseUrl}/api/v1/recommendations/save`,
    JSON.stringify({snapshot_token: token}),
    requestParams({className: 'write', token: tok, contentType: 'application/json'}),
  );
  check(save, {'save rec 2xx': (r) => r.status >= 200 && r.status < 300});
}
