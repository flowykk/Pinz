// Smoke: 1 VU, 1 минута, прогон всех journeys по разу. Падает если эндпоинты не отвечают.
import {sleep} from 'k6';
import {slo} from '../lib/slo.js';
import {pickUser} from '../lib/data.js';
import {dailyUser, tripCreation, social, saveRecommendation} from '../lib/journeys.js';

const BASE = __ENV.BASE_URL;
if (!BASE) throw new Error('BASE_URL is required');

export const options = {
  vus: 1,
  duration: '1m',
  thresholds: slo,
};

export default function () {
  const user = pickUser();
  dailyUser(BASE, user);
  saveRecommendation(BASE, user);
  social(BASE, user, null);
  tripCreation(BASE, user, 10);
  sleep(1);
}
