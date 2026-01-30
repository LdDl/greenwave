<script>
  import TimeSpaceDiagram from '../components/TimeSpaceDiagram.svelte';
  import ConfirmModal from '../components/ConfirmModal.svelte';
  import EditSignalModal from '../components/EditSignalModal.svelte';
  import EditJunctionModal from '../components/EditJunctionModal.svelte';
  import DropdownMenu from '../components/DropdownMenu.svelte';
  import { isLoading, error, resetToDemo, resetToEmpty } from '$lib/stores';
  import { exportToJSON, importFromJSON, validateImportedConfig, prepareInputExport, prepareOutputExport } from '$lib/utils/export-import.js';
  import { junctions, desiredSpeed, desiredIntensity, desiredFlow, optimizationDirection } from '$lib/stores/core';
  import { wavesAreOutdated, originalGreenWaves, originalThroughWaves, originalReverseGreenWaves, originalReverseThroughWaves, showGreenWaves, storeWaveCalculationPositions, actualFlow, actualIntensity } from '$lib/stores/greenwave';
  import { optimizedResultsAreOutdated, optimizedWaveCalculationPositions, optimizedLastCalculatedSpeed, optimizedJunctions, optimizedOffsets, optimizedGreenWaves, optimizedThroughWaves, optimizedReverseGreenWaves, optimizedReverseThroughWaves, actualFlowOptimized, actualIntensityOptimized } from '$lib/stores/optimization';
  import { invalidateSignals, resetResultsInvalidation } from '$lib/stores/signals';
  import { extractGreenWaves } from '$lib/api/greenwave.js';
  import { optimizeOffsets } from '$lib/api/optimize.js';
  import { prepareJunctionsForAPI, applyOffsetsToJunctions, validateJunctionCycles, calculateTotalDuration } from '$lib/utils/junction-helpers.js';
  import { onMount } from 'svelte';
  import { invalidateAll, validateInput, validateResults } from '$lib/stores/invalidation';

  onMount(() => {
    // Mark stores as initialized after the first render
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
  
  // Helper: Check if we're in "clean" state (no data loss risk)
  $: isCleanState = $junctions.length === 0 && !hasGreenWaveData;
  
  // Extract green waves from API
  async function handleExtractWaves() {
    if (isExtractDisabled) return;
    
    try {
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
      isLoading.set(false);
    }
  }
  
  // Smart modal handlers
  function handleResetClick() {
    if (isCleanState) {
      // Already clean - no need to confirm
      resetToEmpty();
    } else {
      // Has data - show confirmation
      showResetModal = true;
    }
  }
  
  function handleDemoDataClick() {
    if (isCleanState) {
      // No data to lose - load directly
      resetToDemo();
    } else {
      // Has data - show confirmation
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
  // Update the store with new distance
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
    try {
      isLoading.set(true);
      error.set(null);

      const junctionsForAPI = prepareJunctionsForAPI($junctions);
      const optimizeResponse = await optimizeOffsets(junctionsForAPI, $desiredSpeed, 'genetic', {}, $optimizationDirection);

      // Update the store with optimized offsets
      optimizedOffsets.set(optimizeResponse.best_offsets || []);

      // Handle the optimization response
      optimizedJunctions.set(applyOffsetsToJunctions($junctions, optimizeResponse.best_offsets));

      // Extract the optimized green waves
      const optimizedJunctionsForAPI = prepareJunctionsForAPI($optimizedJunctions);
      const response = await extractGreenWaves(optimizedJunctionsForAPI, $desiredSpeed, $optimizationDirection);
      optimizedGreenWaves.set(response.green_waves || []);
      optimizedThroughWaves.set(response.through_green_waves || []);
      optimizedReverseGreenWaves.set(response.reverse_green_waves || []);
      optimizedReverseThroughWaves.set(response.reverse_through_green_waves || []);

      // Store the current state for validation
      optimizedWaveCalculationPositions.set($junctions.map(j => ({ id: j.id, y: j.point.y })));
      optimizedLastCalculatedSpeed.set($desiredSpeed);

      validateResults()
    } catch (optimizeError) {
      error.set(optimizeError.message || 'Failed to optimize');
      console.error('Optimize Error:', optimizeError);
    } finally {
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

    console.log("Saving signal:", updatedSignal);
    console.log("Selected signal context:", selectedSignalContext);

    // Ensure selectedSignalContext has the full context
    if (!selectedSignalContext || !selectedSignalContext.junction || !selectedSignalContext.phase) {
      console.error("Invalid selectedSignalContext:", selectedSignalContext);
      return;
    }

    // Update the signal in the corresponding junction
    junctions.update((junctionList) => {
      const updatedJunctions = junctionList.map((junction) => {
        if (junction.id === selectedSignalContext.junction.id) {
          return {
            ...junction,
            cycle: junction.cycle.map((phase) => {
              if (phase === selectedSignalContext.phase) {
                return {
                  ...phase,
                  signals: phase.signals.map((signal) => {
                    if (signal === selectedSignal) {
                      return { ...signal, ...updatedSignal }; // Update the signal
                    }
                    return signal;
                  }),
                };
              }
              return phase;
            }),
          };
        }
        return junction;
      });

      return updatedJunctions; // Reassign the store to trigger reactivity
    });

    invalidateAll('signal changes');

    // Close the modal
    closeSignalModal();
  }

  function openSignalModal(event) {
    const { junction, phase, signal } = event.detail;
    // Store the context for saving
    selectedSignalContext = { junction, phase };
    // Pass only the signal to the modal
    selectedSignal = signal;
    isSignalModalOpen = true;
  }

  function closeSignalModal() {
    selectedSignal = null;
    isSignalModalOpen = false;
  }

  // Junction modal handlers
  function openJunctionModal(event) {
    const { junction } = event.detail;
    // Find the original junction from the store (not the one with calculated total_duration)
    const originalJunction = $junctions.find(j => j.id === junction.id);
    selectedJunction = originalJunction || junction;
    isNewJunction = false;
    isJunctionModalOpen = true;
  }

  function openNewJunctionModal() {
    // Create a new junction with default values
    const maxId = $junctions.length > 0 ? Math.max(...$junctions.map(j => j.id)) : -1;
    const maxY = $junctions.length > 0 ? Math.max(...$junctions.map(j => j.point.y)) : -100;

    selectedJunction = {
      id: maxId + 1,
      label: `Junction ${maxId + 2}`,
      cycle: [
        {
          id: (maxId + 1) * 10,
          signals: [
            { duration: 30, color: 'GREEN' },
            { duration: 20, color: 'RED' }
          ]
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
      // Add new junction
      junctions.update(junctionList => [...junctionList, junction]);
    } else {
      // Update existing junction
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

  function startDiagram() {
    console.log("Starting diagram with empty junctions");
    junctions.set([]);
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
            // No data to lose - import directly
            junctions.set(imported.junctions);
            if (imported.desiredSpeed) desiredSpeed.set(imported.desiredSpeed);
            if (imported.desiredIntensity) desiredIntensity.set(imported.desiredIntensity);
            if (imported.direction) optimizationDirection.set(imported.direction);
            invalidateAll('configuration imported');
          } else {
            // Has data - show confirmation
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

<!-- Reset Confirmation Modal -->
<ConfirmModal 
  bind:show={showResetModal}
  title="Reset All Data"
  message="This will clear all junctions, reset the desired speed, and remove all calculated results. This action cannot be undone."
  confirmText="Reset"
  cancelText="Cancel"
  onConfirm={confirmReset}
  danger={true}
/>

<!-- Demo Data Confirmation Modal -->
<ConfirmModal
  bind:show={showDemoModal}
  title="Load Demo Data"
  message="This will replace your current configuration with sample data and clear all calculated results."
  confirmText="Load Demo Data"
  cancelText="Cancel"
  onConfirm={confirmDemoData}
  danger={false}
/>

<!-- Import Confirmation Modal -->
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
      
      <!-- Error Message -->
      {#if $error}
        <div class="bg-red-100 border border-red-400 text-red-700 px-4 py-3 rounded mb-4">
          <strong>Error:</strong> {$error}
        </div>
      {/if}
    </div>
    
    <!-- Main content grid -->
    <div class="grid grid-cols-1 lg:grid-cols-2 gap-6 flex-1 min-h-0">
      <!-- Left side - optimized results -->
      <div class="bg-white rounded-lg shadow-md p-6 flex flex-col flex-1 min-h-0">
        <div class="flex justify-between items-center mb-4">
          <h2 class="text-xl font-semibold">Optimized results</h2>
          <button
            on:click={clearResults}
            class="px-4 py-2 bg-red-500 text-white rounded hover:bg-red-600 text-sm w-24 text-center"
            >
            Clear results
          </button>
        </div>
        
        <!-- Results chart container -->
        <div class="flex-1 border border-gray-300 rounded-lg mb-4 min-h-0 overflow-hidden">
          {#if $optimizedJunctions.length > 0}
            <TimeSpaceDiagram
              junctions={$optimizedJunctions}
              greenWaves={$optimizedGreenWaves}
              throughWaves={$optimizedThroughWaves}
              reverseGreenWaves={$optimizedReverseGreenWaves}
              reverseThroughWaves={$optimizedReverseThroughWaves}
              showWaves={true}
              interactive={false}
              resultsAreOutdated={$optimizedResultsAreOutdated}
              isResults={true}
            />
          {:else}
            <div class="flex items-center border-2 border-dashed border-gray-300 rounded-lg justify-center h-full text-gray-500">
              <p>No optimized results to display. Run optimization to see results.</p>
            </div>
          {/if}
        </div>

        <!-- Controls for results -->
        <div class="border-t pt-4 mt-auto">
          <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
            <!-- Desired intensity -->
            <div>
              <label for="opt-desired-intensity" class="block text-sm font-medium mb-2">Desired intensity (vehicles/hour)</label>
              <input
                id="opt-desired-intensity"
                type="number"
                bind:value={$desiredIntensity}
                class="w-full px-3 py-2 border rounded-md"
                min="0"
                class:border-orange-500={hasResults && $optimizedResultsAreOutdated.isOutdated}
                class:border-black-300={!hasResults || !$optimizedResultsAreOutdated.isOutdated}
                disabled={hasResults && $optimizedResultsAreOutdated.isOutdated}
              />
            </div>

            <!-- Desired Flow -->
            <div>
              <label for="opt-desired-flow" class="block text-sm font-medium mb-2">Desired flow (vehicles/second)</label>
              <input
                id="opt-desired-flow"
                type="number"
                value={$desiredFlow}
                class="w-full px-3 py-2 border rounded-md"
                min="0"
                step="0.5"
                class:border-orange-500={hasResults && $optimizedResultsAreOutdated.isOutdated}
                class:border-black-300={!hasResults || !$optimizedResultsAreOutdated.isOutdated}
                on:input={(e) => desiredIntensity.set(e.target.value * 3600)}
                disabled={hasResults && $optimizedResultsAreOutdated.isOutdated}
              />
            </div>

            <!-- Actual Intensity -->
            <div>
              <span class="block text-sm font-medium mb-2">Actual intensity (vehicles/hour)</span>
              <div class="w-full px-3 py-2 border rounded-md bg-gray-100 text-gray-700">
                {($actualIntensityOptimized || 0).toFixed(2)}
                {#if hasResults && $optimizedResultsAreOutdated.isOutdated}
                  <span class="text-orange-500">(Outdated)</span>
                {/if}
              </div>
            </div>

            <!-- Actual Flow -->
            <div>
              <span class="block text-sm font-medium mb-2">Actual flow (vehicles/second)</span>
              <div class="w-full px-3 py-2 border rounded-md bg-gray-100 text-gray-700">
                {$actualFlowOptimized.toFixed(6)}
                {#if hasResults && $optimizedResultsAreOutdated.isOutdated}
                  <span class="text-orange-500">(Outdated)</span>
                {/if}
              </div>
            </div>
          </div>
        </div>
      </div>
      
      <!-- Right side - Input Data -->
      <div class="bg-white rounded-lg shadow-md p-6 flex flex-col flex-1 min-h-0">
        <div class="flex justify-between items-center mb-4">
          <h2 class="text-xl font-semibold">Input configuration</h2>
          <div class="flex gap-4 items-center">
            <div class="flex gap-2">
              <DropdownMenu
                label="File"
                items={fileMenuItems}
                on:select={handleFileMenuSelect}
              />

              <button
                on:click={openNewJunctionModal}
                class="px-4 py-2 bg-blue-500 text-white rounded hover:bg-blue-600 text-sm w-32 text-center"
                title="Add a new junction"
              >
                + Add Junction
              </button>

              <button
                on:click={handleExtractWaves}
                disabled={isExtractDisabled}
                class="px-4 py-2 bg-green-500 text-white rounded hover:bg-green-600 text-sm disabled:bg-gray-400 disabled:cursor-not-allowed flex items-center justify-center gap-2 w-32"
              >
                {#if $isLoading}
                  <div class="w-4 h-4 border-2 border-white border-t-transparent rounded-full animate-spin"></div>
                  Loading...
                {:else}
                  Extract waves
                {/if}
              </button>

              <button
                on:click={handleOptimize}
                disabled={isExtractDisabled}
                class="px-4 py-2 bg-purple-500 text-white rounded hover:bg-purple-600 text-sm w-24 text-center disabled:bg-gray-400 disabled:cursor-not-allowed"
                title="Recalculate waves and optimize offsets"
              >
                Optimize
              </button>
            </div>
            
            <!-- Toggle Group - Separate -->
            <label class="flex items-center cursor-pointer">
              <input 
                type="checkbox" 
                bind:checked={$showGreenWaves}
                disabled={!hasGreenWaveData}
                class="mr-2"
                class:opacity-50={!hasGreenWaveData}
                class:cursor-not-allowed={!hasGreenWaveData}
              />
              <span class="text-sm" class:text-gray-400={!hasGreenWaveData}>
                Show waves
              </span>
            </label>
          </div>
        </div>
        
        <!-- Chart container -->
        <div class="flex-1 border border-gray-300 rounded mb-4 min-h-0 overflow-hidden">
          {#if $junctions.length === 0}
            <!-- Empty State -->
            <div class="flex items-center justify-center h-full text-gray-500">
              <div class="text-center">
                <svg class="w-16 h-16 mx-auto mb-4 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 6v6m0 0v6m0-6h6m-6 0H6"></path>
                </svg>
                <h3 class="text-lg font-medium mb-2">No junctions configured</h3>
                <p class="text-sm mb-4">Add junctions to start visualizing traffic light coordination</p>
                <div class="flex gap-2 justify-center">
                  <button
                    on:click={openNewJunctionModal}
                    class="px-4 py-2 bg-blue-500 text-white rounded hover:bg-blue-600 text-sm"
                  >
                    + Add First Junction
                  </button>
                  <button
                    on:click={handleDemoDataClick}
                    class="px-4 py-2 bg-gray-500 text-white rounded hover:bg-gray-600 text-sm"
                  >
                    Load Demo Data
                  </button>
                </div>
              </div>
            </div>
          {:else}
            <!-- Regular Chart -->
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
              isResults={false}
              on:editSignal={openSignalModal}
              on:editJunction={openJunctionModal}
            />
          {/if}
        </div>
        
        <!-- Controls -->
        <div class="border-t pt-4 mt-auto">
          <!-- Optimization direction -->
          <div class="mb-4">
            <label for="input-direction" class="block text-sm font-medium mb-2">Optimization direction</label>
            <select
              id="input-direction"
              bind:value={$optimizationDirection}
              class="w-full px-3 py-2 border rounded-md"
            >
              <option value="forward">Forward only</option>
              <option value="bidirectional">Bidirectional</option>
            </select>
          </div>

          <div class="grid grid-cols-1 sm:grid-cols-2 gap-4 mb-4">
            <!-- Desired speed -->
            <div>
              <label for="input-desired-speed" class="block text-sm font-medium mb-2">Desired speed (km/h)</label>
              <input
                id="input-desired-speed"
                type="number"
                bind:value={$desiredSpeed}
                class="w-full px-3 py-2 border rounded-md"
                min="10"
                max="100"
              />
            </div>

            <!-- Junctions -->
            <div>
              <span class="block text-sm font-medium mb-2">Junctions</span>
              <p class="text-sm text-gray-600">{$junctions.length} junctions configured</p>

              {#if $junctions.length === 0}
                <p class="text-sm text-blue-600 mt-1">📍 Use File menu or add junctions manually</p>
              {:else if $junctions.length === 1}
                <p class="text-sm text-orange-600 mt-1">⚠️ Add at least 1 more junction to extract waves</p>
              {:else if hasValidationError}
                <p class="text-sm text-red-600 mt-1">⚠️ {validationErrorMessage}</p>
              {:else if hasGreenWaveData}
                <p class="text-sm text-green-600 mt-1">✓ Green waves calculated</p>
              {:else}
                <p class="text-sm text-orange-600 mt-1">Press "Extract waves" to calculate green waves</p>
              {/if}
            </div>
          </div>

          <div class="grid grid-cols-1 sm:grid-cols-2 gap-4 mb-4">
            <!-- Desired intensity -->
            <div>
              <label for="input-desired-intensity" class="block text-sm font-medium mb-2">Desired intensity (vehicles/hour)</label>
              <input
                id="input-desired-intensity"
                type="number"
                bind:value={$desiredIntensity}
                class="w-full px-3 py-2 border rounded-md"
                min="0"
                class:border-orange-500={$wavesAreOutdated.isOutdated}
                class:border-black-300={!$wavesAreOutdated.isOutdated}
                disabled={$wavesAreOutdated.isOutdated}
              />
            </div>

            <!-- Desired flow -->
            <div>
              <label for="input-desired-flow" class="block text-sm font-medium mb-2">Desired flow (vehicles/second)</label>
              <input
                id="input-desired-flow"
                type="number"
                value={$desiredFlow}
                class="w-full px-3 py-2 border rounded-md"
                min="0"
                step="0.5"
                class:border-orange-500={$wavesAreOutdated.isOutdated}
                class:border-black-300={!$wavesAreOutdated.isOutdated}
                on:input={(e) => desiredIntensity.set(e.target.value * 3600)}
                disabled={$wavesAreOutdated.isOutdated}
              />
            </div>

            <!-- Actual intensity -->
            <div>
              <span class="block text-sm font-medium mb-2">Actual intensity (vehicles/hour)</span>
              <div class="w-full px-3 py-2 border rounded-md bg-gray-100 text-gray-700">
                {($actualIntensity || 0).toFixed(2)}
                {#if $wavesAreOutdated.isOutdated}
                  <span class="text-orange-500">(Outdated)</span>
                {/if}
              </div>
            </div>

            <!-- Actual flow -->
            <div>
              <span class="block text-sm font-medium mb-2">Actual flow (vehicles/second)</span>
              <div class="w-full px-3 py-2 border rounded-md bg-gray-100 text-gray-700">
                {($actualFlow || 0).toFixed(6)}
                {#if $wavesAreOutdated.isOutdated}
                  <span class="text-orange-500">(Outdated)</span>
                {/if}
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</div>