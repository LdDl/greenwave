export function modalFocus(element) {
  const previous = document.activeElement;
  const previousOverflow = document.body.style.overflow;
  document.body.style.overflow = 'hidden';
  element.focus();
  function trap(event) {
    if (event.key !== 'Tab') return;
    const items = [...element.querySelectorAll('button, input, select, textarea, a[href], [tabindex="0"]')]
      .filter(item => !item.disabled && item.getClientRects().length);
    if (!items.length) { event.preventDefault(); return; }
    const first = items[0];
    const last = items[items.length - 1];
    if (event.shiftKey && (document.activeElement === first || document.activeElement === element)) {
      event.preventDefault();
      last.focus();
    } else if (!event.shiftKey && (document.activeElement === last || document.activeElement === element)) {
      event.preventDefault();
      first.focus();
    }
  }
  element.addEventListener('keydown', trap);
  return { destroy() {
    element.removeEventListener('keydown', trap);
    document.body.style.overflow = previousOverflow;
    if (previous instanceof HTMLElement || previous instanceof SVGElement) previous.focus();
  } };
}
