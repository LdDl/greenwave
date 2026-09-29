import { corridorGroupId, reverseCorridorGroupId } from './junction-program.js';

export function waveStatus(junctionCount, waves = [], throughWaves = [], calculated = false, outdated = false) {
  if (junctionCount < 2) return { kind: 'pending', message: 'Add at least two junctions to calculate waves.' };
  if (!calculated) return { kind: 'pending', message: 'Not calculated yet.' };
  if (outdated) return { kind: 'pending', message: 'Recalculate for the current configuration.' };
  const bandwidth = throughWaves.filter(wave => wave.depth === junctionCount).reduce((sum, wave) => sum + wave.bandwidth, 0);
  if (bandwidth > 0) return { kind: 'success', message: `Through all ${junctionCount} junctions: ${Number(bandwidth.toFixed(1))} s band.` };
  if (waves.some(segment => segment.some(wave => wave.band_width > 0)) || throughWaves.some(wave => wave.bandwidth > 0)) {
    return { kind: 'warning', message: 'No full-corridor wave. Only some junctions are connected by green waves.' };
  }
  return { kind: 'warning', message: 'No green wave found for this direction.' };
}

export function reverseGroupNotice(junctions, groupIds = {}, reverseGroupIds = {}) {
  if (!junctions.length) return '';
  const missing = [];
  const withoutGreen = [];
  let shared = 0;
  for (const junction of junctions) {
    const id = reverseCorridorGroupId(junction, groupIds, reverseGroupIds);
    const groups = junction.cycle.map(phase => phase.signal_groups.find(group => group.id === id));
    if (id === null || groups.some(group => !group)) missing.push(junction.label);
    else if (!groups.some(group => group.signals.some(signal => ['GREEN', 'GREENPRIORITY'].includes(signal.color.toUpperCase()) && signal.duration > 0))) withoutGreen.push(junction.label);
    if (id !== null && id === corridorGroupId(junction, groupIds)) shared++;
  }
  if (missing.length) return `Choose a reverse group in Groups: ${missing.join(', ')}.`;
  if (withoutGreen.length) return `The reverse group has no green signal: ${withoutGreen.join(', ')}.`;
  if (shared === junctions.length) return 'Reverse uses the forward group at every junction. A separate group can be selected in Groups.';
  if (shared) return `Reverse uses the forward group at ${shared} of ${junctions.length} junctions.`;
  return '';
}
