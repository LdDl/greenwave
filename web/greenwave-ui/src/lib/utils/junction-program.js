export function signalColor(color) {
  return { GREEN: '#16a34a', RED: '#dc2626', YELLOW: '#ca8a04' }[color] ?? '#64748b';
}

export function programGroupIds(junction) {
  return [...new Set((junction?.cycle ?? []).flatMap(phase => phase.signal_groups.map(group => group.id)))];
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
