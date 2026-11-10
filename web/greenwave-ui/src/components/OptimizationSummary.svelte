<script>
	export let report;
	export let outdated = false;
	export let matchesInput = false;
	const number = (value) => Number(value.toFixed(1));
</script>

<section class="optimization-summary" aria-label="Optimization comparison" class:outdated>
	<div class="run-status" role="status">
		<strong>{outdated ? 'Previous optimization' : 'Optimization finished'}:</strong>
		{report.changed} of {report.offsets.length} offsets changed.
		{#if report.outcome === 'unchanged'}No better plan found in this run.{:else if report.outcome === 'worse'}The
			returned plan scores lower than the input.{/if}
		{#if !outdated && matchesInput}Input already uses the result.{/if}
	</div>
	<div class="bands">
		<span>Full-corridor band, before → after:</span>
		{#each report.directions as row (row.label)}<strong
				>{row.label} {number(row.before.fullBand)} → {number(row.after.fullBand)} s</strong
			>{/each}
	</div>
	<details>
		<summary>Run details and offsets</summary>
		<!-- Keyboard users need to scroll the result tables. -->
		<!-- svelte-ignore a11y_no_noninteractive_tabindex -->
		<div class="run-details" tabindex="0" role="region" aria-label="Optimization run details">
			<p>
				Compared with the input at the start of this run. Only offsets are optimized; phase
				durations and groups stay unchanged.
			</p>
			<table>
				<caption>Longest connected wave, before → after</caption>
				<thead
					><tr
						><th scope="col">Direction</th><th scope="col">Junctions reached</th><th scope="col"
							>Band at that depth</th
						></tr
					></thead
				>
				<tbody
					>{#each report.directions as row (row.label)}<tr
							><th scope="row">{row.label}</th><td
								>{row.before.depth} → {row.after.depth} of {report.offsets.length}</td
							><td>{number(row.before.longestBand)} → {number(row.after.longestBand)} s</td></tr
						>{/each}</tbody
				>
			</table>
			<p>
				Search score: {number(report.beforeScore)} → {number(report.afterScore)}. The score rewards
				wider and deeper waves, including partial waves. A higher score does not guarantee a
				full-corridor wave.
			</p>
			<table>
				<caption>Junction offsets (seconds)</caption>
				<thead
					><tr
						><th scope="col">Junction</th><th scope="col">Before</th><th scope="col">After</th><th
							scope="col">Change</th
						></tr
					></thead
				>
				<tbody
					>{#each report.offsets as row (row.id)}<tr class:changed={row.before !== row.after}
							><th scope="row">{row.label}</th><td>{row.before}</td><td>{row.after}</td><td
								>{row.after > row.before ? '+' : ''}{row.after - row.before}</td
							></tr
						>{/each}</tbody
				>
			</table>
		</div>
	</details>
</section>

<style>
	.optimization-summary {
		flex-shrink: 0;
		margin-bottom: 10px;
		padding: 9px 12px;
		border: 1px solid #b6dcd6;
		border-radius: 10px;
		background: #f5fbfa;
		font-size: 12px;
		color: #284c47;
	}
	.outdated {
		background: #fffbeb;
		border-color: #f3d28a;
		color: #854d0e;
	}
	.run-status {
		line-height: 1.5;
	}
	.bands {
		display: flex;
		flex-wrap: wrap;
		gap: 4px 14px;
		margin-top: 3px;
		font-variant-numeric: tabular-nums;
	}
	summary {
		cursor: pointer;
		width: fit-content;
		margin-top: 5px;
	}
	.run-details {
		max-height: 220px;
		overflow: auto;
		margin-top: 8px;
	}
	p {
		margin: 8px 0;
	}
	table {
		width: 100%;
		text-align: left;
		font-variant-numeric: tabular-nums;
	}
	caption {
		text-align: left;
		font-weight: 600;
		padding: 6px 0;
	}
	td,
	th {
		padding: 5px 8px;
		border-bottom: 1px solid #dbe8e5;
	}
	tbody th {
		font-weight: 400;
	}
	.changed {
		background: #e7f4f1;
	}
</style>
