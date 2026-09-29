<script>
  import { onMount } from 'svelte';
  import { createEventDispatcher } from 'svelte';

  import * as d3 from 'd3';
  import { diagramLayout } from '$lib/utils/diagram-layout.js';
  import { corridorGroupId, corridorTimelineRows } from '$lib/utils/junction-program.js';

  export let junctions = [];
  export let groupIds = {};
  export let reverseGroupIds = {};
  export let direction = 'forward';
  export let wavesAreOutdated = { isOutdated: false, reason: null };
  export let interactive = false;
  export let greenWaves = [];
  export let throughWaves = [];
  export let reverseGreenWaves = [];
  export let reverseThroughWaves = [];
  export let showWaves = false;
  export let showOffsets = false;
  export let overview = false;
  export let zoom = 1;
  export let timeZoom = 1;
  
  const dispatch = createEventDispatcher();

  let svg;
  let ruler;
  let container;
  let width = 700;
  let height = 400;
  let isDragging = false; 

  const margin = { top: 62, right: 70, bottom: 46, left: 65 };
  let distanceMinimum = 0;
  let distanceMaximum = 100;
  let coincident = false;
  let chartWidth = width - margin.left - margin.right;
  let chartHeight = height - margin.top - margin.bottom;
  
  // Helper function to calculate total duration for a junction.
  // Uses the first signal group as the reference (all groups are synchronized).
  function calculateTotalDuration(junction) {
    return junction.cycle.reduce((total, phase) => {
      const signals = phase.signal_groups[0].signals;
      return total + signals.reduce((phaseTotal, signal) => {
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
      total_duration: calculateTotalDuration(junction),
      group_id: corridorGroupId(junction, groupIds),
      timelineRows: corridorTimelineRows(junction, groupIds, reverseGroupIds, direction).map(row => ({ ...row, offset: row.offset * 1.8 }))
    }));
    
    // Calculate max time domain
    const maxTime = Math.max(...junctionsWithDuration.map(j => j.total_duration));
    
    // Scales
    const xScale = d3.scaleLinear()
      .domain([0, maxTime])
      .range([0, chartWidth]);
    
    const yScale = d3.scaleLinear()
      .domain([distanceMinimum, distanceMaximum])
      .range([chartHeight, 0]);
    
    const rulerChart = d3.select(ruler);
    rulerChart.selectAll('*').remove();
    rulerChart.append('g').attr('transform', `translate(${margin.left},0)`)
      .call(d3.axisBottom(xScale).ticks(Math.max(4, chartWidth / 90)));
    rulerChart.append('text').attr('x', 8).attr('y', 16).attr('fill', '#64748b').attr('font-size', 10).text('Time (s)');
    chart.append('g').attr('class', 'time-grid').selectAll('line').data(xScale.ticks()).join('line')
      .attr('x1', xScale).attr('x2', xScale).attr('y1', 0).attr('y2', chartHeight)
      .attr('stroke', '#e9eef3').attr('stroke-dasharray', '3,4');

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
      .call(d3.axisLeft(yScale).ticks(Math.min(40, Math.max(4, chartHeight / 80))))
      .append("text")
      .attr("transform", "rotate(-90)")
      .attr("y", -45)
      .attr("x", -chartHeight / 2)
      .attr("fill", "black")
      .style("text-anchor", "middle")
      .text("Distance (meters)");
    
    // Draw waves if enabled
    if (showWaves) {
      // Forward direction waves
      if (throughWaves.length > 0) {
        drawThroughWaves(chart, junctionsWithDuration, xScale, yScale, wavesAreOutdated.isOutdated);
      }
      if (greenWaves.length > 0) {
        drawGreenWaves(chart, junctionsWithDuration, xScale, yScale, wavesAreOutdated.isOutdated);
      }
      // Reverse direction waves (for bidirectional mode)
      if (reverseThroughWaves.length > 0) {
        drawReverseThroughWaves(chart, junctionsWithDuration, xScale, yScale, wavesAreOutdated.isOutdated);
      }
      if (reverseGreenWaves.length > 0) {
        drawReverseGreenWaves(chart, junctionsWithDuration, xScale, yScale, wavesAreOutdated.isOutdated);
      }
    }
    
    // Draw phases
    drawPhases(chart, junctionsWithDuration, xScale, yScale);

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
          isDragging = true;
          dispatch('dragStart');
        })
        .on("drag", function(event, d) {
          // Use the simple working approach with event.y
          let newDistance = Math.round(yScale.invert(Math.max(0, Math.min(chartHeight, event.y))));
          const index = junctions.findIndex(junction => junction.id === d.id);
          const ordered = junctions.every((junction, i) => i === 0 || junction.point.y >= junctions[i - 1].point.y);
          if (ordered) {
            const lower = index > 0 ? junctions[index - 1].point.y : 0;
            const upper = index < junctions.length - 1 ? junctions[index + 1].point.y : Infinity;
            newDistance = Math.min(upper, Math.max(lower, newDistance));
          }
          const newY = yScale(newDistance);
          
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
          isDragging = false;
          dispatch('dragEnd');
          // Force complete redraw after drag ends
          setTimeout(() => updateChart(), 10);
        })
      );
    }
    
    // Draw junction labels with duration
    const junctionLabels = junctionGroups.append("text")
      .attr("x", 0)
      .attr("y", d => d.timelineRows.length > 1 ? -47 : -32)
      .attr("text-anchor", "start")
      .attr("font-size", "12px")
      .attr("font-weight", "600")
      .attr("fill", "#333")
      .text(d => `${d.label || `J${d.id}`}, ${d.total_duration}s${d.cycle[0].signal_groups.length > 1 ? (d.group_id === null ? ', choose group' : `, G${d.group_id}`) : ''}`);

    // Draw offset labels at the right end of X axis (results panel only)
    if (showOffsets) {
      junctionGroups.append("text")
        .attr("x", chartWidth + 4)
        .attr("y", 0)
        .attr("dy", "0.35em")
        .attr("text-anchor", "start")
        .attr("font-size", "10px")
        .attr("font-weight", "bold")
        .attr("fill", "#4B0082")
        .text(d => `+${d.offset}s`);
    }

    // Make labels clickable in interactive mode
    if (interactive) {
      junctionLabels
        .style("cursor", "pointer")
        .on("mouseover", function() {
          d3.select(this).attr("fill", "#4B0082");
        })
        .on("mouseout", function() {
          d3.select(this).attr("fill", "#333");
        })
        .on("click", (event, d) => {
          event.stopPropagation();
          dispatch('editJunction', { junction: d });
        });
    }

    // Draw junction circles
    const junctionCircles = junctionGroups.append("circle")
      .attr("cx", 0)
      .attr("cy", 0)
      .attr("r", 6)
      .attr("fill", "#D8BFD8")
      .attr("stroke", "#4B0082")
      .attr("stroke-width", 2);

    // Make circles clickable in interactive mode
    if (interactive) {
      junctionCircles
        .style("cursor", "pointer")
        .on("mouseover", function() {
          d3.select(this).attr("r", 8).attr("fill", "#C8A2C8");
        })
        .on("mouseout", function() {
          d3.select(this).attr("r", 6).attr("fill", "#D8BFD8");
        })
        .on("click", (event, d) => {
          event.stopPropagation();
          dispatch('editJunction', { junction: d });
        });
    }
  }

  // Helper function to handle modulo with negative numbers correctly
  function positiveModulo(value, modulus) {
    return ((value % modulus) + modulus) % modulus;
  }

  function updateSignalLinesForJunction(junctionId, newY, junctionsWithDuration, xScale, yScale, chart) {
    const junction = junctionsWithDuration.find(j => j.id === junctionId);
    if (!junction) return;
    chart.selectAll(`.signal-row-${junctionId}`).remove();
    drawJunctionTimelines(chart, junction, xScale, newY);
  }

  function drawSignalTimelines(chart, junctionsWithDuration, xScale, yScale) {
    junctionsWithDuration.forEach(junction => drawJunctionTimelines(chart, junction, xScale, yScale(junction.point.y)));
  }

  function drawJunctionTimelines(chart, junction, xScale, y) {
    for (const row of junction.timelineRows) {
      const container = chart.append('g').attr('class', `signal-row-${junction.id}`).attr('data-group-id', row.groupId);
      let currentTime = junction.offset ?? 0;
      for (const phase of junction.cycle) {
        for (const signal of phase.signal_groups.find(group => group.id === row.groupId)?.signals ?? []) {
          if (signal.duration > 0) {
            const start = positiveModulo(currentTime, junction.total_duration);
            const end = positiveModulo(currentTime + signal.duration, junction.total_duration);
            const parts = end < start || signal.duration >= junction.total_duration ? [[start, junction.total_duration], [0, end]] : [[start, end]];
            for (const [from, to] of parts) {
              if (from === to) continue;
              const line = container.append('line')
                .attr('class', `signal-line-${junction.id}`)
                .attr('x1', xScale(from)).attr('x2', xScale(to))
                .attr('y1', y + row.offset).attr('y2', y + row.offset)
                .attr('stroke', getSignalColor(signal.color)).attr('stroke-width', 6);
              line.append('title').text(`${row.label}: ${signal.color}, ${signal.duration} s`);
              if (interactive) {
                line.style('cursor', 'pointer')
                  .on('mouseover', function() { d3.select(this).attr('stroke-width', 10); })
                  .on('mouseout', function() { d3.select(this).attr('stroke-width', 6); })
                  .on('click', () => handleSignalClick(junction, phase, signal));
              }
            }
          }
          currentTime += signal.duration;
        }
      }
      if (direction === 'bidirectional') {
        container.append('text').attr('x', chartWidth).attr('y', y + row.offset - 3)
          .attr('text-anchor', 'end').attr('font-size', 10).attr('fill', '#334155')
          .attr('stroke', 'white').attr('stroke-width', 3).attr('paint-order', 'stroke')
          .style('pointer-events', 'none').text(row.groupId === null ? 'Choose group' : row.label);
      }
    }
  }

  function drawPhases(chart, junctionsWithDuration, xScale, yScale) {
    junctionsWithDuration.forEach((junction) => {
      let currentTime = junction.offset; // Start at the junction's offset
      const y = yScale(junction.point.y) - (junction.timelineRows.length > 1 ? 20 : 0);

      junction.cycle.forEach((phase, phaseIdx) => {
        // Calculate phase duration from the first signal group (all groups are synchronized)
        const phaseDuration = phase.signal_groups[0].signals.reduce((sum, signal) => sum + signal.duration, 0);

        // Calculate phase start and end times
        const phaseStart = positiveModulo(currentTime, junction.total_duration);
        const phaseEnd = positiveModulo(currentTime + phaseDuration, junction.total_duration);

        // Define alternating colors for phases
        const phaseColor = phaseIdx % 2 === 0 ? "#4B0082" : "#18B7CC";

        // Handle wrapping (phase goes from end to start)
        if (phaseEnd < phaseStart || phaseDuration >= junction.total_duration) {
          // Draw first part (from phaseStart to the end of the timeline)
          chart.append("rect")
            .attr("x", xScale(phaseStart))
            .attr("width", xScale(junction.total_duration) - xScale(phaseStart))
            .attr("y", y - 10)
            .attr("height", 5)
            .attr("fill", phaseColor)
            .attr("fill-opacity", 0.5)
            .attr("stroke", "black") // Add black stroke
            .attr("stroke-width", 1);

          // Draw second part (from 0 to phaseEnd)
          chart.append("rect")
            .attr("x", xScale(0))
            .attr("width", xScale(phaseEnd) - xScale(0))
            .attr("y", y - 10)
            .attr("height", 5)
            .attr("fill", phaseColor)
            .attr("fill-opacity", 0.5)
            .attr("stroke", "black") // Add black stroke
            .attr("stroke-width", 1);
        } else {
          // Draw phase interval (no wrapping)
          chart.append("rect")
            .attr("x", xScale(phaseStart))
            .attr("width", xScale(phaseEnd) - xScale(phaseStart))
            .attr("y", y - 10)
            .attr("height", 5)
            .attr("fill", phaseColor)
            .attr("fill-opacity", 0.5)
            .attr("stroke", "black") // Add black stroke
            .attr("stroke-width", 1);
        }

        // Draw phase label (centered)
        const labelX = phaseEnd < phaseStart || phaseDuration >= junction.total_duration
          ? xScale(positiveModulo((phaseStart + junction.total_duration + phaseEnd) / 2, junction.total_duration))
          : xScale((phaseStart + phaseEnd) / 2);

        chart.append("text")
          .attr("x", labelX)
          .attr("y", y - 15)
          .attr("text-anchor", "middle")
          .attr("font-size", "10px")
          .attr("fill", "#4B0082")
          .text(`Phase ${phaseIdx + 1}`);

        // Update currentTime for the next phase
        currentTime += phaseDuration;
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
      const y1 = yScale(j1.point.y) + j1.timelineRows.at(0).offset;
      const y2 = yScale(j2.point.y) + j2.timelineRows.at(0).offset;
      
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
          const y = yScale(junction.point.y) + junction.timelineRows.at(0).offset;

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

  // Draw reverse green waves (bidirectional mode) - light blue color
  function drawReverseGreenWaves(chart, junctionsWithDuration, xScale, yScale, isOutdated = false) {
    const waveColor = "#4FC3F7"; // Light blue
    const alpha = isOutdated ? 0.15 : 0.3;
    const numJunctions = junctionsWithDuration.length;

    reverseGreenWaves.forEach((segmentWaves, segmentIdx) => {
      if (segmentIdx >= numJunctions - 1) return;

      // Reverse direction: map segment index to actual junctions (from last to first)
      const j1 = junctionsWithDuration[numJunctions - 1 - segmentIdx];
      const j2 = junctionsWithDuration[numJunctions - 2 - segmentIdx];
      const y1 = yScale(j1.point.y) + j1.timelineRows.at(-1).offset;
      const y2 = yScale(j2.point.y) + j2.timelineRows.at(-1).offset;

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
          .attr("stroke", isOutdated ? "#ff6b35" : waveColor)
          .attr("stroke-width", isOutdated ? 1 : 0.5)
          .attr("stroke-opacity", isOutdated ? 0.6 : 0.8)
          .attr("stroke-dasharray", isOutdated ? "3,3" : "none");
      });
    });
  }

  // Draw reverse through waves (bidirectional mode) - stronger blue color
  function drawReverseThroughWaves(chart, junctionsWithDuration, xScale, yScale, isOutdated = false) {
    const waveColor = "#1976D2"; // Stronger blue
    const alpha = isOutdated ? 0.1 : 0.2;
    const numJunctions = junctionsWithDuration.length;

    reverseThroughWaves.forEach(wave => {
      const starts = [];
      const ends = [];

      wave.intervals.forEach((interval, idx) => {
        // Reverse direction: map index to junctions from last to first
        const junctionIdx = numJunctions - 1 - idx;
        if (junctionIdx >= 0 && junctionIdx < numJunctions) {
          const junction = junctionsWithDuration[junctionIdx];
          const y = yScale(junction.point.y) + junction.timelineRows.at(-1).offset;

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
        .attr("stroke", isOutdated ? "#ff6b35" : waveColor)
        .attr("stroke-width", isOutdated ? 1 : 0.5)
        .attr("stroke-opacity", isOutdated ? 0.6 : 0.8)
        .attr("stroke-dasharray", isOutdated ? "3,3" : "none");
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
  $: if (svg && junctions.length > 0) {
    updateChart();
  }
  
  // Also reactive to all prop changes
  $: groupIds, reverseGroupIds, direction, greenWaves, throughWaves, reverseGreenWaves, reverseThroughWaves, showWaves, wavesAreOutdated, updateChart();
  
  function updateSize() {
    if (!container || isDragging) return;
    const layout = diagramLayout(junctions, container.clientWidth, container.clientHeight, { overview, zoom, timeZoom });
    width = layout.width;
    height = layout.height;
    distanceMinimum = layout.minimum;
    distanceMaximum = layout.maximum;
    coincident = layout.coincident;
    chartWidth = width - margin.left - margin.right;
    chartHeight = height - margin.top - margin.bottom;
    updateChart();
  }
  $: junctions, overview, zoom, timeZoom, updateSize();

  export function scrollToViewport({ vertical, horizontal }) {
    if (!container) return;
    const top = vertical * (container.scrollHeight - container.clientHeight);
    const left = horizontal * (container.scrollWidth - container.clientWidth);
    if (Math.abs(container.scrollTop - top) > 1) container.scrollTop = top;
    if (Math.abs(container.scrollLeft - left) > 1) container.scrollLeft = left;
  }

  function handleSignalClick(junction, phase, signal) {
    dispatch('editSignal', { junction, phase, signal });
  }

  onMount(() => {
    updateSize();
    const observer = new ResizeObserver(updateSize);
    observer.observe(container);
    return () => observer.disconnect();
  });
</script>

<!-- Keyboard focus enables native scrolling of the diagram. -->
<!-- svelte-ignore a11y_no_noninteractive_tabindex -->
<div bind:this={container} class="diagram-container" on:scroll={() => dispatch('viewportScroll', { vertical: container.scrollTop / Math.max(1, container.scrollHeight - container.clientHeight), horizontal: container.scrollLeft / Math.max(1, container.scrollWidth - container.clientWidth) })} role="region" aria-label={interactive ? 'Scrollable input time-distance diagram' : 'Scrollable result time-distance diagram'} tabindex="0">
  {#if coincident}<p class="coincident-note">Some junctions have the same distance and their signals overlap. Edit their distances to separate them.</p>{/if}
  <svg bind:this={ruler} {width} height={26} class="diagram-ruler" aria-hidden="true"></svg>
  <svg bind:this={svg} {width} {height} style="display:block" aria-label="Time in seconds and distance in meters"></svg>
</div>

<style>
  .diagram-container { width: 100%; height: 100%; min-height: 0; position: relative; overflow: auto; overscroll-behavior: contain; scrollbar-gutter: stable; }
  .diagram-ruler { position: sticky; top: 0; z-index: 2; display: block; background: #fff; border-bottom: 1px solid #e2e8f0; }
  .coincident-note { position: sticky; top: 0; left: 0; z-index: 1; font-size: 12px; color: #92400e; background: #fffbeb; padding: 8px 12px; }
  @media (max-width: 900px) { .diagram-container { overscroll-behavior: auto; } }
</style>
