import http from 'k6/http';
import {check} from 'k6';
import {requestParams} from './slo.js';

// Требует AUTH_DEV_LOGIN_ENABLED + DEV_LOGIN_PROXY_ENABLED на стенде.
export function devLogin(baseUrl, email) {
  const res = http.post(
    `${baseUrl}/api/v1/auth/dev-login`,
    JSON.stringify({email}),
    requestParams({className: 'auth', contentType: 'application/json'}),
  );
  check(res, {'dev-login 200': (r) => r.status === 200});
  if (res.status !== 200) {
    throw new Error(`dev-login failed for ${email}: ${res.status} ${res.body}`);
  }
  return {
    accessToken: res.json('access_token'),
    refreshToken: res.json('refresh_token'),
  };
}

export function refresh(baseUrl, refreshToken) {
  const res = http.post(
    `${baseUrl}/api/v1/auth/refresh`,
    JSON.stringify({refresh_token: refreshToken}),
    requestParams({className: 'auth', contentType: 'application/json'}),
  );
  check(res, {'refresh 200': (r) => r.status === 200});
  return res.json('access_token');
}
