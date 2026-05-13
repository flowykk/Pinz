import {SharedArray} from 'k6/data';

// credentials.json формата [{email, accessToken, refreshToken, userId}, ...].
export const credentials = new SharedArray('credentials', function () {
  const path = __ENV.CREDENTIALS_PATH || './seed/credentials.json';
  return JSON.parse(open(path));
});

export const seededTrips = new SharedArray('seededTrips', function () {
  const path = __ENV.SEEDED_TRIPS_PATH || './seed/trips.json';
  try {
    return JSON.parse(open(path));
  } catch (_) {
    return [];
  }
});

export function pickUser() {
  return credentials[Math.floor(Math.random() * credentials.length)];
}

export function pickTrip() {
  if (seededTrips.length === 0) return null;
  return seededTrips[Math.floor(Math.random() * seededTrips.length)];
}
