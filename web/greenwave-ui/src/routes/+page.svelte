<script>
  import TimeSpaceDiagram from '../components/TimeSpaceDiagram.svelte';
  import RoadView from '../components/RoadView.svelte';
  import WaveStatus from '../components/WaveStatus.svelte';
  import { reverseGroupNotice } from '$lib/utils/wave-status.js';
  import ConfirmModal from '../components/ConfirmModal.svelte';
  import EditSignalModal from '../components/EditSignalModal.svelte';
  import EditJunctionModal from '../components/EditJunctionModal.svelte';
  import DropdownMenu from '../components/DropdownMenu.svelte';
  import { slide } from 'svelte/transition';
  import { isLoading, error, resetToDemo, resetToEmpty } from '$lib/stores';
  import { exportToJSON, importFromJSON, validateImportedConfig, prepareInputExport, prepareOutputExport } from '$lib/utils/export-import.js';
  import { junctions, desiredSpeed, desiredIntensity, desiredFlow, optimizationDirection } from '$lib/stores/core';
  import { corridorEditor, corridorHistory, corridorPersistence, corridorGraph, corridorGroupIds, corridorReverseGroupIds } from '$lib/stores/corridor.js';
  import { wavesAreOutdated, originalGreenWaves, originalThroughWaves, originalReverseGreenWaves, originalReverseThroughWaves, showGreenWaves, lastCalculatedSpeed, storeWaveCalculationPositions, actualFlow, actualIntensity, actualReverseFlow, actualReverseIntensity } from '$lib/stores/greenwave';
  import { optimizedGroupIds, optimizedReverseGroupIds, optimizedDirection, optimizedResultsAreOutdated, optimizedWaveCalculationPositions, optimizedLastCalculatedSpeed, optimizedJunctions, optimizedOffsets, optimizedGreenWaves, optimizedThroughWaves, optimizedReverseGreenWaves, optimizedReverseThroughWaves, actualFlowOptimized, actualIntensityOptimized, actualReverseFlowOptimized, actualReverseIntensityOptimized } from '$lib/stores/optimization';
  import { extractGreenWaves } from '$lib/api/greenwave.js';
  import { optimizeOffsets } from '$lib/api/optimize.js';
  import { prepareJunctionsForAPI, applyOffsetsToJunctions, validateJunctionCycles } from '$lib/utils/junction-helpers.js';
  import { onDestroy } from 'svelte';
  import { get } from 'svelte/store';
  import { corridorGroupId, corridorProgramError, programGroupIds } from '$lib/utils/junction-program.js';
  import { invalidateAll, validateInput, validateResults } from '$lib/stores/invalidation';

  onDestroy(() => corridorEditor.finish());

  function reveal(node) {
    return slide(node, { duration: window.matchMedia('(prefers-reduced-motion: reduce)').matches ? 0 : 180 });
  }

  // Confirmation modal state
  let showResetModal = false;
  let showDemoModal = false;
  let showImportModal = false;
  let pendingImportData = null;

  // Per-operation loading flags (global $isLoading stays for disabling buttons)
  let isExtracting = false;
  let isOptimizing = false;

  // View mode: 'diagram' | 'road'
  let viewMode = 'diagram';

  // Signal modal state
  let selectedSignal = null;
  let selectedSignalContext = null;
  let isSignalModalOpen = false;

  // Junction modal state
  let selectedJunction = null;
  let selectedJunctionSignal = null;
  let isJunctionModalOpen = false;
  let isNewJunction = false;

  // Reactive variables
  $: hasGreenWaveData = $originalGreenWaves.length > 0;
  $: hasResults = $optimizedJunctions.length > 0;

  // Validation: check if all junctions have the same cycle duration
  $: cycleValidation = $junctions.length >= 2 ? validateJunctionCycles($junctions) : { isValid: true, durations: [] };
  $: programError = corridorProgramError($junctions, $corridorGroupIds, $corridorReverseGroupIds, $optimizationDirection);
  $: hasValidationError = !!programError || !cycleValidation.isValid;
  $: validationErrorMessage = programError || (hasValidationError
    ? `Different cycle durations: ${$junctions.map((j, i) => `${j.label}: ${cycleValidation.durations[i]}s`).join(', ')}`
    : '');

  $: isExtractDisabled = $isLoading || $junctions.length < 2 || hasValidationError || !Number.isFinite($desiredSpeed) || $desiredSpeed <= 0;

  $: isCleanState = $junctions.length === 0 && !hasGreenWaveData;

  // Extract green waves from API
  async function handleExtractWaves() {
    if (isExtractDisabled) return;
    const revision = get(corridorEditor.calculationRevision);
    try {
      isExtracting = true;
      isLoading.set(true);
      error.set(null);

      const junctionsForAPI = prepareJunctionsForAPI($junctions);
      const response = await extractGreenWaves(junctionsForAPI, $desiredSpeed, $optimizationDirection, $corridorGroupIds, $corridorReverseGroupIds);
      if (revision !== get(corridorEditor.calculationRevision)) throw new Error('Input changed during extraction. Extract waves again for the current corridor.');
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

  // Smart modal handlers
  function handleResetClick() {
    if (isCleanState) {
      resetToEmpty();
    } else {
      showResetModal = true;
    }
  }

  function handleDemoDataClick() {
    if (isCleanState) {
      resetToDemo();
    } else {
      showDemoModal = true;
    }
  }

  function confirmReset() {
    resetToEmpty();
  }

  function confirmDemoData() {
    resetToDemo();
  }

  function confirmImport() {
    if (pendingImportData) {
      corridorEditor.replaceInput(pendingImportData);
      invalidateAll('configuration imported');
      pendingImportData = null;
    }
  }

  function updateJunction(event) {
    const { id, newDistance } = event.detail;
    junctions.update(list => list.map(junction => junction.id === id
      ? { ...junction, point: { ...junction.point, y: newDistance } }
      : junction));
  }

  // Handle optimization
  async function handleOptimize() {
    if (isExtractDisabled) return;
    const revision = get(corridorEditor.calculationRevision);
    try {
      isOptimizing = true;
      isLoading.set(true);
      error.set(null);

      const junctionsForAPI = prepareJunctionsForAPI($junctions);
      const optimizeResponse = await optimizeOffsets(junctionsForAPI, $desiredSpeed, 'genetic', {}, $optimizationDirection, $corridorGroupIds, $corridorReverseGroupIds);
      if (revision !== get(corridorEditor.calculationRevision)) throw new Error('Input changed during optimization. Optimize the current corridor again.');
      const nextOffsets = optimizeResponse.best_offsets || [];
      const nextJunctions = applyOffsetsToJunctions($junctions, nextOffsets);
      const optimizedJunctionsForAPI = prepareJunctionsForAPI(nextJunctions);
      const response = await extractGreenWaves(optimizedJunctionsForAPI, $desiredSpeed, $optimizationDirection, $corridorGroupIds, $corridorReverseGroupIds);
      if (revision !== get(corridorEditor.calculationRevision)) throw new Error('Input changed during optimization. Optimize the current corridor again.');

      optimizedOffsets.set(nextOffsets);
      optimizedJunctions.set(nextJunctions);
      optimizedGroupIds.set(structuredClone($corridorGroupIds));
      optimizedReverseGroupIds.set(structuredClone($corridorReverseGroupIds));
      optimizedDirection.set($optimizationDirection);
      optimizedGreenWaves.set(response.green_waves || []);
      optimizedThroughWaves.set(response.through_green_waves || []);
      optimizedReverseGreenWaves.set(response.reverse_green_waves || []);
      optimizedReverseThroughWaves.set(response.reverse_through_green_waves || []);

      optimizedWaveCalculationPositions.set($junctions.map(j => ({ id: j.id, y: j.point.y })));
      optimizedLastCalculatedSpeed.set($desiredSpeed);
      validateResults();
    } catch (optimizeError) {
      error.set(optimizeError.message || 'Failed to optimize');
      console.error('Optimize Error:', optimizeError);
    } finally {
      isOptimizing = false;
      isLoading.set(false);
    }
  }

  function clearResults() {
    optimizedJunctions.set([]);
    optimizedGroupIds.set({});
    optimizedReverseGroupIds.set({});
    optimizedDirection.set('forward');
    optimizedOffsets.set([]);
    optimizedGreenWaves.set([]);
    optimizedThroughWaves.set([]);
    optimizedReverseGreenWaves.set([]);
    optimizedReverseThroughWaves.set([]);
    validateResults();
  }

  function saveSignal(e) {
    const updatedSignal = e.detail.signal;

    if (!selectedSignalContext) {
      console.error("Invalid selectedSignalContext:", selectedSignalContext);
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
                      if (sg.id === selectedSignalContext.groupId && index === selectedSignalContext.signalIndex) {
                        return { ...signal, ...updatedSignal };
                      }
                      return signal;
                    }),
                  })),
                };
              }
              return phase;
            }),
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
    const group = phase.signal_groups.find(group => group.signals.includes(signal));
    if (programGroupIds(junction).length > 1) {
      openJunctionModal({ detail: { junction } });
      selectedJunctionSignal = { groupId: group.id, phaseId: phase.id, signalIndex: group.signals.indexOf(signal) };
      return;
    }
    selectedSignalContext = { junctionId: junction.id, phaseId: phase.id, groupId: group.id, signalIndex: group.signals.indexOf(signal) };
    selectedSignal = structuredClone(signal);
    isSignalModalOpen = true;
  }

  function closeSignalModal() {
    selectedSignal = null;
    isSignalModalOpen = false;
  }

  function openJunctionModal(event) {
    const { junction } = event.detail;
    const originalJunction = $junctions.find(j => j.id === junction.id);
    selectedJunction = originalJunction || junction;
    selectedJunctionSignal = null;
    isNewJunction = false;
    isJunctionModalOpen = true;
  }

  function openNewJunctionModal() {
    const maxId = $junctions.length > 0 ? Math.max(...$junctions.map(j => j.id)) : -1;
    const maxY = $junctions.length > 0 ? Math.max(...$junctions.map(j => j.point.y)) : -100;

    selectedJunction = {
      id: maxId + 1,
      label: `Junction ${maxId + 2}`,
      cycle: [
        {
          id: (maxId + 1) * 10,
          signal_groups: [{ id: 0, signals: [
            { duration: 30, color: 'GREEN' },
            { duration: 20, color: 'RED' }
          ]}]
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
    corridorEditor.saveJunction(junction, isNew, event.detail.groupId, event.detail.reverseGroupId);
    invalidateAll('junction configuration changed');
    closeJunctionModal();
  }

  function deleteJunction(event) {
    const { junction } = event.detail;
    corridorEditor.removeJunction(junction.id);
    invalidateAll('junction deleted');
    closeJunctionModal();
  }

  function closeJunctionModal() {
    selectedJunction = null;
    selectedJunctionSignal = null;
    isJunctionModalOpen = false;
    isNewJunction = false;
  }

  // File menu items
  $: fileMenuItems = [
    { id: 'import-input', label: 'Import' },
    { id: 'export-input', label: 'Export Input' },
    { id: 'export-output', label: 'Export Output', disabled: $optimizedJunctions.length === 0 },
    { separator: true },
    { id: 'demo-data', label: 'Load Demo Data' },
    { id: 'reset', label: 'Reset All', danger: true }
  ];

  async function handleFileMenuSelect(event) {
    const { id } = event.detail;

    switch (id) {
      case 'export-input': {
        const inputData = prepareInputExport($junctions, $desiredSpeed, $desiredIntensity, $optimizationDirection, $corridorGroupIds, $corridorReverseGroupIds);
        exportToJSON(inputData, 'greenwave-input.json');
        break;

      }
      case 'import-input':
        try {
          const imported = await importFromJSON();
          const validation = validateImportedConfig(imported);

          if (!validation.isValid) {
            error.set(`Invalid file: ${validation.errors.join(', ')}`);
            return;
          }

          if (isCleanState) {
            corridorEditor.replaceInput(imported);
            invalidateAll('configuration imported');
          } else {
            pendingImportData = imported;
            showImportModal = true;
          }
        } catch (err) {
          if (err.message !== 'No file selected') {
            error.set(err.message);
          }
        }
        break;

      case 'export-output': {
        const outputData = prepareOutputExport($optimizedJunctions, $desiredSpeed, $desiredIntensity, $optimizedDirection, $optimizedGroupIds, $optimizedReverseGroupIds);
        exportToJSON(outputData, 'greenwave-output.json');
        break;

      }
      case 'demo-data':
        handleDemoDataClick();
        break;

      case 'reset':
        handleResetClick();
        break;
    }
  }
</script>

<!-- Confirmation Modals -->
<ConfirmModal
  bind:show={showResetModal}
  title="Reset All Data"
  message="This will clear the shared input corridor, reset the desired speed, and remove calculated results. Undo can restore the input configuration."
  confirmText="Reset"
  cancelText="Cancel"
  onConfirm={confirmReset}
  danger={true}
/>

<ConfirmModal
  bind:show={showDemoModal}
  title="Load Demo Data"
  message="This will replace your current configuration with sample data and clear all calculated results."
  confirmText="Load Demo Data"
  cancelText="Cancel"
  onConfirm={confirmDemoData}
  danger={false}
/>

<ConfirmModal
  bind:show={showImportModal}
  title="Import Configuration"
  message="This will replace your current configuration with imported data and clear all calculated results."
  confirmText="Import"
  cancelText="Cancel"
  onConfirm={confirmImport}
  danger={false}
/>

{#if isSignalModalOpen}
  <EditSignalModal
    signal={selectedSignal}
    on:save={saveSignal}
    on:close={closeSignalModal}
  />
{/if}

{#if isJunctionModalOpen}
  <EditJunctionModal
    junction={selectedJunction}
    isNew={isNewJunction}
    graph={$corridorGraph}
    corridorGroup={corridorGroupId(selectedJunction, $corridorGroupIds)}
    reverseCorridorGroup={$corridorReverseGroupIds[selectedJunction.id] ?? null}
    direction={$optimizationDirection}
    initialSignal={selectedJunctionSignal}
    on:save={saveJunction}
    on:delete={deleteJunction}
    on:close={closeJunctionModal}
  />
{/if}

<div class="corridor-page min-h-screen bg-gray-50 flex flex-col">
  <div class="corridor-workspace container mx-auto p-4 flex-1 flex flex-col">

    <!-- Header -->
    <div class="corridor-page-header mb-4">
      <div class="flex flex-wrap items-center justify-between gap-2">
        <div class="w-32">
          <a href="/network"
            class="inline-flex items-center gap-1.5 text-sm text-gray-500 hover:text-gray-800 font-medium transition-colors"
          >
            <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                d="M4 6h16M4 12h16M4 18h16"/>
            </svg>
            Network
          </a>
        </div>
        <h1 class="order-first w-full text-xl sm:text-2xl lg:order-none lg:w-auto lg:text-3xl font-bold text-center">Green Wave Traffic Light Optimizer</h1>
        <div class="w-32 flex justify-end">
          <div class="inline-flex bg-gray-100 rounded-lg p-0.5 gap-0.5 text-sm">
            <button
              on:click={() => viewMode = 'diagram'}
              class="px-3 py-1.5 rounded-md font-medium transition-all"
              class:bg-white={viewMode === 'diagram'}
              class:text-gray-900={viewMode === 'diagram'}
              class:shadow-sm={viewMode === 'diagram'}
              class:text-gray-500={viewMode !== 'diagram'}
            >Diagram</button>
            <button
              on:click={() => viewMode = 'road'}
              class="px-3 py-1.5 rounded-md font-medium transition-all"
              class:bg-white={viewMode === 'road'}
              class:text-gray-900={viewMode === 'road'}
              class:shadow-sm={viewMode === 'road'}
              class:text-gray-500={viewMode !== 'road'}
            >Road</button>
          </div>
        </div>
      </div>

      <!-- Error banner -->
      {#if $error}
        <div
          transition:slide={{ duration: 200 }}
          class="bg-red-100 border border-red-400 text-red-700 px-4 py-3 rounded-md mt-3 flex justify-between items-start gap-3"
        >
          <span><strong>Error:</strong> {$error}</span>
          <button
            on:click={() => error.set(null)}
            class="text-red-500 hover:text-red-700 leading-none shrink-0 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-red-400 rounded"
            aria-label="Dismiss error"
          >✕</button>
        </div>
      {/if}
    </div>

    <!-- Main content grid  Input LEFT, Results RIGHT -->
    <div class="corridor-grid grid grid-cols-1 lg:grid-cols-2 items-start gap-4">

      <!-- LEFT: Input Configuration -->
      <div class="corridor-panel bg-white rounded-lg shadow-md p-4 sm:p-5 flex flex-col min-w-0">

        <!-- Panel header: title + wrapping toolbar -->
        <div class="corridor-panel-header mb-4">
          <div class="flex justify-between items-center mb-3">
            <h2 class="text-xl font-semibold">Input configuration</h2>
          </div>
          <p class="mb-3 text-xs" class:text-red-700={$corridorPersistence.state === 'error'} class:text-gray-500={$corridorPersistence.state !== 'error'} role="status">{$corridorPersistence.message}</p>
          <!-- Compact controls wrap when the panel is narrow. -->
          <div class="corridor-toolbar flex flex-wrap gap-1.5 items-center">
            <DropdownMenu
              label="File"
              items={fileMenuItems}
              on:select={handleFileMenuSelect}
            />

            <button on:click={() => corridorEditor.undo()} disabled={!$corridorHistory.undo} class="rounded-md border px-3 py-2 text-sm disabled:opacity-40">Undo</button>
            <button on:click={() => corridorEditor.redo()} disabled={!$corridorHistory.redo} class="rounded-md border px-3 py-2 text-sm disabled:opacity-40">Redo</button>

            <button
              on:click={openNewJunctionModal}
              class="px-3 py-2 bg-blue-500 text-white rounded-md hover:bg-blue-600 text-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-400 focus-visible:ring-offset-1"
              title="Add a new junction"
            >
              + Add Junction
            </button>

            <button
              on:click={handleExtractWaves}
              disabled={isExtractDisabled}
              class="px-3 py-2 bg-green-500 text-white rounded-md hover:bg-green-600 text-sm disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-1.5 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-green-400 focus-visible:ring-offset-1"
            >
              {#if isExtracting}
                <div class="w-3.5 h-3.5 border-2 border-white border-t-transparent rounded-full animate-spin"></div>
                Extracting...
              {:else}
                Extract waves
              {/if}
            </button>

            <button
              on:click={handleOptimize}
              disabled={isExtractDisabled}
              class="px-3 py-2 bg-purple-500 text-white rounded-md hover:bg-purple-600 text-sm disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-1.5 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-purple-400 focus-visible:ring-offset-1"
              title="Recalculate waves and optimize offsets"
            >
              {#if isOptimizing}
                <div class="w-3.5 h-3.5 border-2 border-white border-t-transparent rounded-full animate-spin"></div>
                Optimizing...
              {:else}
                Optimize
              {/if}
            </button>

            <!-- Keep the toggle in the toolbar flow. -->
            <label class="flex items-center gap-2 cursor-pointer select-none py-1 px-1">
              <input
                type="checkbox"
                bind:checked={$showGreenWaves}
                disabled={!hasGreenWaveData}
                class="w-4 h-4 cursor-pointer disabled:cursor-not-allowed"
              />
              <span class="text-sm" class:text-gray-400={!hasGreenWaveData}>Show waves</span>
            </label>
          </div>
        </div>

        <!-- Chart height stays independent of the controls below it. -->
        <div class="corridor-chart border border-gray-300 rounded-md mb-1 overflow-hidden">
          {#if $junctions.length === 0}
            <div class="flex items-center justify-center h-full text-gray-500">
              <div class="text-center p-6">
                <svg class="w-16 h-16 mx-auto mb-4 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 6v6m0 0v6m0-6h6m-6 0H6"></path>
                </svg>
                <h3 class="text-lg font-medium mb-2">No junctions configured</h3>
                <p class="text-sm mb-4">Add junctions to start visualizing traffic light coordination</p>
                <div class="flex gap-2 justify-center flex-wrap">
                  <button on:click={openNewJunctionModal} class="px-4 py-2 bg-blue-500 text-white rounded-md hover:bg-blue-600 text-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-400">
                    + Add First Junction
                  </button>
                  <button on:click={handleDemoDataClick} class="px-4 py-2 bg-gray-500 text-white rounded-md hover:bg-gray-600 text-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-gray-400">
                    Load Demo Data
                  </button>
                </div>
              </div>
            </div>
          {:else if viewMode === 'diagram'}
            <TimeSpaceDiagram
              junctions={$junctions}
              groupIds={$corridorGroupIds} reverseGroupIds={$corridorReverseGroupIds} direction={$optimizationDirection}
              wavesAreOutdated={$wavesAreOutdated}
              interactive={true}
              greenWaves={$originalGreenWaves}
              throughWaves={$originalThroughWaves}
              reverseGreenWaves={$originalReverseGreenWaves}
              reverseThroughWaves={$originalReverseThroughWaves}
              showWaves={$showGreenWaves}
              on:updateJunction={updateJunction}
              on:dragStart={() => corridorEditor.begin()}
              on:dragEnd={() => corridorEditor.finish()}
              on:editSignal={openSignalModal}
              on:editJunction={openJunctionModal}
            />
          {:else}
            <RoadView junctions={$junctions} groupIds={$corridorGroupIds} reverseGroupIds={$corridorReverseGroupIds} direction={$optimizationDirection} />
          {/if}
        </div>

        <div class="corridor-chart-hint mb-3 text-xs text-gray-500 leading-relaxed">
          {#if $junctions.length > 0}
            {#if $wavesAreOutdated.isOutdated}
              <span class="text-orange-600">⚠️ {$wavesAreOutdated.reason}</span>
            {:else}
              Drag junctions to reposition · Click junction label or circle to edit · Click signal line to change
            {/if}
          {/if}
        </div>

        <!-- Keyboard focus lets users scroll this region with Page Up and Page Down. -->
        <!-- svelte-ignore a11y_no_noninteractive_tabindex -->
        <section aria-label="Input settings and indicators" tabindex="0" class="corridor-controls border-t pt-3 space-y-3">
          <div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
            <div>
              <label for="input-direction" class="block text-sm font-medium mb-2">Optimization direction</label>
              <select id="input-direction" bind:value={$optimizationDirection} class="w-full px-3 py-2 border rounded-md">
                <option value="forward">Forward only</option>
                <option value="bidirectional">Bidirectional</option>
              </select>
            </div>
            <div>
              <label for="input-desired-speed" class="block text-sm font-medium mb-2">Desired speed (km/h)</label>
              <input id="input-desired-speed" type="number" bind:value={$desiredSpeed} class="w-full px-3 py-2 border rounded-md" min="10" max="100" />
            </div>
          </div>

          <!-- Junction status -> own line so it never shifts the grid above -->
          <div class="text-sm leading-snug">
            <span class="font-medium text-gray-700">{$junctions.length} junctions</span>
            {#if $junctions.length === 0}
              <span class="text-blue-600"> -> use File menu or add manually</span>
            {:else if $junctions.length === 1}
              <span class="text-orange-600"> -> add at least 1 more to extract waves</span>
            {:else if hasValidationError}
              <span class="block text-red-600 mt-0.5">⚠ {validationErrorMessage}</span>
            {:else if $lastCalculatedSpeed !== null && !$wavesAreOutdated.isOutdated}
              <span class="text-gray-600"> · calculation complete</span>
            {:else}
              <span class="text-orange-600"> -> press "Extract waves" to calculate</span>
            {/if}
          </div>

          <div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
            <div>
              <label for="input-desired-intensity" class="block text-sm font-medium mb-2">Desired intensity (veh/h)</label>
              <input id="input-desired-intensity" type="number" bind:value={$desiredIntensity} class="w-full px-3 py-2 border rounded-md" min="0" />
            </div>
            <div>
              <label for="input-desired-flow" class="block text-sm font-medium mb-2">Desired flow (veh/s)</label>
              <input id="input-desired-flow" type="number" value={$desiredFlow} class="w-full px-3 py-2 border rounded-md" min="0" step="0.5" on:input={(e) => desiredIntensity.set(e.target.value * 3600)} />
            </div>
          </div>

          <div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
            <div>
              <span class="block text-sm font-medium mb-2">Actual intensity{#if $optimizationDirection === 'bidirectional'} <span class="text-green-600">(fwd)</span>{/if}</span>
              <div class="w-full px-3 py-2 border rounded-md bg-gray-100 text-gray-700 tabular-nums">
                {($actualIntensity || 0).toFixed(2)} veh/h
                {#if $wavesAreOutdated.isOutdated}<span class="text-orange-500 text-xs">(outdated)</span>{/if}
              </div>
            </div>
            <div>
              <span class="block text-sm font-medium mb-2">Actual flow{#if $optimizationDirection === 'bidirectional'} <span class="text-green-600">(fwd)</span>{/if}</span>
              <div class="w-full px-3 py-2 border rounded-md bg-gray-100 text-gray-700 tabular-nums">
                {($actualFlow || 0).toFixed(6)} veh/s
                {#if $wavesAreOutdated.isOutdated}<span class="text-orange-500 text-xs">(outdated)</span>{/if}
              </div>
            </div>
          </div>

          <WaveStatus junctionCount={$junctions.length} waves={$originalGreenWaves} throughWaves={$originalThroughWaves} calculated={$lastCalculatedSpeed !== null} outdated={$wavesAreOutdated.isOutdated} />

          {#if $optimizationDirection === 'bidirectional'}
            <div transition:reveal class="space-y-2" data-direction-panel="input-reverse">
              <WaveStatus direction="Reverse" junctionCount={$junctions.length} waves={$originalReverseGreenWaves} throughWaves={$originalReverseThroughWaves} calculated={$lastCalculatedSpeed !== null} outdated={$wavesAreOutdated.isOutdated} notice={reverseGroupNotice($junctions, $corridorGroupIds, $corridorReverseGroupIds)} />
              <div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
                <div>
                  <span class="block text-sm font-medium mb-2">Actual intensity <span class="text-blue-600">(rev)</span></span>
                  <div class="w-full px-3 py-2 border rounded-md bg-gray-100 text-gray-700 tabular-nums">
                    {($actualReverseIntensity || 0).toFixed(2)} veh/h
                    {#if $wavesAreOutdated.isOutdated}<span class="text-orange-500 text-xs">(outdated)</span>{/if}
                  </div>
                </div>
                <div>
                  <span class="block text-sm font-medium mb-2">Actual flow <span class="text-blue-600">(rev)</span></span>
                  <div class="w-full px-3 py-2 border rounded-md bg-gray-100 text-gray-700 tabular-nums">
                    {($actualReverseFlow || 0).toFixed(6)} veh/s
                    {#if $wavesAreOutdated.isOutdated}<span class="text-orange-500 text-xs">(outdated)</span>{/if}
                  </div>
                </div>
              </div>
            </div>
          {/if}
        </section>
      </div>

      <!-- RIGHT: Optimized Results -->
      <div class="corridor-panel bg-white rounded-lg shadow-md p-4 sm:p-5 flex flex-col min-w-0">
        <div class="corridor-panel-header flex justify-between items-start mb-4">
          <h2 class="text-xl font-semibold">Optimized results</h2>
          <button
            on:click={clearResults}
            class="px-3 py-2 bg-red-500 text-white rounded-md hover:bg-red-600 text-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-red-400 focus-visible:ring-offset-1"
          >
            Clear results
          </button>
        </div>

        <!-- Chart height stays independent of the controls below it. -->
        <div class="corridor-chart border border-gray-300 rounded-md mb-1 overflow-hidden">
          {#if $optimizedJunctions.length > 0}
            {#if viewMode === 'diagram'}
              <TimeSpaceDiagram
                junctions={$optimizedJunctions}
                groupIds={$optimizedGroupIds} reverseGroupIds={$optimizedReverseGroupIds} direction={$optimizedDirection}
                greenWaves={$optimizedGreenWaves}
                throughWaves={$optimizedThroughWaves}
                reverseGreenWaves={$optimizedReverseGreenWaves}
                reverseThroughWaves={$optimizedReverseThroughWaves}
                showWaves={true}
                showOffsets={true}
                interactive={false}
              />
            {:else}
              <RoadView junctions={$optimizedJunctions} groupIds={$optimizedGroupIds} reverseGroupIds={$optimizedReverseGroupIds} direction={$optimizedDirection} showOffsets={true} />
            {/if}
          {:else}
            <div class="flex items-center border-2 border-dashed border-gray-300 rounded-md justify-center h-full text-gray-500 p-6">
              <p class="text-center text-sm">No optimized results yet. Configure input and press Optimize.</p>
            </div>
          {/if}
        </div>

        <div class="corridor-chart-hint mb-3 text-xs text-gray-500 leading-relaxed">
          {#if $optimizedJunctions.length > 0 && !$optimizedResultsAreOutdated.isOutdated}
            Press Optimize to recalculate with current settings
          {/if}
        </div>

        <!-- Keyboard focus lets users scroll this region with Page Up and Page Down. -->
        <!-- svelte-ignore a11y_no_noninteractive_tabindex -->
        <section aria-label="Result settings and indicators" tabindex="0" class="corridor-controls border-t pt-3 space-y-3">
          <!-- Optimization status -> own line, wraps freely -->
          <div class="text-sm leading-snug">
            {#if $optimizedJunctions.length > 0}
              <span class="font-medium text-gray-700">{$optimizedJunctions.length} junctions optimized</span>
              {#if $optimizedResultsAreOutdated.isOutdated}
                <span class="block text-orange-600 mt-0.5">⚠ {$optimizedResultsAreOutdated.reason}</span>
              {:else}
                <span class="text-green-600"> -> offsets applied</span>
              {/if}
            {:else}
              <span class="text-gray-400">No results yet -> press Optimize</span>
            {/if}
          </div>

          <div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
            <div>
              <label for="opt-desired-intensity" class="block text-sm font-medium mb-2">Desired intensity (veh/h)</label>
              <input id="opt-desired-intensity" type="number" bind:value={$desiredIntensity} class="w-full px-3 py-2 border rounded-md" min="0" />
            </div>
            <div>
              <label for="opt-desired-flow" class="block text-sm font-medium mb-2">Desired flow (veh/s)</label>
              <input id="opt-desired-flow" type="number" value={$desiredFlow} class="w-full px-3 py-2 border rounded-md" min="0" step="0.5" on:input={(e) => desiredIntensity.set(e.target.value * 3600)} />
            </div>
          </div>

          <div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
            <div>
              <span class="block text-sm font-medium mb-2">Actual intensity{#if $optimizedDirection === 'bidirectional'} <span class="text-green-600">(fwd)</span>{/if}</span>
              <div class="w-full px-3 py-2 border rounded-md bg-gray-100 text-gray-700 tabular-nums">
                {($actualIntensityOptimized || 0).toFixed(2)} veh/h
                {#if hasResults && $optimizedResultsAreOutdated.isOutdated}<span class="text-orange-500 text-xs">(outdated)</span>{/if}
              </div>
            </div>
            <div>
              <span class="block text-sm font-medium mb-2">Actual flow{#if $optimizedDirection === 'bidirectional'} <span class="text-green-600">(fwd)</span>{/if}</span>
              <div class="w-full px-3 py-2 border rounded-md bg-gray-100 text-gray-700 tabular-nums">
                {$actualFlowOptimized.toFixed(6)} veh/s
                {#if hasResults && $optimizedResultsAreOutdated.isOutdated}<span class="text-orange-500 text-xs">(outdated)</span>{/if}
              </div>
            </div>
          </div>

          {#if hasResults}
            <WaveStatus junctionCount={$optimizedJunctions.length} waves={$optimizedGreenWaves} throughWaves={$optimizedThroughWaves} calculated={hasResults} outdated={$optimizedResultsAreOutdated.isOutdated} />
          {/if}

          {#if $optimizedDirection === 'bidirectional'}
            <div transition:reveal class="space-y-2" data-direction-panel="result-reverse">
              <WaveStatus direction="Reverse" junctionCount={$optimizedJunctions.length} waves={$optimizedReverseGreenWaves} throughWaves={$optimizedReverseThroughWaves} calculated={hasResults} outdated={$optimizedResultsAreOutdated.isOutdated} notice={reverseGroupNotice($optimizedJunctions, $optimizedGroupIds, $optimizedReverseGroupIds)} />
              <div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
                <div>
                  <span class="block text-sm font-medium mb-2">Actual intensity <span class="text-blue-600">(rev)</span></span>
                  <div class="w-full px-3 py-2 border rounded-md bg-gray-100 text-gray-700 tabular-nums">
                    {($actualReverseIntensityOptimized || 0).toFixed(2)} veh/h
                    {#if hasResults && $optimizedResultsAreOutdated.isOutdated}<span class="text-orange-500 text-xs">(outdated)</span>{/if}
                  </div>
                </div>
                <div>
                  <span class="block text-sm font-medium mb-2">Actual flow <span class="text-blue-600">(rev)</span></span>
                  <div class="w-full px-3 py-2 border rounded-md bg-gray-100 text-gray-700 tabular-nums">
                    {$actualReverseFlowOptimized.toFixed(6)} veh/s
                    {#if hasResults && $optimizedResultsAreOutdated.isOutdated}<span class="text-orange-500 text-xs">(outdated)</span>{/if}
                  </div>
                </div>
              </div>
            </div>
          {/if}
        </section>
      </div>

    </div>
  </div>
</div>

<style>
  .corridor-chart {
    height: clamp(320px, 42vh, 480px);
    flex: none;
  }

  .corridor-controls:focus-visible {
    outline: 2px solid #60a5fa;
    outline-offset: 2px;
  }

  .corridor-toolbar :global([role='menu']) {
    left: 0;
    right: auto;
  }

  .corridor-toolbar :global(button) {
    min-height: 32px;
    padding: 0.375rem 0.625rem;
    font-size: 0.75rem;
  }

  .corridor-controls :global(label),
  .corridor-controls :global(span.block) {
    margin-bottom: 0.25rem;
    font-size: 0.75rem;
  }

  .corridor-controls :global(input),
  .corridor-controls :global(select),
  .corridor-controls :global(.tabular-nums) {
    padding: 0.375rem 0.625rem;
    font-size: 0.875rem;
  }

  @media (min-width: 1024px) and (min-height: 640px) {
    .corridor-page {
      height: 100dvh;
      min-height: 0;
    }

    .corridor-workspace {
      min-height: 0;
    }

    .corridor-page-header {
      flex: none;
    }

    .corridor-grid {
      flex: 1;
      min-height: 0;
      align-items: stretch;
      grid-template-rows: auto clamp(220px, 38dvh, 480px) auto minmax(0, 1fr);
      row-gap: 0.5rem;
    }

    .corridor-panel {
      display: grid;
      grid-template-rows: subgrid;
      grid-row: span 4;
      min-height: 0;
      padding: 1rem;
    }

    .corridor-panel-header,
    .corridor-chart,
    .corridor-chart-hint {
      margin-bottom: 0;
    }

    .corridor-chart {
      height: auto;
      min-height: 0;
    }

    .corridor-controls {
      min-height: 0;
      overflow-y: auto;
      overscroll-behavior-y: contain;
      scrollbar-gutter: stable;
      scrollbar-width: thin;
      padding-right: 0.5rem;
      padding-bottom: 0.25rem;
    }
  }

  @media (pointer: coarse) {
    .corridor-toolbar :global(button) {
      min-height: 40px;
    }
  }
</style>
