(() => {
  const minimum = 120;

  function clampHeight(height, available) {
    const min = Math.min(minimum, available / 2);
    return Math.max(min, Math.min(available - min, height));
  }

  function resize(handle, height) {
    const workspace = handle.parentElement;
    const request = workspace.querySelector('form.card');
    const response = workspace.querySelector('.response');
    const available = Math.max(0, workspace.clientHeight - [...workspace.children]
      .filter(child => child !== request && child !== response)
      .reduce((total, child) => {
        if (!child.getClientRects().length) return total;
        const style = getComputedStyle(child);
        return total + child.getBoundingClientRect().height + parseFloat(style.marginTop) + parseFloat(style.marginBottom);
      }, 0) -
      parseFloat(getComputedStyle(workspace).paddingTop) - parseFloat(getComputedStyle(workspace).paddingBottom));
    if (!available) return;
    const value = clampHeight(height, available);
    workspace.style.setProperty('--request-height', `${value}px`);
    handle.setAttribute('aria-valuemin', String(Math.round(Math.min(minimum, available / 2) / available * 100)));
    handle.setAttribute('aria-valuemax', String(Math.round((available - Math.min(minimum, available / 2)) / available * 100)));
    handle.setAttribute('aria-valuenow', String(Math.round(value / available * 100)));
  }

  if (typeof module !== 'undefined') module.exports = { clampHeight };
  if (typeof document === 'undefined') return;

  document.addEventListener('keydown', event => {
    const handle = event.target.closest('.split-handle');
    if (!handle || !['ArrowUp', 'ArrowDown'].includes(event.key)) return;
    event.preventDefault();
    const height = handle.parentElement.querySelector('form.card').getBoundingClientRect().height;
    resize(handle, height + (event.key === 'ArrowUp' ? -10 : 10));
  });

  document.addEventListener('pointerdown', event => {
    const handle = event.target.closest('.split-handle');
    if (!handle || event.button !== 0) return;
    event.preventDefault();
    const start = event.clientY;
    const height = handle.parentElement.querySelector('form.card').getBoundingClientRect().height;
    handle.setPointerCapture(event.pointerId);
    const move = e => resize(handle, height + e.clientY - start);
    const end = () => {
      handle.removeEventListener('pointermove', move);
      handle.removeEventListener('pointerup', end);
      handle.removeEventListener('pointercancel', end);
    };
    handle.addEventListener('pointermove', move);
    handle.addEventListener('pointerup', end);
    handle.addEventListener('pointercancel', end);
  });

  window.addEventListener('resize', () => {
    document.querySelectorAll('.workspace').forEach(workspace => {
      const handle = workspace.querySelector('.split-handle');
      const request = workspace.querySelector('form.card');
      if (handle && request && handle.getClientRects().length) resize(handle, request.getBoundingClientRect().height);
    });
  });
})();
