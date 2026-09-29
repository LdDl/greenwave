const record = (value) => value && typeof value === 'object' && !Array.isArray(value);

// Drafts may have incomplete turning totals; running a simulation validates those totals.
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
