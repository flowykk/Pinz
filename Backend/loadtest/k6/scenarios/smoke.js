// Smoke: 1 VU, 1 минута, прогон всех journeys по разу.
import {sleep} from 'k6';
import {slo} from '../lib/slo.js';
import {pickUser} from '../lib/data.js';
import {dailyUser, tripCreation, addMediaToTrip, pinUpload, social, saveRecommendation} from '../lib/journeys.js';
import {makeHandleSummary} from '../lib/summary.js';

export const handleSummary = makeHandleSummary('smoke');

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
  const tripId = tripCreation(BASE, user, 10);
  addMediaToTrip(BASE, user, tripId, 5);
  pinUpload(BASE, user, tripId, 5);
  sleep(1);
}
