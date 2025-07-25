<script>
  import TimeSpaceDiagram from '../components/TimeSpaceDiagram.svelte';
  import ConfirmModal from '../components/ConfirmModal.svelte';
  import EditSignalModal from '../components/EditSignalModal.svelte';
  import { isLoading, error, resetToDemo, resetToEmpty } from '$lib/stores';
  import { junctions, desiredSpeed, desiredIntensity, desiredFlow } from '$lib/stores/core';
  import { wavesAreOutdated, originalGreenWaves, originalThroughWaves, showGreenWaves, storeWaveCalculationPositions, actualFlow, actualIntensity } from '$lib/stores/greenwave';
  import { optimizedResultsAreOutdated, optimizedWaveCalculationPositions, optimizedLastCalculatedSpeed, optimizedJunctions, optimizedOffsets, optimizedGreenWaves, optimizedThroughWaves, actualFlowOptimized, actualIntensityOptimized } from '$lib/stores/optimization';
  import { extractGreenWaves } from '$lib/api/greenwave.js';
  import { optimizeOffsets } from '$lib/api/optimize.js';
  import { prepareJunctionsForAPI, applyOffsetsToJunctions } from '$lib/utils/junction-helpers.js';
  import { onMount } from 'svelte';

  // Confirmation modal state
  let showResetModal = false;
  let showDemoModal = false;
  
  // Signal modal state
  let selectedSignal = null;
  let selectedSignalContext = null;
  let isSignalModalOpen = false;

  // Reactive variables
  $: hasGreenWaveData = $originalGreenWaves.length > 0;
  $: hasResults = $optimizedGreenWaves.length > 0 || $optimizedThroughWaves.length > 0;
  $: isExtractDisabled = $isLoading || $junctions.length < 2;
  
  // Helper: Check if we're in "clean" state (no data loss risk)
  $: isCleanState = $junctions.length === 0 && !hasGreenWaveData;
  
  // Extract green waves from API
  async function handleExtractWaves() {
    if (isExtractDisabled) return;
    
    try {
      isLoading.set(true);
      error.set(null);
      
      const junctionsForAPI = prepareJunctionsForAPI($junctions);
      const response = await extractGreenWaves(junctionsForAPI, $desiredSpeed);
      
      originalGreenWaves.set(response.green_waves || []);
      originalThroughWaves.set(response.through_green_waves || []);
      showGreenWaves.set(true);
      storeWaveCalculationPositions($junctions, $desiredSpeed);
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
  }

  // Handle optimization
  async function handleOptimize() {
    try {
      isLoading.set(true);
      error.set(null);

      const junctionsForAPI = prepareJunctionsForAPI($junctions);
      const optimizeResponse = await optimizeOffsets(junctionsForAPI, $desiredSpeed);

      // Update the store with optimized offsets
      optimizedOffsets.set(optimizeResponse.best_offsets || []);

      // Handle the optimization response
      optimizedJunctions.set(applyOffsetsToJunctions($junctions, optimizeResponse.best_offsets));

      // Extract the optimized green waves
      const optimizedJunctionsForAPI = prepareJunctionsForAPI($optimizedJunctions);
      const response = await extractGreenWaves(optimizedJunctionsForAPI, $desiredSpeed);
      optimizedGreenWaves.set(response.green_waves || []);
      optimizedThroughWaves.set(response.through_green_waves || []);

      // Store the current state for validation
      optimizedWaveCalculationPositions.set($junctions.map(j => ({ id: j.id, y: j.point.y })));
      optimizedLastCalculatedSpeed.set($desiredSpeed);
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

{#if isSignalModalOpen}
  <EditSignalModal
    signal={selectedSignal}
    on:save={saveSignal}
    on:close={closeSignalModal}
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
              <label class="block text-sm font-medium mb-2">Desired intensity (vehicles/hour)</label>
              <input 
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
              <label class="block text-sm font-medium mb-2">Desired flow (vehicles/second)</label>
              <input 
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
              <label class="block text-sm font-medium mb-2">Actual intensity (vehicles/hour)</label>
              <div 
                class="w-full px-3 py-2 border rounded-md bg-gray-100 text-gray-700"
              >
                {($actualIntensityOptimized || 0).toFixed(2)}
                {#if hasResults && $optimizedResultsAreOutdated.isOutdated}
                  <span class="text-orange-500">(Outdated)</span>
                {/if}
              </div>
            </div>

            <!-- Actual Flow -->
            <div>
              <label class="block text-sm font-medium mb-2">Actual flow (vehicles/second)</label>
              <div 
                class="w-full px-3 py-2 border rounded-md bg-gray-100 text-gray-700"
              >
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
              <button 
                on:click={handleResetClick}
                class="px-4 py-2 bg-red-500 text-white rounded hover:bg-red-600 text-sm w-24 text-center"
                title="Clear all data and start fresh"
              >
                Reset
              </button>
        
              <button 
                on:click={handleDemoDataClick}
                class="px-4 py-2 bg-gray-500 text-white rounded hover:bg-gray-600 text-sm w-24 text-center"
                title="Load sample configuration"
              >
                Demo data
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
                class="px-4 py-2 bg-purple-500 text-white rounded hover:bg-purple-600 text-sm w-24 text-center"
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
                <button 
                  on:click={handleDemoDataClick}
                  class="px-4 py-2 bg-blue-500 text-white rounded hover:bg-blue-600 text-sm"
                >
                  Load demo data
                </button>
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
              showWaves={$showGreenWaves}
              on:updateJunction={updateJunction}
              isResults={false}
              on:editSignal={openSignalModal}
            />
          {/if}
        </div>
        
        <!-- Controls -->
        <div class="border-t pt-4 mt-auto">
          <div class="grid grid-cols-1 sm:grid-cols-2 gap-4 mb-4">
            <!-- Desired speed -->
            <div>
              <label class="block text-sm font-medium mb-2">Desired speed (km/h)</label>
              <input 
                type="number" 
                bind:value={$desiredSpeed} 
                class="w-full px-3 py-2 border rounded-md"
                min="10"
                max="100"
              />
            </div>
        
            <!-- Junctions -->
            <div>
              <label class="block text-sm font-medium mb-2">Junctions</label>
              <p class="text-sm text-gray-600">{$junctions.length} junctions configured</p>
              
              {#if $junctions.length === 0}
                <p class="text-sm text-blue-600 mt-1">📍 Click "Demo data" or add junctions manually</p>
              {:else if $junctions.length === 1}
                <p class="text-sm text-orange-600 mt-1">⚠️ Add at least 1 more junction to extract waves</p>
              {:else if hasGreenWaveData}
                <p class="text-sm text-green-600 mt-1">✓ Green waves calculated</p>
              {:else}
                <p class="text-sm text-orange-600 mt-1">Press "Extract waves" to calculate green waves</p>
              {/if}
            </div>
          </div>

          <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
            <!-- Desired intensity -->
            <div>
              <label class="block text-sm font-medium mb-2">Desired intensity (vehicles/hour)</label>
              <input 
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
              <label class="block text-sm font-medium mb-2">Desired flow (vehicles/second)</label>
              <input 
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
              <label class="block text-sm font-medium mb-2">Actual intensity (vehicles/hour)</label>
              <div 
                class="w-full px-3 py-2 border rounded-md bg-gray-100 text-gray-700"
              >
                {($actualIntensity || 0).toFixed(2)}
                {#if $wavesAreOutdated.isOutdated}
                  <span class="text-orange-500">(Outdated)</span>
                {/if}
              </div>
            </div>
        
            <!-- Actual flow -->
            <div>
              <label class="block text-sm font-medium mb-2">Actual flow (vehicles/second)</label>
              <div 
                class="w-full px-3 py-2 border rounded-md bg-gray-100 text-gray-700"
              >
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