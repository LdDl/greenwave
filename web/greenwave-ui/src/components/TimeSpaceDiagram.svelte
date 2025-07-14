<!-- components/TimeSpaceDiagram.svelte -->
<script>
  import { onMount } from 'svelte';
  import * as d3 from 'd3';
  import { junctions } from '$lib/stores';
  
  let svg;
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
  
  function getSignalColor(color) {
    const colorMap = {
      'RED': '#dc2626',
      'YELLOW': '#fbbf24', 
      'GREEN': '#16a34a',
      'GREENPRIORITY': '#15803d'
    };
    return colorMap[color] || '#000000';
  }
  
  // Update chart when junctions change
  $: if (svg) {
    updateChart();
  }
  
  onMount(() => {
    updateChart();
  });
</script>

<div class="diagram-container">
  <svg bind:this={svg} {width} {height} class="border border-gray-300 rounded"></svg>
</div>

<style>
  .diagram-container {
    width: 100%;
    height: 100%;
    overflow: hidden;
  }
</style>