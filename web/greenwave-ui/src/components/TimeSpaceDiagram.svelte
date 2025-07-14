<script>
  import { onMount } from 'svelte';
  import * as d3 from 'd3';
  import { junctions } from '$lib/stores';
  
  export let greenWaves = [];
  export let throughWaves = [];
  export let showWaves = false;
  
  let svg;
  let container;
  let width = 700;
  let height = 400;
  
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
    if (!svg || !$junctions.length) return;
    
    const g = d3.select(svg);
    g.selectAll("*").remove();
    
    // Create main group with margins
    const chart = g.append("g")
      .attr("transform", `translate(${margin.left},${margin.top})`);
    
    // Calculate total durations for all junctions
    const junctionsWithDuration = $junctions.map(junction => ({
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
    
    // Draw waves if enabled (order matters for layering)
    if (showWaves) {
      // Draw through waves FIRST (deepest layer)
      if (throughWaves.length > 0) {
        drawThroughWaves(chart, junctionsWithDuration, xScale, yScale);
      }
      
      // Draw green waves SECOND (on top of through waves)
      if (greenWaves.length > 0) {
        drawGreenWaves(chart, junctionsWithDuration, xScale, yScale);
      }
    }
    
    // Draw junctions
    const junctionGroups = chart.selectAll(".junction")
      .data(junctionsWithDuration)
      .enter()
      .append("g")
      .attr("class", "junction")
      .attr("transform", d => `translate(0, ${yScale(d.point.y)})`);
    
    // Draw junction labels with duration (combined in one text element)
    junctionGroups.append("text")
      .attr("x", 0)
      .attr("y", -15)
      .attr("text-anchor", "middle")
      .attr("font-size", "10px")
      .attr("font-weight", "bold")
      .attr("fill", "#333")
      .text(d => `${d.label || `J${d.id}`}, ${d.total_duration}s`);
    
    // Draw signal timelines
    junctionsWithDuration.forEach((junction, jIdx) => {
      let currentTime = junction.offset;
      const y = yScale(junction.point.y);
      
      junction.cycle.forEach(phase => {
        phase.signals.forEach(signal => {
          if (signal.duration > 0) {
            const startTime = currentTime % junction.total_duration;
            const endTime = (currentTime + signal.duration) % junction.total_duration;
            
            // Handle wrap-around case
            if (endTime < startTime) {
              // First part (to end of cycle)
              chart.append("line")
                .attr("x1", xScale(startTime))
                .attr("x2", xScale(junction.total_duration))
                .attr("y1", y)
                .attr("y2", y)
                .attr("stroke", getSignalColor(signal.color))
                .attr("stroke-width", 4);
              
              // Second part (from start of cycle)
              chart.append("line")
                .attr("x1", xScale(0))
                .attr("x2", xScale(endTime))
                .attr("y1", y)
                .attr("y2", y)
                .attr("stroke", getSignalColor(signal.color))
                .attr("stroke-width", 4);
            } else {
              // Normal case
              chart.append("line")
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
    
    // Draw junction circles
    junctionGroups.append("circle")
      .attr("cx", 0)
      .attr("cy", 0)
      .attr("r", 6)
      .attr("fill", "#D8BFD8")
      .attr("stroke", "#4B0082")
      .attr("stroke-width", 2);
  }
  
  // Draw green waves function (matches your Python implementation)
  function drawGreenWaves(chart, junctionsWithDuration, xScale, yScale) {
    const waveColor = "#57B844";
    const alpha = 0.3;
    
    // For each segment between junctions
    greenWaves.forEach((segmentWaves, segmentIdx) => {
      if (segmentIdx >= junctionsWithDuration.length - 1) {
        // Protection against segment/junction mismatch
        return;
      }
      
      const j1 = junctionsWithDuration[segmentIdx];
      const j2 = junctionsWithDuration[segmentIdx + 1];
      const y1 = yScale(j1.point.y);
      const y2 = yScale(j2.point.y);
      
      // For each green wave in the segment
      segmentWaves.forEach(wave => {
        const startJ1 = wave.interval_jun_one.start;
        const endJ1 = wave.interval_jun_one.end;
        const startJ2 = wave.interval_jun_two.start;
        const endJ2 = wave.interval_jun_two.end;
        
        // Create polygon points (matching your Python logic)
        const polygonPoints = [
          [xScale(startJ1), y1],
          [xScale(startJ2), y2],
          [xScale(endJ2), y2],
          [xScale(endJ1), y1]
        ];
        
        // Draw the polygon
        chart.append("polygon")
          .attr("points", polygonPoints.map(p => p.join(",")).join(" "))
          .attr("fill", waveColor)
          .attr("fill-opacity", alpha)
          .attr("stroke", waveColor)
          .attr("stroke-width", 0.5)
          .attr("stroke-opacity", 0.8);
      });
    });
  }
  
  // Draw through waves function (matches your Python implementation)
  function drawThroughWaves(chart, junctionsWithDuration, xScale, yScale) {
    const waveColor = "#541FE4";
    const alpha = 0.2;
    
    // For each through wave
    throughWaves.forEach(wave => {
      // Create start points (for each junction in the wave)
      const starts = [];
      const ends = [];
      
      // Process each interval in the through wave
      wave.intervals.forEach((interval, junctionIdx) => {
        if (junctionIdx < junctionsWithDuration.length) {
          const junction = junctionsWithDuration[junctionIdx];
          const y = yScale(junction.point.y);
          
          starts.push([xScale(interval.start), y]);
          ends.push([xScale(interval.end), y]);
        }
      });
      
      // Reverse ends array (matching Python logic)
      ends.reverse();
      
      // Combine start and end points to create polygon
      const polygonPoints = [...starts, ...ends];
      
      // Draw the polygon
      chart.append("polygon")
        .attr("points", polygonPoints.map(p => p.join(",")).join(" "))
        .attr("fill", waveColor)
        .attr("fill-opacity", alpha)
        .attr("stroke", waveColor)
        .attr("stroke-width", 0.5)
        .attr("stroke-opacity", 0.8);
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
  $: greenWaves, throughWaves, showWaves, updateChart();
  
  function updateSize() {
    if (container) {
      width = container.clientWidth;
      height = container.clientHeight;
      updateChart();
    }
  }
  
  onMount(() => {
    updateChart();
    window.addEventListener('resize', updateSize);
    return () => window.removeEventListener('resize', updateSize);
  });
</script>

<div bind:this={container} class="diagram-container w-full h-full">
  <svg bind:this={svg} {width} {height} class="w-full h-full"></svg>
</div>

<style>
  .diagram-container {
    width: 100%;
    height: 100%;
    min-height: 300px;
  }
</style>