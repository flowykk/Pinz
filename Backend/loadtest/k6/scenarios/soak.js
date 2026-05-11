// Soak: 300 VU, 4 часа. Утечки памяти, рост connection-pool, поведение cleanup-воркеров.
import {sleep} from 'k6';
import {slo} from '../lib/slo.js';
import {pickUser} from '../lib/data.js';
import {dailyUser, tripCreation, social, saveRecommendation} from '../lib/journeys.js';

const BASE = __ENV.BASE_URL;
if (!BASE) throw new Error('BASE_URL is required');

export const options = {
  scenarios: {
    daily:    {executor: 'constant-vus', vus: 200, duration: '4h', exec: 'daily'},
    create:   {executor: 'constant-vus', vus: 30,  duration: '4h', exec: 'create'},
    socialS:  {executor: 'constant-vus', vus: 50,  duration: '4h', exec: 'socialS'},
    save:     {executor: 'constant-vus', vus: 20,  duration: '4h', exec: 'save'},
  },
  thresholds: slo,
};

export function daily()   { dailyUser(BASE, pickUser()); sleep(2); }
export function create()  { tripCreation(BASE, pickUser(), 50); sleep(5); }
export function socialS() { social(BASE, pickUser(), null); sleep(2); }
export function save()    { saveRecommendation(BASE, pickUser()); sleep(5); }
