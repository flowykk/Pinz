// makeHandleSummary возвращает k6-хук handleSummary, который пишет:
//   reports/<name>-summary.txt   — человекочитаемая выжимка (как в stdout);
//   reports/<name>-summary.json  — машинный агрегат (p50/p95/p99/avg/min/max);
// и параллельно печатает summary в stdout.
import {textSummary} from 'https://jslib.k6.io/k6-summary/0.1.0/index.js';

export function makeHandleSummary(name) {
  return function handleSummary(data) {
    return {
      [`reports/${name}-summary.txt`]: textSummary(data, {indent: '  ', enableColors: false}),
      [`reports/${name}-summary.json`]: JSON.stringify(data, null, 2),
      stdout: textSummary(data, {indent: ' ', enableColors: true}),
    };
  };
}
