export function signalColor(color) {
  return { GREEN: '#16a34a', RED: '#dc2626', YELLOW: '#ca8a04' }[color] ?? '#64748b';
}

export function programGroupIds(junction) {
  return [...new Set((junction?.cycle ?? []).flatMap(phase => phase.signal_groups.map(group => group.id)))];
}

export function readProgram(junction) {
  const groupIds = programGroupIds(junction);
  if (!junction?.cycle?.length || !groupIds.length) throw new Error('A program needs phases and signal groups.');
  const phaseDurations = junction.cycle.map((phase, index) => {
    if (phase.signal_groups.length !== groupIds.length || groupIds.some(id => !phase.signal_groups.some(group => group.id === id))) {
      throw new Error(`Phase ${index + 1}: every phase must contain the same permanent signal groups.`);
    }
    const totals = phase.signal_groups.map(group => {
      if (!group.signals.length || group.signals.some(signal => !Number.isFinite(signal.duration) || signal.duration <= 0)) {
        throw new Error(`Phase ${index + 1}, G${group.id}: signal duration must be positive.`);
      }
      return { id: group.id, duration: group.signals.reduce((sum, signal) => sum + signal.duration, 0) };
    });
    if (totals.some(group => !Number.isFinite(group.duration))) throw new Error('Program duration exceeds the supported range.');
    const duration = totals[0].duration;
    if (totals.some(group => Math.abs(group.duration - duration) > 1e-8)) {
      throw new Error(`Phase ${index + 1}: group durations must match (${totals.map(group => `G${group.id}: ${group.duration} s`).join(', ')}).`);
    }
    return duration;
  });
  const duration = phaseDurations.reduce((sum, duration) => sum + duration, 0);
  if (!Number.isFinite(duration)) throw new Error('Program duration exceeds the supported range.');
  return { groupIds, phaseDurations, duration };
}

export function corridorGroupId(junction, selections = {}) {
  if (Object.hasOwn(selections, junction.id)) return selections[junction.id];
  const ids = programGroupIds(junction);
  return ids.length === 1 ? ids[0] : null;
}

export function reverseCorridorGroupId(junction, selections = {}, reverseSelections = {}) {
  return Object.hasOwn(reverseSelections, junction.id) ? reverseSelections[junction.id] : corridorGroupId(junction, selections);
}

export function corridorTimelineRows(junction, selections = {}, reverseSelections = {}, direction = 'forward') {
  const forwardId = corridorGroupId(junction, selections);
  const reverseId = reverseCorridorGroupId(junction, selections, reverseSelections);
  if (direction === 'bidirectional' && reverseId !== forwardId) {
    return [{ groupId: forwardId, label: `Fwd G${forwardId}`, offset: -6 }, { groupId: reverseId, label: `Rev G${reverseId}`, offset: 6 }];
  }
  return [{ groupId: forwardId, label: `${direction === 'bidirectional' ? 'Fwd/Rev' : 'Fwd'} G${forwardId}`, offset: 0 }];
}

export function calculationGroupIds(junctions, selections = {}) {
  return Object.fromEntries(junctions.map(junction => {
    const { groupIds } = readProgram(junction);
    const id = corridorGroupId(junction, selections);
    if (id === null) throw new Error(`${junction.label}: choose a corridor group in the junction's Groups section.`);
    if (!groupIds.includes(id)) throw new Error(`${junction.label}: selected corridor group G${id} does not exist.`);
    return [junction.id, id];
  }));
}

export function corridorProgramError(junctions, selections = {}, reverseSelections = {}, direction = 'forward') {
  try {
    calculationGroupIds(junctions, selections);
    if (direction === 'bidirectional') calculationGroupIds(junctions, { ...selections, ...reverseSelections });
    return '';
  }
  catch (cause) { return cause.message; }
}

export function addProgramGroup(junction) {
  const { groupIds, phaseDurations } = readProgram(junction);
  const groupId = Math.max(...groupIds) + 1;
  if (!Number.isSafeInteger(groupId)) throw new Error('No signal group ID is available.');
  const next = structuredClone(junction);
  next.cycle.forEach((phase, index) => phase.signal_groups.push({ id: groupId, signals: [{ color: 'RED', duration: phaseDurations[index] }] }));
  return { junction: next, groupId };
}

export function removeProgramGroup(junction, groupId, selectedId, reverseSelectedId = selectedId) {
  const ids = programGroupIds(junction);
  if (ids.length <= 1) throw new Error('Keep at least one signal group.');
  if (groupId === selectedId || groupId === reverseSelectedId) throw new Error('Choose another corridor group before removing this group.');
  const next = structuredClone(junction);
  for (const phase of next.cycle) phase.signal_groups = phase.signal_groups.filter(group => group.id !== groupId);
  return next;
}

export function programTimeline(junction, groupId) {
  const segments = [];
  let duration = 0;
  for (const [phaseIndex, phase] of (junction?.cycle ?? []).entries()) {
    const group = phase.signal_groups.find(group => group.id === groupId);
    if (!group) throw new Error(`Group G${groupId} is missing in phase ${phaseIndex + 1}.`);
    for (const [signalIndex, signal] of group.signals.entries()) {
      if (!Number.isFinite(signal.duration) || signal.duration <= 0) throw new Error('Enter positive signal durations to preview the program.');
      const start = duration;
      duration += signal.duration;
      if (!Number.isFinite(duration)) throw new Error('Program duration exceeds the supported range.');
      segments.push({ ...signal, phaseId: phase.id, phaseIndex, groupId, signalIndex, start, end: duration });
    }
  }
  return { duration, segments };
}

export function programStateAt(timeline, time, offset = 0) {
  if (!timeline.duration || !Number.isFinite(time) || !Number.isFinite(offset)) return null;
  const remainder = (time - offset) % timeline.duration;
  const position = remainder < 0 ? remainder + timeline.duration : remainder;
  const segment = timeline.segments.find(segment => position >= segment.start && position < segment.end);
  return segment ? { position, segment } : null;
}
