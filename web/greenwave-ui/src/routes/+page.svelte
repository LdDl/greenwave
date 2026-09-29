<script>
	import TimeSpaceDiagram from '../components/TimeSpaceDiagram.svelte';
	import RoadView from '../components/RoadView.svelte';
	import WaveStatus from '../components/WaveStatus.svelte';
	import OptimizationSummary from '../components/OptimizationSummary.svelte';
	import { calculationWindow, optimizationReport } from '$lib/utils/optimization-report.js';
	import { reverseGroupNotice } from '$lib/utils/wave-status.js';
	import EditSignalModal from '../components/EditSignalModal.svelte';
	import EditJunctionModal from '../components/EditJunctionModal.svelte';
	import {
		analysisMode,
		analysisInspectorOpen,
		corridorsOpen,
		optimizedCorridorName
	} from '$lib/stores/workspace.js';
	import { slide } from 'svelte/transition';
	import { isLoading, error, resetToDemo } from '$lib/stores';
	import { exportToJSON, prepareOutputExport } from '$lib/utils/export-import.js';
	import {
		junctions,
		desiredSpeed,
		desiredIntensity,
		desiredFlow,
		optimizationDirection
	} from '$lib/stores/core';
	import {
		corridorEditor,
		corridorProject,
		selectedCorridor,
		corridorIssue,
		corridorGraph,
		corridorGroupIds,
		corridorReverseGroupIds
	} from '$lib/stores/corridor.js';
	import {
		wavesAreOutdated,
		originalGreenWaves,
		originalThroughWaves,
		originalReverseGreenWaves,
		originalReverseThroughWaves,
		showGreenWaves,
		lastCalculatedSpeed,
		storeWaveCalculationPositions,
		actualFlow,
		actualIntensity,
		actualReverseFlow,
		actualReverseIntensity
	} from '$lib/stores/greenwave';
	import {
		optimizedGroupIds,
		optimizedReverseGroupIds,
		optimizedDirection,
		optimizedResultsAreOutdated,
		optimizedWaveCalculationPositions,
		optimizedLastCalculatedSpeed,
		optimizedInputRevision,
		lastOptimizationReport,
		optimizedJunctions,
		optimizedOffsets,
		optimizedGreenWaves,
		optimizedThroughWaves,
		optimizedReverseGreenWaves,
		optimizedReverseThroughWaves,
		actualFlowOptimized,
		actualIntensityOptimized,
		actualReverseFlowOptimized,
		actualReverseIntensityOptimized
	} from '$lib/stores/optimization';
	import { extractGreenWaves } from '$lib/api/greenwave.js';
	import { optimizeOffsets } from '$lib/api/optimize.js';
	import {
		prepareJunctionsForAPI,
		applyOffsetsToJunctions,
		validateJunctionCycles
	} from '$lib/utils/junction-helpers.js';
	import { onDestroy } from 'svelte';
	import { get } from 'svelte/store';
	import {
		corridorGroupId,
		corridorProgramError,
		programGroupIds
	} from '$lib/utils/junction-program.js';
	import { invalidateAll, validateInput, validateResults } from '$lib/stores/invalidation';

	onDestroy(() => corridorEditor.finish());
	const calculationRevision = corridorEditor.calculationRevision;

	function reveal(node) {
		return slide(node, {
			duration: window.matchMedia('(prefers-reduced-motion: reduce)').matches ? 0 : 180
		});
	}

	// Per-operation loading flags (global $isLoading stays for disabling buttons)
	let isExtracting = false;
	let isOptimizing = false;

	// View mode: 'diagram' | 'road'
	let inputDiagram;
	let resultDiagram;
	let linkedScroll = true;
	let viewMode = 'diagram';
	let inspectorMode = 'input';
	let overview = false;
	let chartZoom = 1;
	let timeZoom = 1;
	$: if ($analysisMode !== 'compare') inspectorMode = $analysisMode;
	$: corridorLength = Math.round(($junctions.at(-1)?.point.y ?? 0) - ($junctions[0]?.point.y ?? 0));

	// Signal modal state
	let selectedSignal = null;
	let selectedSignalContext = null;
	let isSignalModalOpen = false;

	// Junction modal state
	let selectedJunction = null;
	let selectedJunctionSignal = null;
	let isJunctionModalOpen = false;
	let isNewJunction = false;
	let junctionSaveError = '';

	// Reactive variables
	$: hasGreenWaveData = $originalGreenWaves.length > 0;
	$: hasResults = $optimizedJunctions.length > 0;
	$: offsetsMatchInput =
		hasResults &&
		$optimizedJunctions.length === $junctions.length &&
		$optimizedJunctions.every((result) =>
			$junctions.some((node) => node.id === result.id && (node.offset ?? 0) === result.offset)
		);
	$: canApplyOffsets =
		hasResults &&
		!$isLoading &&
		!$optimizedResultsAreOutdated.isOutdated &&
		$optimizedInputRevision === $calculationRevision &&
		!offsetsMatchInput;

	// Validation: check if all junctions have the same cycle duration
	$: cycleValidation =
		$junctions.length >= 2 ? validateJunctionCycles($junctions) : { isValid: true, durations: [] };
	$: programError = corridorProgramError(
		$junctions,
		$corridorGroupIds,
		$corridorReverseGroupIds,
		$optimizationDirection
	);
	$: hasValidationError = !!$corridorIssue || !!programError || !cycleValidation.isValid;
	$: validationErrorMessage =
		$corridorIssue ||
		programError ||
		(hasValidationError
			? `Different cycle durations: ${$junctions.map((j, i) => `${j.label}: ${cycleValidation.durations[i]}s`).join(', ')}`
			: '');
	$: inputWindow = !hasValidationError ? calculationWindow($junctions, $desiredSpeed) : null;

	$: isExtractDisabled =
		$isLoading ||
		$junctions.length < 2 ||
		hasValidationError ||
		!Number.isFinite($desiredSpeed) ||
		$desiredSpeed <= 0;

	// Extract green waves from API
	async function handleExtractWaves() {
		if (isExtractDisabled) return;
		const revision = get(corridorEditor.calculationRevision);
		try {
			isExtracting = true;
			isLoading.set(true);
			error.set(null);

			const junctionsForAPI = prepareJunctionsForAPI($junctions);
			const response = await extractGreenWaves(
				junctionsForAPI,
				$desiredSpeed,
				$optimizationDirection,
				$corridorGroupIds,
				$corridorReverseGroupIds
			);
			if (revision !== get(corridorEditor.calculationRevision))
				throw new Error(
					'Input changed during extraction. Extract waves again for the current corridor.'
				);
			originalGreenWaves.set(response.green_waves || []);
			originalThroughWaves.set(response.through_green_waves || []);
			originalReverseGreenWaves.set(response.reverse_green_waves || []);
			originalReverseThroughWaves.set(response.reverse_through_green_waves || []);
			showGreenWaves.set(true);
			storeWaveCalculationPositions($junctions, $desiredSpeed);
			validateInput();
		} catch (apiError) {
			error.set(apiError.message || 'Failed to extract green waves');
			console.error('API Error:', apiError);
		} finally {
			isExtracting = false;
			isLoading.set(false);
		}
	}

	function updateJunction(event) {
		const { id, newDistance } = event.detail;
		junctions.update((list) =>
			list.map((junction) =>
				junction.id === id
					? { ...junction, point: { ...junction.point, y: newDistance } }
					: junction
			)
		);
	}

	// Handle optimization
	async function handleOptimize() {
		if (isExtractDisabled) return;
		const revision = get(corridorEditor.calculationRevision);
		try {
			isOptimizing = true;
			isLoading.set(true);
			error.set(null);

			const input = structuredClone($junctions);
			const junctionsForAPI = prepareJunctionsForAPI(input);
			const [before, response] = await Promise.all([
				extractGreenWaves(
					junctionsForAPI,
					$desiredSpeed,
					$optimizationDirection,
					$corridorGroupIds,
					$corridorReverseGroupIds
				),
				optimizeOffsets(
					junctionsForAPI,
					$desiredSpeed,
					'genetic',
					{},
					$optimizationDirection,
					$corridorGroupIds,
					$corridorReverseGroupIds
				)
			]);
			if (revision !== get(corridorEditor.calculationRevision))
				throw new Error('Input changed during optimization. Optimize the current corridor again.');
			const nextJunctions = applyOffsetsToJunctions(input, response.best_offsets);
			const nextOffsets = nextJunctions.map((junction) => junction.offset);
			const report = optimizationReport(
				input,
				nextJunctions,
				before,
				response,
				$optimizationDirection
			);

			originalGreenWaves.set(before.green_waves || []);
			originalThroughWaves.set(before.through_green_waves || []);
			originalReverseGreenWaves.set(before.reverse_green_waves || []);
			originalReverseThroughWaves.set(before.reverse_through_green_waves || []);
			showGreenWaves.set(true);
			storeWaveCalculationPositions(input, $desiredSpeed);
			validateInput();
			lastOptimizationReport.set(report);

			optimizedCorridorName.set($selectedCorridor.name);
			optimizedOffsets.set(nextOffsets);
			optimizedJunctions.set(nextJunctions);
			optimizedGroupIds.set(structuredClone($corridorGroupIds));
			optimizedReverseGroupIds.set(structuredClone($corridorReverseGroupIds));
			optimizedDirection.set($optimizationDirection);
			optimizedGreenWaves.set(response.green_waves || []);
			optimizedThroughWaves.set(response.through_green_waves || []);
			optimizedReverseGreenWaves.set(response.reverse_green_waves || []);
			optimizedReverseThroughWaves.set(response.reverse_through_green_waves || []);

			optimizedWaveCalculationPositions.set($junctions.map((j) => ({ id: j.id, y: j.point.y })));
			optimizedLastCalculatedSpeed.set($desiredSpeed);
			optimizedInputRevision.set(revision);
			validateResults();
			inspectorMode = 'results';
			analysisMode.set('compare');
		} catch (optimizeError) {
			error.set(optimizeError.message || 'Failed to optimize');
			console.error('Optimize Error:', optimizeError);
		} finally {
			isOptimizing = false;
			isLoading.set(false);
		}
	}

	function clearResults() {
		lastOptimizationReport.set(null);
		optimizedCorridorName.set('');
		analysisMode.set('input');
		optimizedJunctions.set([]);
		optimizedGroupIds.set({});
		optimizedReverseGroupIds.set({});
		optimizedDirection.set('forward');
		optimizedOffsets.set([]);
		optimizedGreenWaves.set([]);
		optimizedThroughWaves.set([]);
		optimizedReverseGreenWaves.set([]);
		optimizedReverseThroughWaves.set([]);
		optimizedInputRevision.set(null);
		optimizedLastCalculatedSpeed.set(null);
		optimizedWaveCalculationPositions.set([]);
		validateResults();
	}

	function applyOptimizedOffsets() {
		if (!canApplyOffsets) return;
		try {
			const applied = corridorEditor.applyOffsets($optimizedJunctions, $optimizedInputRevision);
			if (!applied) return;
			originalGreenWaves.set(structuredClone($optimizedGreenWaves));
			originalThroughWaves.set(structuredClone($optimizedThroughWaves));
			originalReverseGreenWaves.set(structuredClone($optimizedReverseGreenWaves));
			originalReverseThroughWaves.set(structuredClone($optimizedReverseThroughWaves));
			storeWaveCalculationPositions($junctions, $optimizedLastCalculatedSpeed);
			optimizedInputRevision.set($calculationRevision);
			showGreenWaves.set(true);
			validateInput();
			validateResults();
			error.set(null);
		} catch (cause) {
			error.set(cause.message);
		}
	}

	function saveSignal(e) {
		const updatedSignal = e.detail.signal;

		if (!selectedSignalContext) {
			console.error('Invalid selectedSignalContext:', selectedSignalContext);
			return;
		}

		junctions.update((junctionList) => {
			const updatedJunctions = junctionList.map((junction) => {
				if (junction.id === selectedSignalContext.junctionId) {
					return {
						...junction,
						cycle: junction.cycle.map((phase) => {
							if (phase.id === selectedSignalContext.phaseId) {
								return {
									...phase,
									signal_groups: phase.signal_groups.map((sg) => ({
										...sg,
										signals: sg.signals.map((signal, index) => {
											if (
												sg.id === selectedSignalContext.groupId &&
												index === selectedSignalContext.signalIndex
											) {
												return { ...signal, ...updatedSignal };
											}
											return signal;
										})
									}))
								};
							}
							return phase;
						})
					};
				}
				return junction;
			});
			return updatedJunctions;
		});

		invalidateAll('signal changes');
		closeSignalModal();
	}

	function openSignalModal(event) {
		const { junction, phase, signal } = event.detail;
		const group = phase.signal_groups.find((group) => group.signals.includes(signal));
		if (programGroupIds(junction).length > 1) {
			openJunctionModal({ detail: { junction } });
			selectedJunctionSignal = {
				groupId: group.id,
				phaseId: phase.id,
				signalIndex: group.signals.indexOf(signal)
			};
			return;
		}
		selectedSignalContext = {
			junctionId: junction.id,
			phaseId: phase.id,
			groupId: group.id,
			signalIndex: group.signals.indexOf(signal)
		};
		selectedSignal = structuredClone(signal);
		isSignalModalOpen = true;
	}

	function closeSignalModal() {
		selectedSignal = null;
		isSignalModalOpen = false;
	}

	function openJunctionModal(event) {
		junctionSaveError = '';
		const { junction } = event.detail;
		const originalJunction = $junctions.find((j) => j.id === junction.id);
		selectedJunction = originalJunction || junction;
		selectedJunctionSignal = null;
		isNewJunction = false;
		isJunctionModalOpen = true;
	}

	function openNewJunctionModal() {
		junctionSaveError = '';
		const maxId = Math.max(-1, ...$corridorProject.junctions.map((j) => j.id));
		const maxY = $junctions.length > 0 ? Math.max(...$junctions.map((j) => j.point.y)) : -100;

		selectedJunction = {
			id: maxId + 1,
			label: `Junction ${maxId + 2}`,
			cycle: [
				{
					id: (maxId + 1) * 10,
					signal_groups: [
						{
							id: 0,
							signals: [
								{ duration: 30, color: 'GREEN' },
								{ duration: 20, color: 'RED' }
							]
						}
					]
				}
			],
			offset: 0,
			point: { x: $junctions[0]?.point.x ?? 0, y: maxY + 150 }
		};
		isNewJunction = true;
		isJunctionModalOpen = true;
	}

	function saveJunction(event) {
		const { junction, isNew } = event.detail;
		try {
			corridorEditor.saveJunction(
				junction,
				isNew,
				event.detail.groupId,
				event.detail.reverseGroupId,
				event.detail.assignments
			);
			invalidateAll('junction configuration changed');
			closeJunctionModal();
		} catch (cause) {
			junctionSaveError = cause.message;
		}
	}

	function deleteJunction(event) {
		const { junction } = event.detail;
		corridorEditor.removeJunction(junction.id);
		invalidateAll('junction deleted');
		closeJunctionModal();
	}

	function closeJunctionModal() {
		junctionSaveError = '';
		selectedJunction = null;
		selectedJunctionSignal = null;
		isJunctionModalOpen = false;
		isNewJunction = false;
	}

	function exportResult() {
		exportToJSON(
			prepareOutputExport(
				$optimizedJunctions,
				$optimizedLastCalculatedSpeed ?? $desiredSpeed,
				$desiredIntensity,
				$optimizedDirection,
				$optimizedGroupIds,
				$optimizedReverseGroupIds
			),
			'greenwave-result.json'
		);
	}
