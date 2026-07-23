// Целевой load. По умолчанию 1000 VU, ramp 5 мин + hold 30 мин.
// Переопределяется env: LOAD_VUS, LOAD_RAMP_S, LOAD_HOLD_S.
import {sleep} from 'k6';
import {slo} from '../lib/slo.js';
import {pickUser} from '../lib/data.js';
import {dailyUser, tripCreation, addMediaToTrip, pinUpload, social, saveRecommendation} from '../lib/journeys.js';
import {makeHandleSummary} from '../lib/summary.js';

export const handleSummary = makeHandleSummary('load');

const BASE = __ENV.BASE_URL;
if (!BASE) throw new Error('BASE_URL is required');

const TARGET = parseInt(__ENV.LOAD_VUS    || '1000', 10);
const RAMP_S = parseInt(__ENV.LOAD_RAMP_S || '300', 10);
const HOLD_S = parseInt(__ENV.LOAD_HOLD_S || '1800', 10);

const stages = [
  {duration: `${RAMP_S}s`, target: TARGET},
  {duration: `${HOLD_S}s`, target: TARGET},
  {duration: '60s',        target: 0},
];

const share = (frac) => stages.map(s => ({...s, target: Math.max(1, Math.round(s.target * frac))}));

export const options = {
  scenarios: {
    daily:   {executor: 'ramping-vus', startVUs: 0, stages: share(0.60), exec: 'daily',   gracefulStop: '30s'},
    create:  {executor: 'ramping-vus', startVUs: 0, stages: share(0.10), exec: 'create',  gracefulStop: '30s'},
    addmed:  {executor: 'ramping-vus', startVUs: 0, stages: share(0.05), exec: 'addmed',  gracefulStop: '30s'},
    pinup:   {executor: 'ramping-vus', startVUs: 0, stages: share(0.05), exec: 'pinup',   gracefulStop: '30s'},
    socialS: {executor: 'ramping-vus', startVUs: 0, stages: share(0.15), exec: 'socialS', gracefulStop: '30s'},
    save:    {executor: 'ramping-vus', startVUs: 0, stages: share(0.05), exec: 'save',    gracefulStop: '30s'},
  },
  thresholds: slo,
};

export function daily()   { dailyUser(BASE, pickUser()); sleep(1); }
export function create()  { tripCreation(BASE, pickUser(), 50); sleep(3); }
export function addmed()  { const u = pickUser(); const t = tripCreation(BASE, u, 5); addMediaToTrip(BASE, u, t, 10); sleep(3); }
export function pinup()   { const u = pickUser(); const t = tripCreation(BASE, u, 5); pinUpload(BASE, u, t, 5); sleep(3); }
export function socialS() { social(BASE, pickUser(), null); sleep(1); }
export function save()    { saveRecommendation(BASE, pickUser()); sleep(3); }
