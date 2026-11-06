// Keep the editor DOM (including selected files) intact when aborting a send.
(() => {
  let active;
  const cancelled = new WeakSet();
  const status = text => { document.getElementById('request-status').textContent = text; };
  document.addEventListener('htmx:beforeRequest', event => {
    const { elt, xhr, requestConfig } = event.detail;
    if (!elt.matches('form[action="/request"]')) return;
    if (requestConfig.parameters.action !== 'send') return;
    active = { elt, xhr };
    document.getElementById('cancel-request').disabled = false;
    status('Sending…');
  });
  document.addEventListener('click', event => {
    if (!event.target.closest('#cancel-request') || !active) return;
    const { elt, xhr } = active;
    cancelled.add(xhr);
    status('Cancelled. Remote side effects may already have happened. Unsaved changes and selected files retained.');
    // HTMX aborts its XHR and performs its own loading/disabled-control cleanup.
    htmx.trigger(elt, 'htmx:abort');
  });
  document.addEventListener('htmx:beforeSwap', event => {
    if (cancelled.has(event.detail.xhr)) event.detail.shouldSwap = false;
  });
  document.addEventListener('htmx:afterRequest', event => {
    if (!active || event.detail.xhr !== active.xhr) return;
    if (!cancelled.has(active.xhr)) status(event.detail.successful ? 'Request complete.' : 'Request failed. Check connection or submitted values.');
    document.getElementById('cancel-request').disabled = true;
    active = undefined;
  });
})();
