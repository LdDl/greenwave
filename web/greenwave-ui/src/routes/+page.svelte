<script>
  import TimeSpaceDiagram from '../components/TimeSpaceDiagram.svelte';
  import ConfirmModal from '../components/ConfirmModal.svelte';
  import EditSignalModal from '../components/EditSignalModal.svelte';
  import EditJunctionModal from '../components/EditJunctionModal.svelte';
  import DropdownMenu from '../components/DropdownMenu.svelte';
  import { slide } from 'svelte/transition';
  import { isLoading, error, resetToDemo, resetToEmpty } from '$lib/stores';
  import { exportToJSON, importFromJSON, validateImportedConfig, prepareInputExport, prepareOutputExport } from '$lib/utils/export-import.js';
  import { junctions, desiredSpeed, desiredIntensity, desiredFlow, optimizationDirection } from '$lib/stores/core';
  import { wavesAreOutdated, originalGreenWaves, originalThroughWaves, originalReverseGreenWaves, originalReverseThroughWaves, showGreenWaves, storeWaveCalculationPositions, actualFlow, actualIntensity, actualReverseFlow, actualReverseIntensity } from '$lib/stores/greenwave';
  import { optimizedResultsAreOutdated, optimizedWaveCalculationPositions, optimizedLastCalculatedSpeed, optimizedJunctions, optimizedOffsets, optimizedGreenWaves, optimizedThroughWaves, optimizedReverseGreenWaves, optimizedReverseThroughWaves, actualFlowOptimized, actualIntensityOptimized, actualReverseFlowOptimized, actualReverseIntensityOptimized } from '$lib/stores/optimization';
  import { invalidateSignals, resetResultsInvalidation } from '$lib/stores/signals';
  import { extractGreenWaves } from '$lib/api/greenwave.js';
  import { optimizeOffsets } from '$lib/api/optimize.js';
  import { prepareJunctionsForAPI, applyOffsetsToJunctions, validateJunctionCycles, calculateTotalDuration } from '$lib/utils/junction-helpers.js';
  import { onMount } from 'svelte';
  import { invalidateAll, validateInput, validateResults } from '$lib/stores/invalidation';

  onMount(() => {
    isDesiredSpeedInitialized = true;
    previousDesiredSpeed = $desiredSpeed;
    isDirectionInitialized = true;
    previousDirection = $optimizationDirection;
  });

  // Confirmation modal state
  let showResetModal = false;
  let showDemoModal = false;
  let showImportModal = false;
  let pendingImportData = null;

  // Per-operation loading flags (global $isLoading stays for disabling buttons)
  let isExtracting = false;
  let isOptimizing = false;

  // Signal modal state
  let selectedSignal = null;
  let selectedSignalContext = null;
  let isSignalModalOpen = false;

  // Junction modal state
  let selectedJunction = null;
  let isJunctionModalOpen = false;
  let isNewJunction = false;

  // Reactive variables
  $: hasGreenWaveData = $originalGreenWaves.length > 0;
  $: hasResults = $optimizedGreenWaves.length > 0 || $optimizedThroughWaves.length > 0;

  // Validation: check if all junctions have the same cycle duration
  $: cycleValidation = $junctions.length >= 2 ? validateJunctionCycles($junctions) : { isValid: true, durations: [] };
  $: hasValidationError = !cycleValidation.isValid;
  $: validationErrorMessage = hasValidationError
    ? `Different cycle durations: ${$junctions.map((j, i) => `${j.label}: ${cycleValidation.durations[i]}s`).join(', ')}`
    : '';

  $: isExtractDisabled = $isLoading || $junctions.length < 2 || hasValidationError;

  $: isCleanState = $junctions.length === 0 && !hasGreenWaveData;

  // Extract green waves from API
  async function handleExtractWaves() {
    if (isExtractDisabled) return;
    try {
      isExtracting = true;
      isLoading.set(true);
      error.set(null);

      const junctionsForAPI = prepareJunctionsForAPI($junctions);
      const response = await extractGreenWaves(junctionsForAPI, $desiredSpeed, $optimizationDirection);
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
      junctions.set(pendingImportData.junctions);
      if (pendingImportData.desiredSpeed) desiredSpeed.set(pendingImportData.desiredSpeed);
      if (pendingImportData.desiredIntensity) desiredIntensity.set(pendingImportData.desiredIntensity);
      if (pendingImportData.direction) optimizationDirection.set(pendingImportData.direction);
      invalidateAll('configuration imported');
      pendingImportData = null;
    }
  }

  // Handle desired speed changes
  let desiredSpeedTimeout;
  let isDesiredSpeedInitialized = false;
  let previousDesiredSpeed = null;
  $: if (isDesiredSpeedInitialized && $desiredSpeed !== previousDesiredSpeed) {
    previousDesiredSpeed = $desiredSpeed;
    clearTimeout(desiredSpeedTimeout);
    desiredSpeedTimeout = setTimeout(() => {
      invalidateAll('desired speed changed');
    }, 100);
  }

  // Handle optimization direction changes
  let directionTimeout;
  let isDirectionInitialized = false;
  let previousDirection = null;
  $: if (isDirectionInitialized && $optimizationDirection !== previousDirection) {
    previousDirection = $optimizationDirection;
    clearTimeout(directionTimeout);
    directionTimeout = setTimeout(() => {
      invalidateAll('optimization direction changed');
    }, 100);
  }

  let debounceTimeout;
  function updateJunction(event) {
    const { id, newDistance } = event.detail;
    clearTimeout(debounceTimeout);
    debounceTimeout = setTimeout(() => {
      junctions.update(junctionList => {
        return junctionList.map(junction => {
          if (junction.id === id) {
            return { ...junction, point: { ...junction.point, y: newDistance } };
          }
          return junction;
        });
      });
    }, 100);
    invalidateAll('junction positions changed');
  }

  // Handle optimization
  async function handleOptimize() {
    if (isExtractDisabled) return;
    try {
      isOptimizing = true;
      isLoading.set(true);
      error.set(null);

      const junctionsForAPI = prepareJunctionsForAPI($junctions);
      const optimizeResponse = await optimizeOffsets(junctionsForAPI, $desiredSpeed, 'genetic', {}, $optimizationDirection);

      optimizedOffsets.set(optimizeResponse.best_offsets || []);
      optimizedJunctions.set(applyOffsetsToJunctions($junctions, optimizeResponse.best_offsets));

      const optimizedJunctionsForAPI = prepareJunctionsForAPI($optimizedJunctions);
      const response = await extractGreenWaves(optimizedJunctionsForAPI, $desiredSpeed, $optimizationDirection);
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
    optimizedOffsets.set([]);
    optimizedGreenWaves.set([]);
    optimizedThroughWaves.set([]);
    optimizedReverseGreenWaves.set([]);
    optimizedReverseThroughWaves.set([]);
    optimizedResultsAreOutdated.set(false);
  }

  function saveSignal(e) {
    const updatedSignal = e.detail.signal;

    if (!selectedSignalContext || !selectedSignalContext.junction || !selectedSignalContext.phase) {
      console.error("Invalid selectedSignalContext:", selectedSignalContext);
      return;
    }

    junctions.update((junctionList) => {
      const updatedJunctions = junctionList.map((junction) => {
        if (junction.id === selectedSignalContext.junction.id) {
          return {
            ...junction,
            cycle: junction.cycle.map((phase) => {
              if (phase === selectedSignalContext.phase) {
                return {
                  ...phase,
                  signal_groups: phase.signal_groups.map((sg) => ({
                    ...sg,
                    signals: sg.signals.map((signal) => {
                      if (signal === selectedSignal) {
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
    selectedSignalContext = { junction, phase };
    selectedSignal = signal;
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
      point: { x: 0, y: maxY + 150 }
    };
    isNewJunction = true;
    isJunctionModalOpen = true;
  }

  function saveJunction(event) {
    const { junction, isNew } = event.detail;
    if (isNew) {
      junctions.update(junctionList => [...junctionList, junction]);
    } else {
      junctions.update(junctionList =>
        junctionList.map(j => j.id === junction.id ? junction : j)
      );
    }
    invalidateAll('junction configuration changed');
    closeJunctionModal();
  }

  function deleteJunction(event) {
    const { junction } = event.detail;
    junctions.update(junctionList => junctionList.filter(j => j.id !== junction.id));
    invalidateAll('junction deleted');
    closeJunctionModal();
  }

  function closeJunctionModal() {
    selectedJunction = null;
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
      case 'export-input':
        const inputData = prepareInputExport($junctions, $desiredSpeed, $desiredIntensity, $optimizationDirection);
        exportToJSON(inputData, 'greenwave-input.json');
        break;

      case 'import-input':
        try {
          const imported = await importFromJSON();
          const validation = validateImportedConfig(imported);

          if (!validation.isValid) {
            error.set(`Invalid file: ${validation.errors.join(', ')}`);
            return;
          }

          if (isCleanState) {
            junctions.set(imported.junctions);
            if (imported.desiredSpeed) desiredSpeed.set(imported.desiredSpeed);
            if (imported.desiredIntensity) desiredIntensity.set(imported.desiredIntensity);
            if (imported.direction) optimizationDirection.set(imported.direction);
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

      case 'export-output':
        const outputData = prepareOutputExport($optimizedJunctions, $desiredSpeed, $desiredIntensity, $optimizationDirection);
        exportToJSON(outputData, 'greenwave-output.json');
        break;

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
  message="This will clear all junctions, reset the desired speed, and remove all calculated results. This action cannot be undone."
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
    on:save={saveJunction}
    on:delete={deleteJunction}
    on:close={closeJunctionModal}
  />
{/if}

<div class="min-h-screen bg-gray-50 flex flex-col">
  <div class="container mx-auto p-4 flex-1 flex flex-col">

    <!-- Header -->
    <div class="mb-6">
      <h1 class="text-3xl font-bold text-center mb-4">Green Wave Traffic Light Optimizer</h1>

      <!-- Error banner -->
      {#if $error}
        <div
          transition:slide={{ duration: 200 }}
          class="bg-red-100 border border-red-400 text-red-700 px-4 py-3 rounded-md mb-4 flex justify-between items-start gap-3"
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
    <div class="grid grid-cols-1 lg:grid-cols-2 gap-6 flex-1 min-h-0">

      <!-- LEFT: Input Configuration -->
      <div class="bg-white rounded-lg shadow-md p-6 flex flex-col min-h-0">

        <!-- Panel header: title + wrapping toolbar -->
        <div class="mb-4">
          <div class="flex justify-between items-center mb-3">
            <h2 class="text-xl font-semibold">Input configuration</h2>
          </div>
          <!-- Toolbar wraps on narrow panels instead of clipping -->
          <div class="flex flex-wrap gap-2 items-center">
            <DropdownMenu
              label="File"
              items={fileMenuItems}
              on:select={handleFileMenuSelect}
            />

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

            <!-- Show waves toggle  padded for 44px touch target -->
            <label class="flex items-center gap-2 cursor-pointer select-none py-2 px-1">
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

        <!-- Chart (flex-1 fills available space, controls sit below) -->
        <div class="flex-1 border border-gray-300 rounded-md mb-1 min-h-[280px] overflow-hidden">
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
          {:else}
            <TimeSpaceDiagram
              junctions={$junctions}
              wavesAreOutdated={$wavesAreOutdated}
              interactive={true}
              greenWaves={$originalGreenWaves}
              throughWaves={$originalThroughWaves}
              reverseGreenWaves={$originalReverseGreenWaves}
              reverseThroughWaves={$originalReverseThroughWaves}
              showWaves={$showGreenWaves}
              on:updateJunction={updateJunction}
              on:editSignal={openSignalModal}
              on:editJunction={openJunctionModal}
            />
          {/if}
        </div>

        {#if $junctions.length > 0}
          <div class="mb-3 text-xs text-gray-500 leading-relaxed">
            {#if $wavesAreOutdated.isOutdated}
              <span class="text-orange-600">⚠️ {$wavesAreOutdated.reason}</span>
            {:else}
              Drag junctions to reposition · Click junction label or circle to edit · Click signal line to change
            {/if}
          </div>
        {/if}

        <!-- Controls below chart -> space-y-4 for stable spacing, no mt-auto needed -->
        <div class="border-t pt-4 space-y-4">
          <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
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
            {:else if hasGreenWaveData}
              <span class="text-green-600"> -> green waves calculated</span>
            {:else}
              <span class="text-orange-600"> -> press "Extract waves" to calculate</span>
            {/if}
          </div>

          <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
            <div>
              <label for="input-desired-intensity" class="block text-sm font-medium mb-2">Desired intensity (veh/h)</label>
              <input id="input-desired-intensity" type="number" bind:value={$desiredIntensity} class="w-full px-3 py-2 border rounded-md" min="0" />
            </div>
            <div>
              <label for="input-desired-flow" class="block text-sm font-medium mb-2">Desired flow (veh/s)</label>
              <input id="input-desired-flow" type="number" value={$desiredFlow} class="w-full px-3 py-2 border rounded-md" min="0" step="0.5" on:input={(e) => desiredIntensity.set(e.target.value * 3600)} />
            </div>
          </div>

          <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
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

          {#if $optimizationDirection === 'bidirectional'}
            <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
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
          {/if}
        </div>
      </div>

      <!-- RIGHT: Optimized Results -->
      <div class="bg-white rounded-lg shadow-md p-6 flex flex-col min-h-0">
        <div class="flex justify-between items-center mb-4">
          <h2 class="text-xl font-semibold">Optimized results</h2>
          <button
            on:click={clearResults}
            class="px-3 py-2 bg-red-500 text-white rounded-md hover:bg-red-600 text-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-red-400 focus-visible:ring-offset-1"
          >
            Clear results
          </button>
        </div>

        <!-- Chart (flex-1 fills available space, controls sit below) -->
        <div class="flex-1 border border-gray-300 rounded-md mb-1 min-h-[280px] overflow-hidden">
          {#if $optimizedJunctions.length > 0}
            <TimeSpaceDiagram
              junctions={$optimizedJunctions}
              greenWaves={$optimizedGreenWaves}
              throughWaves={$optimizedThroughWaves}
              reverseGreenWaves={$optimizedReverseGreenWaves}
              reverseThroughWaves={$optimizedReverseThroughWaves}
              showWaves={true}
              interactive={false}
            />
          {:else}
            <div class="flex items-center border-2 border-dashed border-gray-300 rounded-md justify-center h-full text-gray-500 p-6">
              <p class="text-center text-sm">No optimized results yet. Configure input and press Optimize.</p>
            </div>
          {/if}
        </div>

        {#if $optimizedJunctions.length > 0}
          <div class="mb-3 text-xs text-gray-500 leading-relaxed">
            {#if $optimizedResultsAreOutdated.isOutdated}
              <span class="text-orange-600">⚠️ {$optimizedResultsAreOutdated.reason}</span>
            {:else}
              Press Optimize to recalculate with current settings
            {/if}
          </div>
        {/if}

        <!-- Controls below chart -> space-y-4 for stable spacing, no mt-auto needed -->
        <div class="border-t pt-4 space-y-4">
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

          <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
            <div>
              <label for="opt-desired-intensity" class="block text-sm font-medium mb-2">Desired intensity (veh/h)</label>
              <input id="opt-desired-intensity" type="number" bind:value={$desiredIntensity} class="w-full px-3 py-2 border rounded-md" min="0" />
            </div>
            <div>
              <label for="opt-desired-flow" class="block text-sm font-medium mb-2">Desired flow (veh/s)</label>
              <input id="opt-desired-flow" type="number" value={$desiredFlow} class="w-full px-3 py-2 border rounded-md" min="0" step="0.5" on:input={(e) => desiredIntensity.set(e.target.value * 3600)} />
            </div>
          </div>

          <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
            <div>
              <span class="block text-sm font-medium mb-2">Actual intensity{#if $optimizationDirection === 'bidirectional'} <span class="text-green-600">(fwd)</span>{/if}</span>
              <div class="w-full px-3 py-2 border rounded-md bg-gray-100 text-gray-700 tabular-nums">
                {($actualIntensityOptimized || 0).toFixed(2)} veh/h
                {#if hasResults && $optimizedResultsAreOutdated.isOutdated}<span class="text-orange-500 text-xs">(outdated)</span>{/if}
              </div>
            </div>
            <div>
              <span class="block text-sm font-medium mb-2">Actual flow{#if $optimizationDirection === 'bidirectional'} <span class="text-green-600">(fwd)</span>{/if}</span>
              <div class="w-full px-3 py-2 border rounded-md bg-gray-100 text-gray-700 tabular-nums">
                {$actualFlowOptimized.toFixed(6)} veh/s
                {#if hasResults && $optimizedResultsAreOutdated.isOutdated}<span class="text-orange-500 text-xs">(outdated)</span>{/if}
              </div>
            </div>
          </div>

          {#if $optimizationDirection === 'bidirectional'}
            <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
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
          {/if}
        </div>
      </div>

    </div>
  </div>
</div>
