import {escapeHTML as esc} from './forms.mjs';

export function experimentPlot(series, unit, timeLabel = 'Time (s)') {
 const points = series.flatMap(item => item.rows);
 const times = points.map(row => row.time_s);
 const values = points.map(row => row.values[0]);
 const start = Math.min(...times), end = Math.max(...times);
 const low = Math.min(...values), high = Math.max(...values);
 const x = value => 50 + 550 * (end === start ? .5 : (value - start) / (end - start));
 const y = value => 20 + 133 * (high === low ? .5 : (high - value) / (high - low));
 const number = value => Number(value.toPrecision(4));
 const lines = series.map((item, index) => {
  const color = index ? '#ca741c' : '#327ba6';
  const path = item.rows.map(row => `${x(row.time_s).toFixed(2)},${y(row.values[0]).toFixed(2)}`).join(' ');
  const point = item.rows[0];
  return item.rows.length === 1
   ? `<circle cx="${x(point.time_s) + index * 7}" cy="${y(point.values[0])}" r="4" fill="${color}"/>`
   : `<polyline points="${path}" fill="none" stroke="${color}" stroke-width="2.5" ${index ? 'stroke-dasharray="6 3"' : ''}/>`;
 }).join('');
 return `<svg class="experiment-plot" viewBox="0 0 625 198" role="img" aria-label="${esc(series.map(item => item.label).join(' and '))} versus time">
  <path d="M50 20V153H600" fill="none" stroke="currentColor" opacity=".4"/>
  <text x="44" y="24" text-anchor="end">${number(high)}</text>
  <text x="44" y="153" text-anchor="end">${number(low)}</text>
  <text x="50" y="170">${number(start)}</text><text x="600" y="170" text-anchor="end">${number(end)}</text>
  <text x="50" y="12">${esc(unit)}</text><text x="325" y="190" text-anchor="middle">${esc(timeLabel)}</text>
  ${lines}</svg>`;
}

export function firstCrossing(rows, level, direction) {
 for (let index = 1; index < rows.length; index++) {
  const a = rows[index - 1], b = rows[index];
  const from = a.values[0], to = b.values[0];
  if (direction * (from - level) >= 0 || direction * (to - level) < 0) continue;
  return a.time_s + (b.time_s - a.time_s) * (level - from) / (to - from);
 }
 return null;
}
