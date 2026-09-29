<script>
	import { page } from '$app/stores';
	import { tick } from 'svelte';
	import {
		corridorEditor,
		corridorProject,
		corridorHistory,
		corridorPersistence,
		corridorIssue
	} from '$lib/stores/corridor.js';
	import {
		corridorsOpen,
		projectDialogOpen,
		workspaceDialogOpen,
		projectOpened
	} from '$lib/stores/workspace.js';
	import { isLoading, clearCalculatedData } from '$lib/stores';
	import { emptySharedProject, fromInput, parseSharedProject } from '$lib/utils/shared-project.js';
	import { DEMO_DATA } from '$lib/utils/demo-input.js';
	import { rectangularDemoProject } from '$lib/utils/demo-network.js';
	import { exportToJSON } from '$lib/utils/export-import.js';
	import { modalFocus } from '$lib/utils/modal-focus.js';

	let menu = '';
	let toolbar;
	let fileInput;
	let replacement = null;
	let issue = '';
	const legacyNetworkAvailable = corridorEditor.legacyNetworkAvailable;
	$: projectDialogOpen.set(replacement !== null);

	function closeMenu() {
		const trigger = document.getElementById(`workspace-${menu}`);
		menu = '';
		trigger?.focus();
	}
	function toggle(name) {
		menu = menu === name ? '' : name;
	}
	function saveProject() {
		exportToJSON(corridorEditor.snapshot(), 'greenwave-project.json');
		menu = '';
	}
	async function applyProject(project) {
		try {
			corridorEditor.replace(project);
			clearCalculatedData();
			replacement = null;
			issue = '';
			await tick();
			projectOpened.update((value) => value + 1);
		} catch (error) {
			issue = error.message;
		}
	}
	function openProject(project) {
		closeMenu();
		if ($corridorProject.junctions.length) replacement = project;
		else applyProject(project);
	}
	function openLegacy() {
		try {
			openProject(corridorEditor.readLegacyNetwork());
		} catch (error) {
			issue = error.message;
			menu = '';
		}
	}
	async function importProject(event) {
		const file = event.target.files?.[0];
		event.target.value = '';
		if (!file) return;
		try {
			if (file.size > 5 * 1024 * 1024) throw new Error('Choose a JSON file smaller than 5 MB.');
			openProject(parseSharedProject(await file.text()));
		} catch (error) {
			issue = error.message;
		}
	}
	function keys(event) {
		if (event.defaultPrevented) return;
		if (event.key === 'Escape' && menu) {
			closeMenu();
			event.preventDefault();
		}
		if (
			$workspaceDialogOpen ||
			document.querySelector('[role="dialog"]') ||
			event.target.closest('input, textarea, select, [contenteditable="true"]')
		)
			return;
		if ((event.ctrlKey || event.metaKey) && ['z', 'y'].includes(event.key.toLowerCase())) {
			event.preventDefault();
			if (event.shiftKey || event.key.toLowerCase() === 'y') corridorEditor.redo();
			else corridorEditor.undo();
		}
	}
</script>

<svelte:window
	on:click={(event) => {
		if (!toolbar?.contains(event.target)) menu = '';
	}}
	on:keydown={keys}
