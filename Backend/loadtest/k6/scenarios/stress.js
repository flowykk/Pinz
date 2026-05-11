// Stress: ramp до отказа. Запускать с runner-VPS, не с ноутбука.
// SLO здесь ослаблен — задача найти потолок, а не подтвердить SLO.
import {sleep} from 'k6';
import {pickUser} from '../lib/data.js';
import {dailyUser} from '../lib/journeys.js';

const BASE = __ENV.BASE_URL;
if (!BASE) throw new Error('BASE_URL is required');

export const options = {
  scenarios: {
    ramp: {
      executor: 'ramping-vus',
      startVUs: 0,
      stages: [
        {duration: '5m',  target: 500},
        {duration: '5m',  target: 1500},
        {duration: '5m',  target: 3000},
        {duration: '5m',  target: 5000},
        {duration: '2m',  target: 0},
      ],
      gracefulStop: '30s',
    },
  },
  thresholds: {
    'http_req_failed': ['rate<0.10'],
  },
};

export default function () {
  dailyUser(BASE, pickUser());
  sleep(1);
}
