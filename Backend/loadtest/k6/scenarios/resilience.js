// Resilience: 200 VU mix; параллельно во внешнем shell-скрипте
// (loadtest/scripts/resilience-faults.sh) поочерёдно гасятся стабы:
//   minio (5–7 минута), mailpit (10–12 минута), geo-stub (15–17 минута).
// k6 здесь только генерирует нагрузку и фиксирует SLO; деградация должна
// быть локализована (только затронутые ручки), остальное продолжает работать.
import {sleep} from 'k6';
import {pickUser} from '../lib/data.js';
import {dailyUser, tripCreation, social} from '../lib/journeys.js';

const BASE = __ENV.BASE_URL;
if (!BASE) throw new Error('BASE_URL is required');

export const options = {
  scenarios: {
    daily:   {executor: 'constant-vus', vus: 150, duration: '20m', exec: 'daily'},
    create:  {executor: 'constant-vus', vus: 30,  duration: '20m', exec: 'create'},
    socialS: {executor: 'constant-vus', vus: 20,  duration: '20m', exec: 'socialS'},
  },
  thresholds: {
    // Сознательно ослаблено: ожидаем локальные ошибки на затронутых классах.
    'http_req_failed{class:read}':  ['rate<0.02'],
    'http_req_failed{class:feed}':  ['rate<0.05'],
    'http_req_failed{class:write}': ['rate<0.10'],
  },
};

export function daily()   { dailyUser(BASE, pickUser()); sleep(1); }
export function create()  { tripCreation(BASE, pickUser(), 20); sleep(3); }
export function socialS() { social(BASE, pickUser(), null); sleep(1); }