/>
<header class="workspace-toolbar" bind:this={toolbar}>
	<a href="/" class="workspace-brand" aria-label="Greenwave home"
		><span aria-hidden="true">▥</span> Greenwave</a
	>
	<nav aria-label="Workspace views" class="workspace-nav">
		<a href="/network" aria-current={$page.url.pathname.startsWith('/network') ? 'page' : undefined}
			>Network</a
		>
		<a href="/" aria-current={$page.url.pathname === '/' ? 'page' : undefined}>Coordination</a>
	</nav>
	<div class="workspace-project-title" title={$corridorProject.name}>{$corridorProject.name}</div>
	<span
		class="workspace-save"
		class:save-error={$corridorPersistence.state === 'error'}
		role="status"
		title={$corridorPersistence.message}
	>
		{$corridorPersistence.state === 'saved'
			? 'Saved locally'
			: $corridorPersistence.state === 'error'
				? 'Save failed'
				: 'Not saved yet'}
	</span>
	<div class="workspace-actions">
		<div class="workspace-menu-anchor">
			<button
				class="ui-button"
				id="workspace-project"
				aria-expanded={menu === 'project'}
				on:click={() => toggle('project')}>Project <span aria-hidden="true">⌄</span></button
			>
			{#if menu === 'project'}
				<div class="workspace-menu" aria-label="Project actions">
					<button
						on:click={() => {
							menu = '';
							fileInput.click();
						}}
						disabled={$isLoading}
						><strong>Open JSON</strong><small>Open a saved project or an older input file</small
						></button
					>
					<button on:click={saveProject}
						><strong>Save project as JSON</strong><small
							>All roads, programs, corridors and layout</small
						></button
					>
					<button on:click={() => openProject(emptySharedProject())} disabled={$isLoading}
						><strong>New project</strong><small>Start with an empty network</small></button
					>
					<details>
						<summary>Compatibility and recovery</summary>
						<p>
							These actions support older files and API integrations. Use "Save project as JSON" for
							a complete backup.
						</p>
						<button
							disabled={!!$corridorIssue}
							on:click={() => {
								exportToJSON(corridorEditor.inputSnapshot(), 'greenwave-input.json');
								menu = '';
							}}>Export corridor for API</button
						>
						{#if $legacyNetworkAvailable}<button on:click={openLegacy} disabled={$isLoading}
								>Recover previous browser network</button
							>
							<p>The previous network stays in browser storage until you choose to open it.</p>{/if}
					</details>
				</div>
			{/if}
		</div>
		<div class="workspace-menu-anchor">
			<button
				class="ui-button"
				id="workspace-examples"
				aria-expanded={menu === 'examples'}
				on:click={() => toggle('examples')}>Examples <span aria-hidden="true">⌄</span></button
			>
			{#if menu === 'examples'}
				<div class="workspace-menu" aria-label="Example projects">
					<button disabled={$isLoading} on:click={() => openProject(fromInput(DEMO_DATA))}
						><strong>Load demo data</strong><small
							>4 junctions, one simple corridor. Start here.</small
						></button
					>
					<button disabled={$isLoading} on:click={() => openProject(rectangularDemoProject())}
						><strong>Writers Quarter</strong><small
							>22 junctions, city blocks and 5 alternative corridors</small
						></button
					>
				</div>
			{/if}
		</div>
		<div class="workspace-history" role="group" aria-label="Project history">
			<button
				class="ui-button"
				disabled={!$corridorHistory.undo || $workspaceDialogOpen}
				on:click={() => corridorEditor.undo()}
				aria-label="Undo"
				title="Undo (Ctrl/Cmd+Z)">Undo</button
			>
			<button
				class="ui-button"
				disabled={!$corridorHistory.redo || $workspaceDialogOpen}
				on:click={() => corridorEditor.redo()}
				aria-label="Redo"
				title="Redo (Ctrl/Cmd+Shift+Z)">Redo</button
			>
		</div>
		<button
			class="ui-button ui-accent"
			on:click={() => {
				menu = '';
				corridorsOpen.set(true);
			}}
			aria-haspopup="dialog"
			>Corridors <span class="ui-count">{$corridorProject.corridors.length}</span></button
		>
	</div>
	<input
		bind:this={fileInput}
		type="file"
		accept=".json,application/json"
		class="hidden"
		aria-label="Open project JSON"
		on:change={importProject}
	/>
</header>
{#if $corridorPersistence.state === 'error'}<div class="workspace-alert" role="alert">
		{$corridorPersistence.message} <button on:click={saveProject}>Save a JSON copy</button>
	</div>{/if}
{#if issue}<div class="workspace-alert" role="alert">
		{issue}<button aria-label="Dismiss project error" on:click={() => (issue = '')}>×</button>
	</div>{/if}
{#if replacement}
	<div
		class="workspace-overlay"
		role="dialog"
		aria-modal="true"
		aria-labelledby="project-replace-title"
		tabindex="-1"
		use:modalFocus
		on:keydown={(event) => {
			if (event.key === 'Escape') {
				event.stopPropagation();
				replacement = null;
			}
		}}
	>
		<div class="workspace-dialog">
			<p class="ui-eyebrow">Open project</p>
			<h2 id="project-replace-title">{replacement.name}</h2>
			<p>
				{replacement.junctions.length} junctions · {replacement.roads.length} roads · {replacement
					.corridors.length} corridors
			</p>
			<p>
				This replaces the current network and all corridors. Calculated results will be cleared.
				Undo restores the previous project during this session.
			</p>
			{#if replacement.migration}<p class="ui-warning">
					{replacement.migration.notes.join(' ')} The original browser copy is retained.
				</p>{/if}
			<div class="ui-actions">
				<button class="ui-button" on:click={saveProject}>Save current project</button><button
					class="ui-button"
					on:click={() => (replacement = null)}>Cancel</button
				><button class="ui-button ui-primary" on:click={() => applyProject(replacement)}
					>Open project</button
				>
			</div>
		</div>
	</div>
{/if}
