// User journeys S2/S3/S5/S8 из стратегии (vkr/loadTestingStrategy.md).
import http from 'k6/http';
import {check, sleep} from 'k6';
import {requestParams} from './slo.js';
import {waitForEvent, tripPipelineDurationMs} from './ws.js';
import {fakeMediaMetadata} from './upload.js';

// S2.
export function dailyUser(baseUrl, user) {
  const tok = user.accessToken;
  const list = http.get(`${baseUrl}/api/v1/trips`, requestParams({className: 'read', token: tok}));
  check(list, {'list trips 200': (r) => r.status === 200});

  for (let page = 1; page <= 3; page++) {
    const feed = http.get(`${baseUrl}/api/v1/feed?page=${page}&limit=20`, requestParams({className: 'feed', token: tok}));
    check(feed, {'feed 200': (r) => r.status === 200});
  }

  const recs = http.get(`${baseUrl}/api/v1/recommendations?city=TestCity_55_37`, requestParams({className: 'feed', token: tok}));
  check(recs, {'recs 200': (r) => r.status === 200});

  const search = http.get(`${baseUrl}/api/v1/pins/search?q=test`, requestParams({className: 'feed', token: tok}));
  check(search, {'search 200': (r) => r.status === 200});
}

// S3.
export function tripCreation(baseUrl, user, mediaCount = 50) {
  const tok = user.accessToken;
  const t0 = Date.now();

  const create = http.post(
    `${baseUrl}/api/v1/trips/creation/start`,
    JSON.stringify({title: `loadtest-${Date.now()}`, category: 'VACATION', season: 'SUMMER'}),
    requestParams({className: 'write', token: tok, contentType: 'application/json'}),
  );
  if (!check(create, {'trip create 200': (r) => r.status === 200 || r.status === 201})) return false;
  const tripId = create.json('trip.id') || create.json('id');
  if (!tripId) return false;

  const media = [];
  for (let i = 0; i < mediaCount; i++) media.push(fakeMediaMetadata(i));
  const grouping = http.post(
    `${baseUrl}/api/v1/trips/creation/${tripId}/media/process-grouping`,
    JSON.stringify({media}),
    requestParams({className: 'sync', token: tok, contentType: 'application/json'}),
  );
  check(grouping, {'process-grouping 200': (r) => r.status === 200});

  const apply = http.post(
    `${baseUrl}/api/v1/trips/creation/${tripId}/apply-groups-and-process`,
    JSON.stringify({}),
    requestParams({className: 'sync', token: tok, contentType: 'application/json'}),
  );
  check(apply, {'apply 200': (r) => r.status === 200});

  const wsBase = baseUrl.replace(/^http/, 'ws');
  const ev = waitForEvent({
    url: `${wsBase}/api/v1/trips/creation/${tripId}/review/ws`,
    token: tok,
    eventType: 'TRIP_PROCESSING_COMPLETED',
    timeoutMs: 20000,
  });
  if (!ev.ok) return false;

  const review = http.get(`${baseUrl}/api/v1/trips/creation/${tripId}/review`, requestParams({className: 'read', token: tok}));
  check(review, {'review 200': (r) => r.status === 200});

  const finalize = http.post(
    `${baseUrl}/api/v1/trips/creation/${tripId}/finalize`,
    JSON.stringify({pins: []}),
    requestParams({className: 'sync', token: tok, contentType: 'application/json'}),
  );
  check(finalize, {'finalize 200': (r) => r.status === 200});

  tripPipelineDurationMs.add(Date.now() - t0);
  return true;
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
  const recs = http.get(`${baseUrl}/api/v1/recommendations?city=TestCity_55_37`, requestParams({className: 'feed', token: tok}));
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
