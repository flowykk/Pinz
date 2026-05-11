// Целевой load: 0 → 1000 VU за 5 мин, hold 30 мин, mix согласно стратегии.
// Доли: S2=60%, S3=10%, S4 (add-media)=5%, S5=10%, S6 (pin-uploads)=5%, S7=3%, S8=2%.
// S4/S6 здесь упрощены до tripCreation+social для скорости первой итерации.
import {sleep} from 'k6';
import {slo} from '../lib/slo.js';
import {pickUser} from '../lib/data.js';
import {dailyUser, tripCreation, social, saveRecommendation} from '../lib/journeys.js';

const BASE = __ENV.BASE_URL;
if (!BASE) throw new Error('BASE_URL is required');

const stages = [
  {duration: '5m',  target: 1000},
  {duration: '30m', target: 1000},
  {duration: '2m',  target: 0},
];

export const options = {
  scenarios: {
    daily:    {executor: 'ramping-vus', startVUs: 0, stages, exec: 'daily',   gracefulStop: '30s'},
    create:   {executor: 'ramping-vus', startVUs: 0, stages: stages.map(s => ({...s, target: Math.round(s.target * 0.10)})), exec: 'create',  gracefulStop: '30s'},
    socialS:  {executor: 'ramping-vus', startVUs: 0, stages: stages.map(s => ({...s, target: Math.round(s.target * 0.10)})), exec: 'socialS', gracefulStop: '30s'},
    save:     {executor: 'ramping-vus', startVUs: 0, stages: stages.map(s => ({...s, target: Math.round(s.target * 0.02)})), exec: 'save',    gracefulStop: '30s'},
  },
  thresholds: slo,
};

export function daily()   { dailyUser(BASE, pickUser()); sleep(1); }
export function create()  { tripCreation(BASE, pickUser(), 50); sleep(3); }
export function socialS() { social(BASE, pickUser(), null); sleep(1); }
export function save()    { saveRecommendation(BASE, pickUser()); sleep(3); }
