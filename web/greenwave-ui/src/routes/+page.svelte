<script>
  import TimeSpaceDiagram from '../components/TimeSpaceDiagram.svelte';
  import ConfirmModal from '../components/ConfirmModal.svelte';
  import { junctions, desiredSpeed, originalGreenWaves, originalThroughWaves, showGreenWaves, isLoading, error, resetToDemo, resetToEmpty } from '$lib/stores';
  import { extractGreenWaves } from '$lib/api/greenwave.js';
  import { prepareJunctionsForAPI } from '$lib/utils/junction-helpers.js';
  import { onMount } from 'svelte';
  
  // Modal state
  let showResetModal = false;
  let showDemoModal = false;
  
  // Reactive variables
  $: hasGreenWaveData = $originalGreenWaves.length > 0;
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
    
    <!-- Main Content Grid -->
    <div class="grid grid-cols-1 lg:grid-cols-2 gap-6 flex-1 min-h-0">
      <!-- Left side - Optimized Results -->
      <div class="bg-white rounded-lg shadow-md p-6 flex flex-col">
        <div class="flex justify-between items-center mb-4">
          <h2 class="text-xl font-semibold">Optimized Results</h2>
          <button class="px-4 py-2 bg-red-500 text-white rounded hover:bg-red-600">
            Clear Results
          </button>
        </div>
        
        <!-- Results Container -->
        <div class="flex-1 border-2 border-dashed border-gray-300 rounded-lg flex items-center justify-center min-h-0">
          <p class="text-gray-500">Run optimization to see results</p>
        </div>
      </div>
      
      <!-- Right side - Input Data -->
      <div class="bg-white rounded-lg shadow-md p-6 flex flex-col">
        <div class="flex justify-between items-center mb-4">
          <div class="flex justify-between items-center mb-4">
            <h2 class="text-xl font-semibold">Input Configuration</h2>
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
                  class="px-4 py-2 bg-purple-500 text-white rounded hover:bg-purple-600 text-sm w-24 text-center"
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
        </div>
        
        <!-- Chart Container -->
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
              interactive={true}
              greenWaves={$originalGreenWaves}
              throughWaves={$originalThroughWaves}
              showWaves={$showGreenWaves}
            />
          {/if}
        </div>
        
        <!-- Controls -->
        <div class="border-t pt-4 mt-auto">
          <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
            <div>
              <label class="block text-sm font-medium mb-2">Desired Speed (km/h)</label>
              <input 
                type="number" 
                bind:value={$desiredSpeed} 
                class="w-full px-3 py-2 border rounded-md"
                min="10"
                max="100"
              />
            </div>
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
                <p class="text-sm text-orange-600 mt-1">Click "Extract waves" to calculate</p>
              {/if}
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</div>