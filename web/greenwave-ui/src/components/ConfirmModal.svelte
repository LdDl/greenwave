<script>
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
  <!-- Modal Backdrop - Using inline styles to bypass Tailwind issues -->
  <div 
    class="fixed inset-0 flex items-center justify-center z-50"
    style="background-color: rgba(0, 0, 0, 0.75);"
    on:click={handleBackdropClick}
    role="dialog"
    aria-modal="true"
  >
    <!-- Modal Content -->
    <div class="bg-white rounded-xl shadow-2xl max-w-md w-full mx-4 p-6 border border-gray-300">
      <h3 class="text-lg font-semibold mb-4 text-gray-900">{title}</h3>
      <p class="text-gray-600 mb-6 leading-relaxed">{message}</p>
      
      <div class="flex justify-end gap-3">
        <button 
          on:click={handleCancel}
          class="px-4 py-2 border border-gray-300 rounded-lg hover:bg-gray-50 text-gray-700 transition-colors"
        >
          {cancelText}
        </button>
        <button 
          on:click={handleConfirm}
          class="px-4 py-2 rounded-lg text-white transition-colors"
          class:bg-red-500={danger}
          class:hover:bg-red-600={danger}
          class:bg-blue-500={!danger}
          class:hover:bg-blue-600={!danger}
        >
          {confirmText}
        </button>
      </div>
    </div>
  </div>
{/if}