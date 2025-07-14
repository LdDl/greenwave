<script>
  import TimeSpaceDiagram from '../components/TimeSpaceDiagram.svelte';
  import { junctions, desiredSpeed, originalGreenWaves, originalThroughWaves, showGreenWaves, isLoading, error } from '$lib/stores';
  import { extractGreenWaves } from '$lib/api/greenwave.js';
  import { prepareJunctionsForAPI } from '$lib/utils/junction-helpers.js';
  
  // Sample data for testing
  const sampleJunctions = [
    {
      id: 0,
      label: "Junction 1",
      cycle: [
        {
          id: 0,
          signals: [
            { duration: 30, color: "GREEN" },
            { duration: 20, color: "RED" }
          ]
        },
        {
          id: 1,
          signals: [
            { duration: 20, color: "GREEN" },
            { duration: 15, color: "RED" }
          ]
        }
      ],
      offset: 0,
      point: { x: 0, y: 0 }
    },
    {
      id: 1,
      label: "Junction 2",
      cycle: [
        {
          id: 10,
          signals: [
            { duration: 20, color: "RED" },
            { duration: 35, color: "GREEN" },
            { duration: 5, color: "YELLOW" }
          ]
        },
        {
          id: 11,
          signals: [
            { duration: 10, color: "RED" },
            { duration: 10, color: "GREEN" },
            { duration: 5, color: "YELLOW" }
          ]
        }
      ],
      offset: 0,
      point: { x: 0, y: 200 }
    },
    {
      id: 2,
      label: "Junction 3",
      cycle: [
        {
          id: 20,
          signals: [
            { duration: 45, color: "RED" },
            { duration: 10, color: "GREEN" }
          ]
        },
        {
          id: 21,
          signals: [
            { duration: 7, color: "RED" },
            { duration: 18, color: "GREEN" },
            { duration: 5, color: "YELLOW" }
          ]
        }
      ],
      offset: 0,
      point: { x: 0, y: 450 }
    },
    {
      id: 3,
      label: "Junction 4",
      cycle: [
        {
          id: 20,
          signals: [
            { duration: 40, color: "RED" },
            { duration: 15, color: "GREEN" }
          ]
        },
        {
          id: 21,
          signals: [
            { duration: 10, color: "RED" },
            { duration: 20, color: "GREEN" }
          ]
        }
      ],
      offset: 0,
      point: { x: 0, y: 600 }
    }
  ];
  
  // Set sample data
  junctions.set(sampleJunctions);
  
  // Reactive variables
  $: hasGreenWaveData = $originalGreenWaves.length > 0;
  $: isExtractDisabled = $isLoading || $junctions.length < 2;
  
  // Extract green waves from API
  async function handleExtractWaves() {
    if (isExtractDisabled) return;
    
    try {
      isLoading.set(true);
      error.set(null);
      
      // Prepare junctions for API
      const junctionsForAPI = prepareJunctionsForAPI($junctions);
      
      // Call API
      const response = await extractGreenWaves(junctionsForAPI, $desiredSpeed);
      
      // Update stores with response data
      originalGreenWaves.set(response.green_waves || []);
      originalThroughWaves.set(response.through_green_waves || []);
      
      // Enable and turn ON the show waves toggle
      showGreenWaves.set(true);
      
    } catch (apiError) {
      error.set(apiError.message || 'Failed to extract green waves');
      console.error('API Error:', apiError);
    } finally {
      isLoading.set(false);
    }
  }
  
  // Toggle show/hide waves
  function handleToggleWaves() {
    if (hasGreenWaveData) {
      showGreenWaves.update(current => !current);
    }
  }
</script>

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
        
        <!-- Results Container - Fixed height and overflow -->
        <div class="flex-1 border-2 border-dashed border-gray-300 rounded-lg flex items-center justify-center min-h-0">
          <p class="text-gray-500">Run optimization to see results</p>
        </div>
      </div>
      
      <!-- Right side - Input Data -->
      <div class="bg-white rounded-lg shadow-md p-6 flex flex-col">
        <div class="flex justify-between items-center mb-4">
          <h2 class="text-xl font-semibold">Input Configuration</h2>
          <div class="flex gap-2 items-center">
            <!-- Show Waves Toggle -->
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
                Show Waves
              </span>
            </label>
            
            <!-- Extract Waves Button -->
            <button 
              on:click={handleExtractWaves}
              disabled={isExtractDisabled}
              class="px-3 py-1 bg-green-500 text-white rounded hover:bg-green-600 text-sm disabled:bg-gray-400 disabled:cursor-not-allowed flex items-center gap-2"
            >
              {#if $isLoading}
                <div class="w-4 h-4 border-2 border-white border-t-transparent rounded-full animate-spin"></div>
                Loading...
              {:else}
                Extract Waves
              {/if}
            </button>
            
            <!-- Optimize Button -->
            <button class="px-3 py-1 bg-purple-500 text-white rounded hover:bg-purple-600 text-sm">
              Optimize
            </button>
          </div>
        </div>
        
        <!-- Chart Container - Fixed height and overflow -->
        <div class="flex-1 border border-gray-300 rounded mb-4 min-h-0 overflow-hidden">
          <TimeSpaceDiagram 
            interactive={true}
            greenWaves={$originalGreenWaves}
            throughWaves={$originalThroughWaves}
            showWaves={$showGreenWaves}
          />
        </div>
        
        <!-- Controls - Fixed at bottom -->
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
              {#if hasGreenWaveData}
                <p class="text-sm text-green-600 mt-1">✓ Green waves calculated</p>
              {/if}
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</div>