</script>

{#if isSignalModalOpen}
	<EditSignalModal signal={selectedSignal} on:save={saveSignal} on:close={closeSignalModal} />
{/if}

{#if isJunctionModalOpen}
	<EditJunctionModal
		junction={selectedJunction}
		isNew={isNewJunction}
		graph={$corridorGraph}
		corridorGroup={corridorGroupId(selectedJunction, $corridorGroupIds)}
		reverseCorridorGroup={$corridorReverseGroupIds[selectedJunction.id] ?? null}
		direction={$optimizationDirection}
		assignments={$corridorProject.movements[selectedJunction.id] ?? {}}
		saveError={junctionSaveError}
		corridorMovements={corridorEditor.routeMovements(selectedJunction.id)}
		initialSignal={selectedJunctionSignal}
		on:save={saveJunction}
		on:delete={deleteJunction}
		on:close={closeJunctionModal}
	/>
{/if}

<svelte:head><title>Coordination | Greenwave</title></svelte:head>
<div class="analysis-workspace">
	<header class="analysis-heading">
		<div class="analysis-route">
			<p class="ui-eyebrow">Active corridor</p>
			<h1>{$selectedCorridor.name}</h1>
			<span>{$junctions.length} junctions · {corridorLength} m</span>
		</div>
		<button class="ui-button" on:click={() => corridorsOpen.set(true)}>Change corridor</button>
		<div class="analysis-mode" role="group" aria-label="Analysis view">
			<button aria-pressed={$analysisMode === 'input'} on:click={() => analysisMode.set('input')}
				>Input</button
			>
			<button
				aria-pressed={$analysisMode === 'results'}
				on:click={() => analysisMode.set('results')}
				>Results{#if hasResults}<span
						class="result-dot"
						class:outdated={$optimizedResultsAreOutdated.isOutdated}
					></span>{/if}</button
			>
			<button
				aria-pressed={$analysisMode === 'compare'}
				disabled={!hasResults}
				title="Compare input with the optimization result"
				on:click={() => analysisMode.set('compare')}>Compare</button
			>
		</div>
		<div class="analysis-actions">
			<button class="ui-button" on:click={openNewJunctionModal}>+ Add junction</button>
			<button class="ui-button ui-accent" on:click={handleExtractWaves} disabled={isExtractDisabled}
				>{isExtracting ? 'Extracting...' : 'Extract waves'}</button
			>
			<button class="ui-button ui-primary" on:click={handleOptimize} disabled={isExtractDisabled}
				>{isOptimizing ? 'Optimizing...' : 'Optimize'}</button
			>
			<button
				class="ui-button details-toggle"
				aria-label={$analysisInspectorOpen ? 'Hide details' : 'Settings & metrics'}
				aria-pressed={$analysisInspectorOpen}
				on:click={() => analysisInspectorOpen.update((value) => !value)}
				>{$analysisInspectorOpen ? 'Hide details' : 'Settings & metrics'}</button
			>
		</div>
	</header>
	{#if $error}<div class="analysis-alert" role="alert">
			<span>{$error}</span><button aria-label="Dismiss error" on:click={() => error.set(null)}
				>×</button
			>
		</div>{/if}
	{#if hasValidationError}<div class="analysis-warning" role="alert">
			<strong>Input needs attention.</strong>
			{validationErrorMessage}
		</div>{/if}
	{#if inputWindow?.exceedsCycle}<div class="analysis-warning" role="status">
			One-cycle calculation: travel takes {Number(inputWindow.travel.toFixed(1))} s, cycle {inputWindow.cycle}
			s. A full-corridor wave spans multiple cycles and is not supported by the current calculation.
			Optimize can improve partial waves.
		</div>{/if}
	{#if hasResults && $optimizedResultsAreOutdated.isOutdated}<div
			class="analysis-warning"
			role="status"
		>
			Result for <strong>{$optimizedCorridorName || 'the previous input'}</strong> is outdated. {$optimizedResultsAreOutdated.reason}
		</div>{/if}
	<div class="analysis-body" class:details-hidden={!$analysisInspectorOpen}>
		<div class="analysis-main">
			<div class="diagram-options">
				<label
					>View <select aria-label="Visualization" bind:value={viewMode}
						><option value="diagram">Time-distance diagram</option><option value="road"
							>Signal strip preview</option
						></select
					></label
				>
				{#if viewMode === 'diagram'}
					<div class="diagram-zoom" role="group" aria-label="Distance scale">
						<span>Spacing</span><button
							class="ui-button"
							aria-label="Decrease diagram spacing"
							disabled={chartZoom <= 0.75}
							on:click={() => {
								overview = false;
								chartZoom = Math.max(0.75, chartZoom - 0.25);
							}}>−</button
						><span>{Math.round(chartZoom * 100)}%</span><button
							class="ui-button"
							aria-label="Increase diagram spacing"
							disabled={chartZoom >= 3}
							on:click={() => {
								overview = false;
								chartZoom = Math.min(3, chartZoom + 0.25);
							}}>+</button
						>
					</div>
					<button class="ui-button" aria-pressed={overview} on:click={() => (overview = !overview)}
						>{overview ? 'Readable spacing' : 'Fit overview'}</button
					>
					<label
						>Time <select aria-label="Time scale" bind:value={timeZoom}
							><option value={1}>1×</option><option value={1.5}>1.5×</option><option value={2}
								>2×</option
							><option value={3}>3×</option></select
						></label
					>
				{/if}
				{#if $analysisMode === 'compare' && viewMode === 'diagram'}<label
						><input type="checkbox" bind:checked={linkedScroll} />Link scrolling</label
					>{/if}
				<label class="waves-toggle"
					><input type="checkbox" bind:checked={$showGreenWaves} disabled={!hasGreenWaveData} />Show
					input waves</label
				>
			</div>
			{#if hasResults && $lastOptimizationReport && $analysisMode !== 'input'}
				<OptimizationSummary
					report={$lastOptimizationReport}
					outdated={$optimizedResultsAreOutdated.isOutdated}
					matchesInput={offsetsMatchInput}
				/>
			{/if}
			{#if viewMode === 'road'}<p class="preview-note">
					Static signal strips at t=0, including offsets. This view does not animate traffic or
					phase switching.
				</p>{/if}
			<div class="analysis-charts" class:comparing={$analysisMode === 'compare'}>
				{#if $analysisMode !== 'results'}
					<section class="diagram-card" aria-label="Input diagram">
						<header>
							<div>
								<h2>Input configuration</h2>
								<span>{$selectedCorridor.name}</span>
							</div>
							<span class="diagram-state"
								>{$wavesAreOutdated.isOutdated
									? 'Outdated waves'
									: $lastCalculatedSpeed !== null
										? 'Calculated'
										: 'Editable input'}</span
							>
						</header>
						<div class="diagram-surface">
							{#if !$junctions.length}
								<div class="analysis-empty">
									<span class="empty-symbol" aria-hidden="true">▥</span>
									<h3>
										{$corridorProject.junctions.length
											? 'Choose the corridor route'
											: 'Start with a simple corridor'}
									</h3>
									<p>
										{$corridorProject.junctions.length
											? 'Add existing network junctions to this corridor, or create a new junction.'
											: 'Load four junctions to explore programs and green waves, or build your own network.'}
									</p>
									<div class="ui-actions">
										{#if $corridorProject.junctions.length}<a
												class="ui-button ui-primary"
												href="/network">Choose route on network</a
											>{:else}<button class="ui-button ui-primary" on:click={resetToDemo}
												>Load demo data</button
											>{/if}<button class="ui-button" on:click={openNewJunctionModal}
											>+ Add first junction</button
										>
									</div>
								</div>
							{:else if viewMode === 'diagram'}
								<TimeSpaceDiagram
									bind:this={inputDiagram}
									on:viewportScroll={(event) => {
										if ($analysisMode === 'compare' && linkedScroll)
											resultDiagram?.scrollToViewport(event.detail);
									}}
									junctions={$junctions}
									groupIds={$corridorGroupIds}
									reverseGroupIds={$corridorReverseGroupIds}
									direction={$optimizationDirection}
									wavesAreOutdated={$wavesAreOutdated}
									interactive={true}
									greenWaves={$originalGreenWaves}
									throughWaves={$originalThroughWaves}
									reverseGreenWaves={$originalReverseGreenWaves}
									reverseThroughWaves={$originalReverseThroughWaves}
									showWaves={$showGreenWaves}
									{overview}
									zoom={chartZoom}
									{timeZoom}
									on:updateJunction={updateJunction}
									on:dragStart={() => corridorEditor.begin()}
									on:dragEnd={() => corridorEditor.finish()}
									on:editSignal={openSignalModal}
									on:editJunction={openJunctionModal}
								/>
							{:else}<RoadView
									junctions={$junctions}
									groupIds={$corridorGroupIds}
									reverseGroupIds={$corridorReverseGroupIds}
									direction={$optimizationDirection}
								/>{/if}
						</div>
						<footer>
							{#if $wavesAreOutdated.isOutdated}<span class="text-amber-700"
									>{$wavesAreOutdated.reason}. Extract waves again.</span
								>{:else}Drag a junction to change distance. Click its name or signal to edit.{/if}<span
								>Scroll to explore · distance remains proportional</span
							>
						</footer>
					</section>
				{/if}
				{#if $analysisMode !== 'input'}
					<section class="diagram-card" aria-label="Result diagram">
						<header>
							<div>
								<h2>Optimized results</h2>
								<span>{$optimizedCorridorName || $selectedCorridor.name}</span>
							</div>
							{#if hasResults}<div class="result-actions">
									<button
										class="ui-button ui-accent"
										on:click={applyOptimizedOffsets}
										disabled={!canApplyOffsets}
										title={$optimizedResultsAreOutdated.isOutdated
											? 'Optimize again for the current input'
											: offsetsMatchInput
												? 'Input already uses these offsets'
												: 'Apply only offsets as one Undo step'}>Apply offsets</button
									><button class="ui-button" on:click={exportResult}>Export result</button><button
										class="ui-button"
										on:click={clearResults}>Clear</button
									>
								</div>{/if}
						</header>
						<div class="diagram-surface">
							{#if !hasResults}<div class="analysis-empty">
									<span class="empty-symbol" aria-hidden="true">↗</span>
									<h3>No optimization result yet</h3>
									<p>
										Optimize this corridor to compare its programs and green waves with the input.
										The result keeps its own program snapshot.
									</p>
									<button
										class="ui-button ui-primary"
										disabled={isExtractDisabled}
										on:click={handleOptimize}>Optimize corridor</button
									>
								</div>
							{:else if viewMode === 'diagram'}<TimeSpaceDiagram
									bind:this={resultDiagram}
									on:viewportScroll={(event) => {
										if ($analysisMode === 'compare' && linkedScroll)
											inputDiagram?.scrollToViewport(event.detail);
									}}
									junctions={$optimizedJunctions}
									groupIds={$optimizedGroupIds}
									reverseGroupIds={$optimizedReverseGroupIds}
									direction={$optimizedDirection}
									wavesAreOutdated={$optimizedResultsAreOutdated}
									greenWaves={$optimizedGreenWaves}
									throughWaves={$optimizedThroughWaves}
									reverseGreenWaves={$optimizedReverseGreenWaves}
									reverseThroughWaves={$optimizedReverseThroughWaves}
									showWaves={true}
									showOffsets={true}
									interactive={false}
									{overview}
									zoom={chartZoom}
									{timeZoom}
								/>
							{:else}<RoadView
									junctions={$optimizedJunctions}
									groupIds={$optimizedGroupIds}
									reverseGroupIds={$optimizedReverseGroupIds}
									direction={$optimizedDirection}
									showOffsets={true}
								/>{/if}
						</div>
						<footer>
							<span
								>{hasResults
									? offsetsMatchInput
										? 'Input already uses these offsets.'
										: 'Apply offsets to update the shared network. Programs remain unchanged.'
									: 'Results will appear here after optimization.'}</span
							><span>Read-only result</span>
						</footer>
					</section>
				{/if}
			</div>
		</div>
		{#if $analysisInspectorOpen}
			<aside class="analysis-inspector" aria-label="Settings and analytical indicators">
				<header>
					<div>
						<p class="ui-eyebrow">
							{inspectorMode === 'input' ? 'Active corridor' : 'Calculation snapshot'}
						</p>
						<h2>{inspectorMode === 'input' ? 'Settings & metrics' : 'Result metrics'}</h2>
					</div>
					{#if $analysisMode === 'compare'}<div class="inspector-switch">
							<button
								aria-pressed={inspectorMode === 'input'}
								on:click={() => (inspectorMode = 'input')}>Input</button
							><button
								aria-pressed={inspectorMode === 'results'}
								on:click={() => (inspectorMode = 'results')}>Result</button
							>
						</div>{/if}
				</header>
				<div class="analysis-inspector-scroll">
					{#if inspectorMode === 'input'}
						<!-- svelte-ignore a11y_no_noninteractive_tabindex -->
						<section
							aria-label="Input settings and indicators"
							tabindex="0"
							class="corridor-controls space-y-4"
						>
							<div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
								<div>
									<label for="input-direction" class="mb-2 block text-sm font-medium"
										>Optimization direction</label
									>
									<select
										id="input-direction"
										bind:value={$optimizationDirection}
										class="w-full rounded-md border px-3 py-2"
									>
										<option value="forward">Forward only</option>
										<option value="bidirectional">Bidirectional</option>
									</select>
								</div>
								<div>
									<label for="input-desired-speed" class="mb-2 block text-sm font-medium"
										>Desired speed (km/h)</label
									>
									<input
										id="input-desired-speed"
										type="number"
										bind:value={$desiredSpeed}
										class="w-full rounded-md border px-3 py-2"
										min="10"
										max="100"
									/>
								</div>
							</div>

							<!-- Junction status -> own line so it never shifts the grid above -->
							<div class="text-sm leading-snug">
								<span class="font-medium text-gray-700">{$junctions.length} junctions</span>
								{#if $junctions.length === 0}
									<span class="text-blue-600"> · load an example or add a junction</span>
								{:else if $junctions.length === 1}
									<span class="text-orange-600"> -> add at least 1 more to extract waves</span>
								{:else if hasValidationError}
									<span class="mt-0.5 block text-red-600">⚠ {validationErrorMessage}</span>
								{:else if $lastCalculatedSpeed !== null && !$wavesAreOutdated.isOutdated}
									<span class="text-gray-600"> · calculation complete</span>
								{:else}
									<span class="text-orange-600"> -> press "Extract waves" to calculate</span>
								{/if}
							</div>

							<div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
								<div>
									<label for="input-desired-intensity" class="mb-2 block text-sm font-medium"
										>Desired intensity (veh/h)</label
									>
									<input
										id="input-desired-intensity"
										type="number"
										bind:value={$desiredIntensity}
										class="w-full rounded-md border px-3 py-2"
										min="0"
									/>
								</div>
								<div>
									<label for="input-desired-flow" class="mb-2 block text-sm font-medium"
										>Desired flow (veh/s)</label
									>
									<input
										id="input-desired-flow"
										type="number"
										value={Number.isFinite($desiredFlow) ? Number($desiredFlow.toFixed(6)) : ''}
										class="w-full rounded-md border px-3 py-2"
										min="0"
										step="0.5"
										on:input={(e) => desiredIntensity.set(e.target.value * 3600)}
									/>
								</div>
							</div>

							<div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
								<div>
									<span class="mb-2 block text-sm font-medium"
										>Actual intensity{#if $optimizationDirection === 'bidirectional'}
											<span class="text-green-600">(fwd)</span>{/if}</span
									>
									<div
										class="w-full rounded-md border bg-gray-100 px-3 py-2 text-gray-700 tabular-nums"
									>
										{($actualIntensity || 0).toFixed(2)} veh/h
										{#if $wavesAreOutdated.isOutdated}<span class="text-xs text-orange-500"
												>(outdated)</span
											>{/if}
									</div>
								</div>
								<div>
									<span class="mb-2 block text-sm font-medium"
										>Actual flow{#if $optimizationDirection === 'bidirectional'}
											<span class="text-green-600">(fwd)</span>{/if}</span
									>
									<div
										class="w-full rounded-md border bg-gray-100 px-3 py-2 text-gray-700 tabular-nums"
									>
										{($actualFlow || 0).toFixed(6)} veh/s
										{#if $wavesAreOutdated.isOutdated}<span class="text-xs text-orange-500"
												>(outdated)</span
											>{/if}
									</div>
								</div>
							</div>

							<WaveStatus
								junctionCount={$junctions.length}
								waves={$originalGreenWaves}
								throughWaves={$originalThroughWaves}
								calculated={$lastCalculatedSpeed !== null}
								outdated={$wavesAreOutdated.isOutdated}
							/>

							{#if $optimizationDirection === 'bidirectional'}
								<div transition:reveal class="space-y-2" data-direction-panel="input-reverse">
									<WaveStatus
										direction="Reverse"
										junctionCount={$junctions.length}
										waves={$originalReverseGreenWaves}
										throughWaves={$originalReverseThroughWaves}
										calculated={$lastCalculatedSpeed !== null}
										outdated={$wavesAreOutdated.isOutdated}
										notice={reverseGroupNotice(
											$junctions,
											$corridorGroupIds,
											$corridorReverseGroupIds
										)}
									/>
									<div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
										<div>
											<span class="mb-2 block text-sm font-medium"
												>Actual intensity <span class="text-blue-600">(rev)</span></span
											>
											<div
												class="w-full rounded-md border bg-gray-100 px-3 py-2 text-gray-700 tabular-nums"
											>
												{($actualReverseIntensity || 0).toFixed(2)} veh/h
												{#if $wavesAreOutdated.isOutdated}<span class="text-xs text-orange-500"
														>(outdated)</span
													>{/if}
											</div>
										</div>
										<div>
											<span class="mb-2 block text-sm font-medium"
												>Actual flow <span class="text-blue-600">(rev)</span></span
											>
											<div
												class="w-full rounded-md border bg-gray-100 px-3 py-2 text-gray-700 tabular-nums"
											>
												{($actualReverseFlow || 0).toFixed(6)} veh/s
												{#if $wavesAreOutdated.isOutdated}<span class="text-xs text-orange-500"
														>(outdated)</span
													>{/if}
											</div>
										</div>
									</div>
								</div>
							{/if}
						</section>
					{:else}
						<!-- svelte-ignore a11y_no_noninteractive_tabindex -->
						<section
							aria-label="Result settings and indicators"
							tabindex="0"
							class="corridor-controls space-y-4"
						>
							<div class="grid grid-cols-2 gap-3">
								<label class="text-xs text-gray-600"
									>Calculation direction<input
										aria-label="Result direction"
										readonly
										value={$optimizedDirection === 'bidirectional' ? 'Bidirectional' : 'Forward'}
										class="mt-1 w-full rounded border bg-gray-50 px-2 py-2 text-sm"
									/></label
								>
								<label class="text-xs text-gray-600"
									>Calculated speed (km/h)<input
										aria-label="Result speed"
										readonly
										value={$optimizedLastCalculatedSpeed ?? ''}
										class="mt-1 w-full rounded border bg-gray-50 px-2 py-2 text-sm"
									/></label
								>
							</div>
							<!-- Optimization status -> own line, wraps freely -->
							<div class="text-sm leading-snug">
								{#if $optimizedJunctions.length > 0}
									<span class="font-medium text-gray-700"
										>{$optimizedJunctions.length} junctions optimized</span
									>
									{#if $optimizedResultsAreOutdated.isOutdated}
										<span class="mt-0.5 block text-orange-600"
											>⚠ {$optimizedResultsAreOutdated.reason}</span
										>
									{:else}
										<span class="text-green-600" role="status"
											>{offsetsMatchInput
												? ' · offsets match input'
												: ' · offsets ready to apply'}</span
										>
									{/if}
								{:else}
									<span class="text-gray-400">No results yet -> press Optimize</span>
								{/if}
							</div>

							<div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
								<div>
									<label for="opt-desired-intensity" class="mb-2 block text-sm font-medium"
										>Desired intensity (veh/h)</label
									>
									<input
										id="opt-desired-intensity"
										type="number"
										bind:value={$desiredIntensity}
										class="w-full rounded-md border px-3 py-2"
										min="0"
									/>
								</div>
								<div>
									<label for="opt-desired-flow" class="mb-2 block text-sm font-medium"
										>Desired flow (veh/s)</label
									>
									<input
										id="opt-desired-flow"
										type="number"
										value={Number.isFinite($desiredFlow) ? Number($desiredFlow.toFixed(6)) : ''}
										class="w-full rounded-md border px-3 py-2"
										min="0"
										step="0.5"
										on:input={(e) => desiredIntensity.set(e.target.value * 3600)}
									/>
								</div>
							</div>

							<div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
								<div>
									<span class="mb-2 block text-sm font-medium"
										>Actual intensity{#if $optimizedDirection === 'bidirectional'}
											<span class="text-green-600">(fwd)</span>{/if}</span
									>
									<div
										class="w-full rounded-md border bg-gray-100 px-3 py-2 text-gray-700 tabular-nums"
									>
										{($actualIntensityOptimized || 0).toFixed(2)} veh/h
										{#if hasResults && $optimizedResultsAreOutdated.isOutdated}<span
												class="text-xs text-orange-500">(outdated)</span
											>{/if}
									</div>
								</div>
								<div>
									<span class="mb-2 block text-sm font-medium"
										>Actual flow{#if $optimizedDirection === 'bidirectional'}
											<span class="text-green-600">(fwd)</span>{/if}</span
									>
									<div
										class="w-full rounded-md border bg-gray-100 px-3 py-2 text-gray-700 tabular-nums"
									>
										{$actualFlowOptimized.toFixed(6)} veh/s
										{#if hasResults && $optimizedResultsAreOutdated.isOutdated}<span
												class="text-xs text-orange-500">(outdated)</span
											>{/if}
									</div>
								</div>
							</div>

							{#if hasResults}
								<WaveStatus
									junctionCount={$optimizedJunctions.length}
									waves={$optimizedGreenWaves}
									throughWaves={$optimizedThroughWaves}
									calculated={hasResults}
									outdated={$optimizedResultsAreOutdated.isOutdated}
								/>
							{/if}

							{#if $optimizedDirection === 'bidirectional'}
								<div transition:reveal class="space-y-2" data-direction-panel="result-reverse">
									<WaveStatus
										direction="Reverse"
										junctionCount={$optimizedJunctions.length}
										waves={$optimizedReverseGreenWaves}
										throughWaves={$optimizedReverseThroughWaves}
										calculated={hasResults}
										outdated={$optimizedResultsAreOutdated.isOutdated}
										notice={reverseGroupNotice(
											$optimizedJunctions,
											$optimizedGroupIds,
											$optimizedReverseGroupIds
										)}
									/>
									<div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
										<div>
											<span class="mb-2 block text-sm font-medium"
												>Actual intensity <span class="text-blue-600">(rev)</span></span
											>
											<div
												class="w-full rounded-md border bg-gray-100 px-3 py-2 text-gray-700 tabular-nums"
											>
												{($actualReverseIntensityOptimized || 0).toFixed(2)} veh/h
												{#if hasResults && $optimizedResultsAreOutdated.isOutdated}<span
														class="text-xs text-orange-500">(outdated)</span
													>{/if}
											</div>
										</div>
										<div>
											<span class="mb-2 block text-sm font-medium"
												>Actual flow <span class="text-blue-600">(rev)</span></span
											>
											<div
												class="w-full rounded-md border bg-gray-100 px-3 py-2 text-gray-700 tabular-nums"
											>
												{$actualReverseFlowOptimized.toFixed(6)} veh/s
												{#if hasResults && $optimizedResultsAreOutdated.isOutdated}<span
														class="text-xs text-orange-500">(outdated)</span
													>{/if}
											</div>
										</div>
									</div>
								</div>
							{/if}
						</section>
					{/if}
				</div>
			</aside>
		{/if}
	</div>
</div>

<style>
	.analysis-workspace {
		height: 100%;
		min-height: 0;
		display: flex;
		flex-direction: column;
	}
	.analysis-heading {
		display: flex;
		align-items: center;
		flex-wrap: wrap;
		gap: 10px;
		padding: 12px 20px;
		background: #fff;
		border-bottom: 1px solid #dce3ec;
		flex: none;
	}
	.analysis-route {
		margin-right: 4px;
	}
	.analysis-route h1 {
		display: inline;
		font-size: 15px;
		font-weight: 650;
	}
	.analysis-route > span {
		display: block;
		font-size: 11px;
		color: #64748b;
		margin-top: 3px;
	}
	.analysis-mode {
		display: flex;
		padding: 3px;
		border: 1px solid #e2e8f0;
		border-radius: 8px;
		gap: 3px;
		margin-left: auto;
	}
	.analysis-mode button {
		display: flex;
		align-items: center;
		gap: 6px;
		font-size: 12px;
		padding: 7px 12px;
		border-radius: 5px;
		color: #64748b;
	}
	.analysis-mode button[aria-pressed='true'] {
		background: #0f766e;
		color: white;
	}
	.analysis-mode button:disabled {
		opacity: 0.4;
	}
	.result-dot {
		width: 6px;
		height: 6px;
		border-radius: 50%;
		background: #10b981;
	}
	.result-dot.outdated {
		background: #f59e0b;
	}
	.analysis-actions {
		display: flex;
		flex-wrap: wrap;
		gap: 6px;
	}
	.analysis-alert,
	.analysis-warning {
		padding: 9px 20px;
		font-size: 12px;
		line-height: 1.5;
		flex: none;
		max-height: 120px;
		overflow-y: auto;
	}
	.analysis-alert {
		display: flex;
		justify-content: space-between;
		background: #fff1f2;
		color: #9f1239;
	}
	.analysis-warning {
		background: #fffbeb;
		color: #92400e;
		border-bottom: 1px solid #fde68a;
	}
	.analysis-body {
		display: grid;
		grid-template-columns: minmax(0, 1fr) 310px;
		min-height: 0;
		flex: 1;
	}
	.analysis-body.details-hidden {
		grid-template-columns: minmax(0, 1fr);
	}
	.analysis-main {
		min-width: 0;
		min-height: 0;
		display: flex;
		flex-direction: column;
		padding: 12px 16px 16px;
		gap: 10px;
	}
	.diagram-options {
		display: flex;
		align-items: center;
		flex-wrap: wrap;
		gap: 8px;
		flex: none;
		font-size: 11px;
		color: #64748b;
	}
	.diagram-options label {
		display: inline-flex;
		gap: 6px;
		align-items: center;
	}
	.diagram-options select {
		font-size: 11px;
		padding: 5px 27px 5px 8px;
		border: 1px solid #d7e0e8;
		border-radius: 6px;
		color: #334155;
	}
	.diagram-options .ui-button {
		min-height: 29px;
		padding: 4px 8px;
		font-size: 11px;
	}
	.diagram-zoom {
		display: flex;
		align-items: center;
		gap: 6px;
	}
	.diagram-zoom span {
		min-width: 28px;
		text-align: center;
	}
	.waves-toggle {
		margin-left: auto;
		color: #475569;
	}
	.waves-toggle input {
		width: 14px;
		height: 14px;
		border-radius: 3px;
	}
	.preview-note {
		font-size: 12px;
		padding: 8px 12px;
		background: #e2e8f0;
		border-radius: 6px;
		color: #475569;
	}
	.analysis-charts {
		display: grid;
		grid-template-columns: minmax(0, 1fr);
		flex: 1;
		min-height: 0;
		gap: 12px;
	}
	.analysis-charts.comparing {
		grid-template-columns: repeat(2, minmax(0, 1fr));
	}
	.diagram-card {
		display: flex;
		flex-direction: column;
		min-width: 0;
		min-height: 0;
		background: #fff;
		border: 1px solid #dce3ec;
		border-radius: 10px;
		overflow: hidden;
		box-shadow: 0 2px 5px #0f172a05;
	}
	.diagram-card > header {
		display: flex;
		justify-content: space-between;
		align-items: center;
		flex-wrap: wrap;
		gap: 8px;
		padding: 12px 16px;
		border-bottom: 1px solid #edf1f5;
		min-height: 64px;
	}
	.diagram-card h2 {
		font-size: 14px;
		font-weight: 650;
	}
	.diagram-card header span {
		font-size: 11px;
		color: #64748b;
	}
	.diagram-state {
		padding: 4px 7px;
		border-radius: 5px;
		background: #f1f5f9;
	}
	.result-actions {
		display: flex;
		gap: 5px;
		flex-wrap: wrap;
	}
	.result-actions .ui-button {
		font-size: 11px;
		min-height: 28px;
		padding: 5px 8px;
	}
	.diagram-surface {
		flex: 1;
		min-height: 0;
		min-width: 0;
	}
	.diagram-card > footer {
		display: flex;
		flex-wrap: wrap;
		gap: 4px 12px;
		justify-content: space-between;
		padding: 9px 14px;
		border-top: 1px solid #edf1f5;
		color: #64748b;
		font-size: 10px;
		line-height: 1.5;
	}
	.analysis-empty {
		display: flex;
		flex-direction: column;
		justify-content: center;
		align-items: center;
		height: 100%;
		text-align: center;
		padding: 24px;
	}
	.empty-symbol {
		font-size: 32px;
		color: #0d9488;
		margin-bottom: 15px;
	}
	.analysis-empty h3 {
		font-size: 18px;
		font-weight: 600;
	}
	.analysis-empty p {
		max-width: 400px;
		font-size: 13px;
		color: #64748b;
		line-height: 1.7;
		margin: 12px 0 20px;
	}
	.analysis-inspector {
		display: flex;
		flex-direction: column;
		min-height: 0;
		background: #fff;
		border-left: 1px solid #dce3ec;
	}
	.analysis-inspector > header {
		padding: 18px 16px;
		border-bottom: 1px solid #edf1f5;
		display: flex;
		justify-content: space-between;
		gap: 8px;
		align-items: center;
	}
	.analysis-inspector h2 {
		font-size: 15px;
		font-weight: 650;
	}
	.analysis-inspector-scroll {
		padding: 16px;
		overflow-y: auto;
		flex: 1;
		min-height: 0;
	}
	.inspector-switch {
		display: flex;
		border: 1px solid #dce3ec;
		border-radius: 5px;
	}
	.inspector-switch button {
		padding: 6px 8px;
		font-size: 11px;
	}
	.inspector-switch button[aria-pressed='true'] {
		background: #e6f6f2;
		color: #0f766e;
	}
	.corridor-controls :global(label),
	.corridor-controls :global(span.block) {
		font-size: 11px;
		margin-bottom: 5px;
		color: #475569;
	}
	.corridor-controls :global(input),
	.corridor-controls :global(select),
	.corridor-controls :global(.tabular-nums) {
		font-size: 12px;
		padding: 8px;
		border-color: #d7e0e8;
	}
	.corridor-controls :global(.grid) {
		grid-template-columns: repeat(2, minmax(0, 1fr));
		gap: 10px;
	}
	.corridor-controls > :global(.text-sm) {
		font-size: 12px;
	}
	@media (min-width: 901px) {
		.analysis-charts.comparing {
			grid-template-rows: auto minmax(0, 1fr) auto;
			row-gap: 0;
		}
		.comparing .diagram-card {
			display: grid;
			grid-template-rows: subgrid;
			grid-row: span 3;
		}
	}
	@media (max-width: 1200px) {
		.analysis-actions {
			width: 100%;
		}
		.analysis-actions > :last-child {
			margin-left: auto;
		}
		.analysis-body {
			grid-template-columns: minmax(0, 1fr) 285px;
		}
		.waves-toggle {
			margin-left: 0;
		}
	}
	@media (max-width: 480px) {
		.analysis-actions .ui-button {
			font-size: 11px;
			padding: 6px 8px;
		}
		.analysis-actions .details-toggle {
			font-size: 0;
		}
		.details-toggle::before {
			content: 'Details';
			font-size: 11px;
		}
	}
	@media (max-width: 900px) {
		.analysis-workspace {
			overflow-y: auto;
		}
		.analysis-body,
		.analysis-body.details-hidden {
			display: flex;
			flex-direction: column;
			flex: none;
		}
		.analysis-main {
			flex: none;
			padding: 10px;
		}
		.analysis-charts {
			flex: none;
		}
		.analysis-charts.comparing {
			grid-template-columns: minmax(0, 1fr);
		}
		.diagram-card {
			height: 65dvh;
			min-height: 430px;
		}
		.analysis-inspector {
			border-left: 0;
			border-top: 1px solid #dce3ec;
		}
		.analysis-inspector-scroll {
			overflow: visible;
		}
		.analysis-heading {
			padding: 10px 12px;
		}
		.analysis-route {
			flex: 1;
		}
		.analysis-mode {
			margin-left: 0;
		}
	}
</style>
