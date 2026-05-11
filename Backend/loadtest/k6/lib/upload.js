import http from 'k6/http';
import encoding from 'k6/encoding';

// Один буфер 1 МБ на все VU, чтобы не аллоцировать каждый раз.
const ONE_MB = (function () {
  const arr = new Uint8Array(1024 * 1024);
  for (let i = 0; i < arr.length; i++) arr[i] = i & 0xff;
  return arr.buffer;
})();

export function putToPresigned(presignedUrl, contentType) {
  const res = http.put(presignedUrl, ONE_MB, {
    headers: {'Content-Type': contentType || 'image/jpeg'},
    timeout: '30s',
    tags: {class: 'upload'},
  });
  return res.status >= 200 && res.status < 300;
}

export function fakeMediaMetadata(idx, opts) {
  const lat = (opts && opts.lat) || (55 + Math.random());
  const lon = (opts && opts.lon) || (37 + Math.random());
  const capturedAt = (opts && opts.capturedAt) || new Date(Date.now() - idx * 60000).toISOString();
  return {
    client_id: `cid_${idx}_${Math.random().toString(36).slice(2)}`,
    content_type: 'image/jpeg',
    captured_at: capturedAt,
    latitude: lat,
    longitude: lon,
  };
}

export function b64() {
  return encoding.b64encode(ONE_MB, 'rawstd');
}
