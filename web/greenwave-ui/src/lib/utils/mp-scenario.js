const record = (value) => value && typeof value === 'object' && !Array.isArray(value);

export function mpScenarios(project) {
	return project.mpScenarios ?? [{ id: 0, name: 'Default scenario', settings: {} }];
}

export function activeMPScenario(project) {
	return mpScenarios(project).find((item) => item.id === (project.activeMPScenarioId ?? 0));
}

export function validateMPScenarios(project) {
	if (Object.hasOwn(project, 'mpScenario'))
		throw new Error('The mpScenario field is not supported. Store named scenarios in mpScenarios.');
	if (project.mpScenarios === undefined) {
		if (project.activeMPScenarioId !== undefined)
			throw new Error('MP scenario selection needs scenarios.');
		return;
	}
	if (!Array.isArray(project.mpScenarios) || !project.mpScenarios.length)
		throw new Error('Use a non-empty MP scenario list.');
	const ids = new Set();
	const names = new Set();
	for (const item of project.mpScenarios) {
		if (!record(item) || !Number.isSafeInteger(item.id) || item.id < 0 || ids.has(item.id))
			throw new Error('MP scenarios need distinct non-negative integer IDs.');
		if (
			typeof item.name !== 'string' ||
			!item.name.trim() ||
			item.name.length > 120 ||
			names.has(item.name.trim().toLowerCase())
		)
			throw new Error('Use distinct MP scenario names of 1 to 120 characters.');
		validateMPScenario(item.settings);
		ids.add(item.id);
		names.add(item.name.trim().toLowerCase());
	}
	if (!ids.has(project.activeMPScenarioId)) throw new Error('Choose an existing MP scenario.');
}

// Drafts may have incomplete turning totals. Running a simulation validates those totals.
export function validateMPScenario(value) {
	if (!record(value)) throw new Error('Invalid MP scenario.');
	for (const key of ['alpha', 'deltaT', 'simTime', 'minGreenS', 'maxGreenS', 'clearanceS'])
		if (value[key] !== undefined && (!Number.isFinite(value[key]) || value[key] < 0))
			throw new Error(`Invalid MP setting: ${key}.`);
	if (
		value.storageModel !== undefined &&
		!['point_queue', 'finite_storage'].includes(value.storageModel)
	)
		throw new Error('Invalid MP storage model.');
	for (const key of ['boundaryNodeIds', 'coordinatedCorridorIds'])
		if (
			value[key] !== undefined &&
			(!Array.isArray(value[key]) || value[key].some((id) => !Number.isSafeInteger(id) || id < 0))
		)
			throw new Error(`Invalid MP selection: ${key}.`);
	for (const key of ['entryRates', 'turningRatios', 'saturationFlows', 'initialMovementQueues']) {
		if (value[key] === undefined) continue;
		if (
			!record(value[key]) ||
			Object.values(value[key]).some((number) => !Number.isFinite(number) || number < 0)
		)
			throw new Error(`Invalid MP values: ${key}.`);
	}
}
