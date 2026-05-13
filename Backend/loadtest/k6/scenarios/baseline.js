// Baseline: 50 VU, 10 минут, traffic-mix. Регрессионная база.
import {sleep} from 'k6';
import {slo} from '../lib/slo.js';
import {pickUser} from '../lib/data.js';
import {dailyUser, tripCreation, addMediaToTrip, pinUpload, social, saveRecommendation} from '../lib/journeys.js';
import {makeHandleSummary} from '../lib/summary.js';

export const handleSummary = makeHandleSummary('baseline');

const BASE = __ENV.BASE_URL;
if (!BASE) throw new Error('BASE_URL is required');

export const options = {
  scenarios: {
    daily:   {executor: 'constant-vus', vus: 30, duration: '10m', exec: 'daily'},
    create:  {executor: 'constant-vus', vus: 5,  duration: '10m', exec: 'create'},
    addmed:  {executor: 'constant-vus', vus: 3,  duration: '10m', exec: 'addmed'},
    pinup:   {executor: 'constant-vus', vus: 3,  duration: '10m', exec: 'pinup'},
    socialS: {executor: 'constant-vus', vus: 7,  duration: '10m', exec: 'socialS'},
    save:    {executor: 'constant-vus', vus: 2,  duration: '10m', exec: 'save'},
  },
  thresholds: slo,
};

export function daily()   { dailyUser(BASE, pickUser()); sleep(1); }
export function create()  { tripCreation(BASE, pickUser(), 50); sleep(2); }
export function addmed()  { const u = pickUser(); const t = tripCreation(BASE, u, 5); addMediaToTrip(BASE, u, t, 10); sleep(2); }
export function pinup()   { const u = pickUser(); const t = tripCreation(BASE, u, 5); pinUpload(BASE, u, t, 5); sleep(2); }
export function socialS() { social(BASE, pickUser(), null); sleep(1); }
export function save()    { saveRecommendation(BASE, pickUser()); sleep(2); }
