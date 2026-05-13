// Soak: 300 VU, 4 часа. Утечки памяти, рост connection-pool, поведение cleanup-воркеров.
import {sleep} from 'k6';
import {slo} from '../lib/slo.js';
import {pickUser} from '../lib/data.js';
import {dailyUser, tripCreation, addMediaToTrip, pinUpload, social, saveRecommendation} from '../lib/journeys.js';
import {makeHandleSummary} from '../lib/summary.js';

export const handleSummary = makeHandleSummary('soak');

const BASE = __ENV.BASE_URL;
if (!BASE) throw new Error('BASE_URL is required');

export const options = {
  scenarios: {
    daily:   {executor: 'constant-vus', vus: 180, duration: '4h', exec: 'daily'},
    create:  {executor: 'constant-vus', vus: 30,  duration: '4h', exec: 'create'},
    addmed:  {executor: 'constant-vus', vus: 15,  duration: '4h', exec: 'addmed'},
    pinup:   {executor: 'constant-vus', vus: 15,  duration: '4h', exec: 'pinup'},
    socialS: {executor: 'constant-vus', vus: 45,  duration: '4h', exec: 'socialS'},
    save:    {executor: 'constant-vus', vus: 15,  duration: '4h', exec: 'save'},
  },
  thresholds: slo,
};

export function daily()   { dailyUser(BASE, pickUser()); sleep(2); }
export function create()  { tripCreation(BASE, pickUser(), 50); sleep(5); }
export function addmed()  { const u = pickUser(); const t = tripCreation(BASE, u, 5); addMediaToTrip(BASE, u, t, 10); sleep(5); }
export function pinup()   { const u = pickUser(); const t = tripCreation(BASE, u, 5); pinUpload(BASE, u, t, 5); sleep(5); }
export function socialS() { social(BASE, pickUser(), null); sleep(2); }
export function save()    { saveRecommendation(BASE, pickUser()); sleep(5); }
