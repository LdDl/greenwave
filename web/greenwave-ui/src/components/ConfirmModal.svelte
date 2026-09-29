<script>
  import { fade, scale } from 'svelte/transition';

  export let show = false;
  export let title = "Confirm Action";
  export let message = "Are you sure you want to proceed?";
  export let confirmText = "Confirm";
  export let cancelText = "Cancel";
  export let onConfirm = () => {};
  export let onCancel = () => {};
  export let danger = false;

  function handleConfirm() {
    onConfirm();
    show = false;
  }

  function handleCancel() {
    onCancel();
    show = false;
  }

  function handleBackdropClick(event) {
    if (event.target === event.currentTarget) {
      handleCancel();
    }
  }
</script>

{#if show}
  <!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
  <div
    transition:fade={{ duration: 150 }}
    class="fixed inset-0 flex items-center justify-center z-50"
    style="background-color: rgba(0, 0, 0, 0.6);"
    on:click={handleBackdropClick}
    on:keydown={(e) => e.key === 'Escape' && handleCancel()}
    role="dialog"
    aria-modal="true"
    tabindex="-1"
  >
    <div
      transition:scale={{ start: 0.96, duration: 150 }}
      class="bg-white rounded-lg shadow-2xl max-w-md w-full mx-4 p-6 border border-gray-200"
    >
      <h3 class="text-lg font-semibold mb-3 text-gray-900">{title}</h3>
      <p class="text-gray-600 mb-6 leading-relaxed text-sm">{message}</p>

      <div class="flex justify-end gap-3">
        <button
          on:click={handleCancel}
          class="px-4 py-2 border border-gray-300 rounded-md hover:bg-gray-50 text-gray-700 text-sm transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-gray-400"
        >
          {cancelText}
        </button>
        <button
          on:click={handleConfirm}
          class="px-4 py-2 rounded-md text-white text-sm transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-offset-1"
          class:bg-red-500={danger}
          class:hover:bg-red-600={danger}
          class:focus-visible:ring-red-400={danger}
          class:bg-blue-500={!danger}
          class:hover:bg-blue-600={!danger}
          class:focus-visible:ring-blue-400={!danger}
        >
          {confirmText}
        </button>
      </div>
    </div>
  </div>
{/if}
