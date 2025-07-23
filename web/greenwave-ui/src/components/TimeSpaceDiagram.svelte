<script>
  import { onMount } from 'svelte';
  import { createEventDispatcher } from 'svelte';

  import * as d3 from 'd3';

  export let isResults = false;
  export let junctions = [];
  export let wavesAreOutdated = { isOutdated: false, reason: null };
  export let resultsAreOutdated = { isOutdated: false, reason: null };
  export let interactive = false;
  export let greenWaves = [];
  export let throughWaves = [];
  export let showWaves = false;
  
  const dispatch = createEventDispatcher();

  let svg;
  let container;
  let width = 700;
  let height = 400;
  let isDragging = false; 

  const margin = { top: 30, right: 30, bottom: 40, left: 60 };
  const chartWidth = width - margin.left - margin.right;
  const chartHeight = height - margin.top - margin.bottom;
  
  // Helper function to calculate total duration for a junction
  function calculateTotalDuration(junction) {
    return junction.cycle.reduce((total, phase) => {
      return total + phase.signals.reduce((phaseTotal, signal) => {
        return phaseTotal + signal.duration;
      }, 0);
    }, 0);
  }
  
  function updateChart() {
    // Don't update chart while dragging (keep yScale stable)
    if (isDragging) return;

    if (!svg || !junctions.length) return;
    
    const g = d3.select(svg);
    g.selectAll("*").remove();
    
    // Create main group with margins
    const chart = g.append("g")
      .attr("transform", `translate(${margin.left},${margin.top})`);
    
    // Calculate total durations for all junctions
    const junctionsWithDuration = junctions.map(junction => ({
      ...junction,
      total_duration: calculateTotalDuration(junction)
    }));
    
    // Calculate max time domain
    const maxTime = Math.max(...junctionsWithDuration.map(j => j.total_duration));
    
    // Scales
    const xScale = d3.scaleLinear()
      .domain([0, maxTime])
      .range([0, chartWidth]);
    
    const yScale = d3.scaleLinear()
      .domain([0, Math.max(...junctionsWithDuration.map(j => j.point.y)) + 50])
      .range([chartHeight, 0]);
    
    // Draw axes
    chart.append("g")
      .attr("transform", `translate(0,${chartHeight})`)
      .call(d3.axisBottom(xScale))
      .append("text")
      .attr("x", chartWidth / 2)
      .attr("y", 35)
      .attr("fill", "black")
      .style("text-anchor", "middle")
      .text("Time (seconds)");
    
    chart.append("g")
      .call(d3.axisLeft(yScale))
      .append("text")
      .attr("transform", "rotate(-90)")
      .attr("y", -45)
      .attr("x", -chartHeight / 2)
      .attr("fill", "black")
      .style("text-anchor", "middle")
      .text("Distance (meters)");
    
    // Draw waves if enabled
    if (showWaves) {
      if (throughWaves.length > 0) {
        drawThroughWaves(chart, junctionsWithDuration, xScale, yScale, wavesAreOutdated.isOutdated);
      }
      if (greenWaves.length > 0) {
        drawGreenWaves(chart, junctionsWithDuration, xScale, yScale, wavesAreOutdated.isOutdated);
      }
    }
    
    // Draw signal timelines
    drawSignalTimelines(chart, junctionsWithDuration, xScale, yScale);
    
    // Draw junctions
    const junctionGroups = chart.selectAll(".junction")
      .data(junctionsWithDuration)
      .enter()
      .append("g")
      .attr("class", "junction")
      .attr("transform", d => `translate(0, ${yScale(d.point.y)})`)
      .style("cursor", interactive ? "move" : "default");
    
    // Make junctions draggable if interactive
    if (interactive) {
      junctionGroups.call(d3.drag()
        .on("start", function(event, d) {
          console.log("🎯 Drag start:", d.label, "at distance:", d.point.y);
          d3.select(this).style("opacity", 0.8);
          isDragging = true; // Set flag
        })
        .on("drag", function(event, d) {
          // Use the simple working approach with event.y
          const newY = Math.max(0, Math.min(chartHeight, event.y));
          const newDistance = yScale.invert(newY);
          
          console.log("🔄 Dragging:", d.label, "new distance:", Math.round(newDistance));
          
          // Update position immediately for visual feedback
          d3.select(this).attr("transform", `translate(0, ${newY})`);
          
          // Update signal lines for this junction in real-time
          updateSignalLinesForJunction(d.id, newY, junctionsWithDuration, xScale, yScale, chart);
          
          // Emit an event to notify the parent about the updated position
          dispatch("updateJunction", { id: d.id, newDistance: Math.round(newDistance) });
        })
        .on("end", function(event, d) {
          console.log("✅ Drag end:", d.label, "final distance:", d.point.y);
          console.log("📊 All junctions now:", junctions.map(j => `${j.label}: ${j.point.y}m`));
          
          if (wavesAreOutdated.isOutdated) {
            console.log("⚠️ Green waves are now outdated - click 'Extract Waves' to recalculate");
          }
          
          d3.select(this).style("opacity", 1);
          isDragging = false; // Clear flag
          // Force complete redraw after drag ends
          setTimeout(() => updateChart(), 10);
        })
      );
    }
    
    // Draw junction labels with duration
    junctionGroups.append("text")
      .attr("x", 0)
      .attr("y", -15)
      .attr("text-anchor", "middle")
      .attr("font-size", "10px")
      .attr("font-weight", "bold")
      .attr("fill", "#333")
      .text(d => `${d.label || `J${d.id}`}, ${d.total_duration}s`);
    
    // Draw junction circles
    junctionGroups.append("circle")
      .attr("cx", 0)
      .attr("cy", 0)
      .attr("r", 6)
      .attr("fill", "#D8BFD8")
      .attr("stroke", "#4B0082")
      .attr("stroke-width", 2);
  }
  
  // Function to update signal lines for a specific junction during drag
  function updateSignalLinesForJunction(junctionId, newY, junctionsWithDuration, xScale, yScale, chart) {
    const junction = junctionsWithDuration.find(j => j.id === junctionId);
    if (!junction) return;
    
    chart.selectAll(`.signal-line-${junctionId}`).remove();
    
    let currentTime = junction.offset;
    
    junction.cycle.forEach(phase => {
      phase.signals.forEach(signal => {
        if (signal.duration > 0) {
          const startTime = currentTime % junction.total_duration;
          const endTime = (currentTime + signal.duration) % junction.total_duration;
          
          if (endTime < startTime) {
            chart.append("line")
              .attr("class", `signal-line-${junctionId}`)
              .attr("x1", xScale(startTime))
              .attr("x2", xScale(junction.total_duration))
              .attr("y1", newY)
              .attr("y2", newY)
              .attr("stroke", getSignalColor(signal.color))
              .attr("stroke-width", 4);
            
            chart.append("line")
              .attr("class", `signal-line-${junctionId}`)
              .attr("x1", xScale(0))
              .attr("x2", xScale(endTime))
              .attr("y1", newY)
              .attr("y2", newY)
              .attr("stroke", getSignalColor(signal.color))
              .attr("stroke-width", 4);
          } else {
            chart.append("line")
              .attr("class", `signal-line-${junctionId}`)
              .attr("x1", xScale(startTime))
              .attr("x2", xScale(endTime))
              .attr("y1", newY)
              .attr("y2", newY)
              .attr("stroke", getSignalColor(signal.color))
              .attr("stroke-width", 4);
          }
        }
        currentTime += signal.duration;
      });
    });
  }
  
  // Draw signal timelines function
  function drawSignalTimelines(chart, junctionsWithDuration, xScale, yScale) {
    junctionsWithDuration.forEach((junction, jIdx) => {
      let currentTime = junction.offset;
      const y = yScale(junction.point.y);
      
      junction.cycle.forEach(phase => {
        phase.signals.forEach(signal => {
          if (signal.duration > 0) {
            const startTime = currentTime % junction.total_duration;
            const endTime = (currentTime + signal.duration) % junction.total_duration;
            
            if (endTime < startTime) {
              chart.append("line")
                .attr("class", `signal-line-${junction.id}`)
                .attr("x1", xScale(startTime))
                .attr("x2", xScale(junction.total_duration))
                .attr("y1", y)
                .attr("y2", y)
                .attr("stroke", getSignalColor(signal.color))
                .attr("stroke-width", 4);
              
              chart.append("line")
                .attr("class", `signal-line-${junction.id}`)
                .attr("x1", xScale(0))
                .attr("x2", xScale(endTime))
                .attr("y1", y)
                .attr("y2", y)
                .attr("stroke", getSignalColor(signal.color))
                .attr("stroke-width", 4);
            } else {
              chart.append("line")
                .attr("class", `signal-line-${junction.id}`)
                .attr("x1", xScale(startTime))
                .attr("x2", xScale(endTime))
                .attr("y1", y)
                .attr("y2", y)
                .attr("stroke", getSignalColor(signal.color))
                .attr("stroke-width", 4);
            }
          }
          currentTime += signal.duration;
        });
      });
    });
  }

  // Draw green waves with visual indication if outdated
  function drawGreenWaves(chart, junctionsWithDuration, xScale, yScale, isOutdated = false) {
    const waveColor = "#57B844";
    const alpha = isOutdated ? 0.15 : 0.3; // Fade if outdated
    
    greenWaves.forEach((segmentWaves, segmentIdx) => {
      if (segmentIdx >= junctionsWithDuration.length - 1) return;
      
      const j1 = junctionsWithDuration[segmentIdx];
      const j2 = junctionsWithDuration[segmentIdx + 1];
      const y1 = yScale(j1.point.y);
      const y2 = yScale(j2.point.y);
      
      segmentWaves.forEach(wave => {
        const startJ1 = wave.interval_jun_one.start;
        const endJ1 = wave.interval_jun_one.end;
        const startJ2 = wave.interval_jun_two.start;
        const endJ2 = wave.interval_jun_two.end;
        
        const polygonPoints = [
          [xScale(startJ1), y1],
          [xScale(startJ2), y2],
          [xScale(endJ2), y2],
          [xScale(endJ1), y1]
        ];
        
        chart.append("polygon")
          .attr("points", polygonPoints.map(p => p.join(",")).join(" "))
          .attr("fill", waveColor)
          .attr("fill-opacity", alpha)
          .attr("stroke", isOutdated ? "#ff6b35" : waveColor) // Orange border if outdated
          .attr("stroke-width", isOutdated ? 1 : 0.5)
          .attr("stroke-opacity", isOutdated ? 0.6 : 0.8)
          .attr("stroke-dasharray", isOutdated ? "3,3" : "none"); // Dashed if outdated
      });
    });
  }
  
  // Draw through waves with visual indication if outdated
  function drawThroughWaves(chart, junctionsWithDuration, xScale, yScale, isOutdated = false) {
    const waveColor = "#541FE4";
    const alpha = isOutdated ? 0.1 : 0.2; // Fade if outdated
    
    throughWaves.forEach(wave => {
      const starts = [];
      const ends = [];
      
      wave.intervals.forEach((interval, junctionIdx) => {
        if (junctionIdx < junctionsWithDuration.length) {
          const junction = junctionsWithDuration[junctionIdx];
          const y = yScale(junction.point.y);
          
          starts.push([xScale(interval.start), y]);
          ends.push([xScale(interval.end), y]);
        }
      });
      
      ends.reverse();
      const polygonPoints = [...starts, ...ends];
      
      chart.append("polygon")
        .attr("points", polygonPoints.map(p => p.join(",")).join(" "))
        .attr("fill", waveColor)
        .attr("fill-opacity", alpha)
        .attr("stroke", isOutdated ? "#ff6b35" : waveColor) // Orange border if outdated
        .attr("stroke-width", isOutdated ? 1 : 0.5)
        .attr("stroke-opacity", isOutdated ? 0.6 : 0.8)
        .attr("stroke-dasharray", isOutdated ? "3,3" : "none"); // Dashed if outdated
    });
  }
  
  function getSignalColor(color) {
    const colorMap = {
      'RED': '#dc2626',
      'YELLOW': '#fbbf24', 
      'GREEN': '#16a34a',
      'GREENPRIORITY': '#15803d'
    };
    return colorMap[color] || '#000000';
  }
  
  // Update chart when ANY relevant state changes
  $: if (svg) {
    updateChart();
  }
  
  // Also reactive to all prop changes  
  $: greenWaves, throughWaves, showWaves, wavesAreOutdated, updateChart();
  
  function updateSize() {
    if (container) {
      width = container.clientWidth;
      height = container.clientHeight;
      updateChart();
    }
  }
  
  onMount(() => {
    updateSize();
    window.addEventListener('resize', updateSize);
    return () => window.removeEventListener('resize', updateSize);
  });
</script>

<div bind:this={container} class="diagram-container w-full h-full relative">
  <svg bind:this={svg} {width} {height} class="w-full h-full"></svg>
  {#if junctions.length > 0}
    <div class="absolute bottom-3 right-3 bg-white bg-opacity-90 rounded-lg px-3 py-2 text-xs text-gray-600 shadow-sm border border-gray-200">
      <div class="flex items-center gap-2">
        <svg class="w-4 h-4 text-blue-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"></path>
        </svg>
        {#if !isResults}
          <span>💡 <strong>Drag junctions</strong> to change distance</span>
        {:else}
          <span>💡 <strong>Press 'Optimize' to refresh</strong></span>
        {/if}
        {#if resultsAreOutdated.isOutdated}
          <span class="text-orange-600 ml-2">⚠️ Results outdated: {resultsAreOutdated.reason}</span>
        {:else if wavesAreOutdated.isOutdated}
          <span class="text-orange-600 ml-2">⚠️ Waves outdated: {wavesAreOutdated.reason}</span>
        {/if}
      </div>
    </div>
  {/if}
</div>

<style>
  .diagram-container {
    width: 100%;
    height: 100%;
    min-height: 300px;
    position: relative;
  }
</style>