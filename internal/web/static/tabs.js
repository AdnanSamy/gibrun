// Only saved IDs/order/selection persist; live drafts, responses and files never do.
(() => {
  const storageKey = 'solarclient.request-tabs.v1';
  const maxTabs = 100;
  let restoring = true;
  function readSession() {
    try {
      const raw = sessionStorage.getItem(storageKey);
      if (!raw || raw.length > 16000) return null;
      const value = JSON.parse(raw);
      if (value.version !== 1 || !Array.isArray(value.ids) || value.ids.length > maxTabs ||
          !value.ids.every(id => typeof id === 'string' && /^[A-Za-z0-9_-]{1,128}$/.test(id)) ||
          !(value.active === null || value.ids.includes(value.active))) return null;
      return { ids: [...new Set(value.ids)], active: value.active };
    } catch (_) { return null; }
  }
  const session = readSession();
  const tabs = [];
  function persist() {
    if (restoring) return;
    const ids = [...new Set(tabs.map(tab => identity(tab.panel)).filter(id => /^[A-Za-z0-9_-]{1,128}$/.test(id)))].slice(0, maxTabs);
    const selected = identity(active.panel);
    try { sessionStorage.setItem(storageKey, JSON.stringify({ version: 1, ids, active: ids.includes(selected) ? selected : null })); } catch (_) { /* Storage may be disabled/full. */ }
  }
  async function restore() {
    if (!session?.ids.length) { restoring = false; persist(); return; }
    busy = true;
    const initial = active, startRevision = revision;
    const restored = [];
    try {
      for (const id of session.ids) {
        const existing = tabs.find(tab => identity(tab.panel) === id);
        if (existing) { restored.push(existing); continue; }
        try {
          const response = await fetch('/?id=' + encodeURIComponent(id), { credentials: 'same-origin', redirect: 'error', cache: 'no-store' });
          if (!response.ok) continue;
          const page = new DOMParser().parseFromString(await response.text(), 'text/html');
          const panel = page.querySelector('#main');
          if (!panel || !form(panel) || identity(panel) !== id) continue; // Deleted IDs render a blank editor.
          const tab = { panel, key: ++serial, dirty: false };
          tabs.push(tab); restored.push(tab); htmx.process(panel);
        } catch (_) { /* Deleted/unavailable requests never replace a live draft. */ }
      }
      const untouched = revision === startRevision && !uploads(initial);
      const discardInitial = untouched && !restored.includes(initial) && restored.length;
      const remaining = tabs.filter(tab => !restored.includes(tab) && !(discardInitial && tab === initial));
      tabs.splice(0, tabs.length, ...restored, ...remaining);
      if (untouched) select(restored.find(tab => identity(tab.panel) === session.active) || (discardInitial ? restored[0] : initial));
      document.dispatchEvent(new CustomEvent('htmx:load'));
    } finally { busy = false; restoring = false; render(); }
  }
  let active, busy = false, serial = 0, revision = 0, submittedRevision = 0;
  const form = panel => panel.querySelector('form[action="/request"]');
  const identity = panel => form(panel).elements.id.value;
  const title = tab => form(tab.panel).elements.name.value || 'Untitled request';
  const uploads = tab => [...tab.panel.querySelectorAll('input[type="file"]')].some(input => input.files.length);
  function blocked() {
    if (!busy) return false;
    alert('A request is in progress. Wait for completion, or Cancel the send before changing tabs.');
    return true;
  }
  const strip = document.createElement('div');
  strip.className = 'workspace-tabs';
  const list = document.createElement('div');
  list.setAttribute('role', 'tablist');
  list.setAttribute('aria-label', 'Open requests');
  const add = document.createElement('button');
  add.type = 'button'; add.textContent = '+';
  add.className = 'new-request-tab';
  add.setAttribute('aria-label', 'New request');
  add.title = 'New request';
  add.onclick = () => open('/');
  list.id = 'request-tab-list';
  const previous = document.createElement('button'), next = document.createElement('button');
  for (const [button, direction, label] of [[previous, -1, 'Scroll tabs left'], [next, 1, 'Scroll tabs right']]) {
    button.type = 'button'; button.className = 'tab-scroll';
    button.textContent = direction < 0 ? '‹' : '›';
    button.setAttribute('aria-label', label); button.setAttribute('aria-controls', list.id);
    button.hidden = true;
    button.onclick = () => { list.scrollLeft += direction * list.clientWidth * 0.75; updateOverflow(); };
  }
  strip.append(previous, list, next, add);
  document.querySelector('#main').before(strip);
  function updateOverflow() {
    const overflowing = list.scrollWidth > list.clientWidth + 1;
    previous.hidden = next.hidden = !overflowing;
    previous.disabled = list.scrollLeft <= 1;
    next.disabled = list.scrollLeft + list.clientWidth >= list.scrollWidth - 1;
  }
  function revealActive() {
    const button = document.getElementById('request-tab-' + active.key);
    if (!button) return;
    const item = button.parentElement.getBoundingClientRect(), viewport = list.getBoundingClientRect();
    // Scroll only the strip: scrollIntoView can also move the document vertically.
    if (item.left < viewport.left) list.scrollLeft += item.left - viewport.left;
    else if (item.right > viewport.right) list.scrollLeft += item.right - viewport.right;
    updateOverflow();
  }
  list.addEventListener('scroll', updateOverflow, { passive: true });
  list.addEventListener('wheel', event => {
    if (event.ctrlKey || (!event.shiftKey && Math.abs(event.deltaX) <= Math.abs(event.deltaY))) return;
    const delta = (event.deltaX || event.deltaY) * (event.deltaMode === 1 ? 16 : event.deltaMode === 2 ? list.clientWidth : 1);
    const before = list.scrollLeft;
    list.scrollLeft += delta;
    if (list.scrollLeft !== before) event.preventDefault();
    updateOverflow();
  }, { passive: false });
  const observer = new ResizeObserver(() => { updateOverflow(); revealActive(); });
  observer.observe(list);
  document.addEventListener('htmx:beforeCleanupElement', function cleanup(event) {
    if (event.detail.elt === strip || event.detail.elt.contains(strip)) {
      observer.disconnect();
      document.removeEventListener('htmx:beforeCleanupElement', cleanup);
    }
  });
  function render() {
    persist();
    list.replaceChildren();
    for (const tab of tabs) {
      const group = document.createElement('div');
      group.setAttribute('role', 'presentation');
      group.className = tab === active ? 'request-tab active' : 'request-tab';
      const button = document.createElement('button');
      button.type = 'button'; button.id = 'request-tab-' + tab.key;
      button.setAttribute('role', 'tab');
      button.setAttribute('aria-selected', String(tab === active));
      button.setAttribute('aria-controls', 'main');
      button.tabIndex = tab === active ? 0 : -1;
      button.textContent = title(tab) + (tab.dirty ? ' •' : '');
      button.title = button.textContent;
      button.onclick = () => { if (!blocked()) select(tab); };
      button.onkeydown = event => {
        const index = tabs.indexOf(tab);
        const next = event.key === 'Home' ? 0 : event.key === 'End' ? tabs.length - 1 : event.key === 'ArrowRight' ? (index + 1) % tabs.length : event.key === 'ArrowLeft' ? (index + tabs.length - 1) % tabs.length : -1;
        if (next >= 0) { event.preventDefault(); if (!blocked()) select(tabs[next], true); }
        if (event.key === 'Delete') { event.preventDefault(); close(tab); }
      };
      const remove = document.createElement('button');
      remove.type = 'button'; remove.textContent = '×';
      remove.className = 'close-request-tab';
      remove.setAttribute('aria-label', 'Close ' + title(tab));
      remove.onclick = () => close(tab);
      group.append(button, remove); list.append(group);
    }
    updateOverflow();
    active.panel.setAttribute('role', 'tabpanel');
    active.panel.setAttribute('aria-labelledby', 'request-tab-' + active.key);
    for (const link of document.querySelectorAll('.request-link')) {
      const selected = link.dataset.id === identity(active.panel);
      link.classList.toggle('active', selected);
      if (selected) link.setAttribute('aria-current', 'page'); else link.removeAttribute('aria-current');
    }
  }
  function select(tab, focus = false) {
    if (active !== tab) { active.panel.replaceWith(tab.panel); active = tab; }
    render();
    const button = document.getElementById('request-tab-' + tab.key);
    revealActive();
    if (focus) button.focus({ preventScroll: true });
  }
  function register(panel) {
    const tab = { panel, key: ++serial, dirty: !identity(panel) };
    tabs.push(tab);
    if (!active) active = tab;
    select(tab);
    htmx.process(panel);
  }
  function refresh(page) {
    const incoming = form(page.querySelector('#main'));
    const ids = new Set([...page.querySelectorAll('.request-link')].map(link => link.dataset.id));
    for (const tab of tabs) {
      const editor = form(tab.panel);
      for (const name of ['collection', 'environment']) {
        const select = editor.elements[name], value = select.value;
        select.replaceChildren(...[...incoming.elements[name].options].map(option => option.cloneNode(true)));
        if (![...select.options].some(option => option.value === value)) {
          const missing = document.createElement('option');
          missing.value = value; missing.disabled = true;
          missing.textContent = 'Removed ' + (name === 'collection' ? 'destination' : 'environment') + ' — choose again';
          select.append(missing);
          select.setCustomValidity('Choose an existing ' + name + ' or explicitly select none.');
        } else select.setCustomValidity('');
        select.value = value;
      }
      if (editor.elements.id.value && !ids.has(editor.elements.id.value)) {
        editor.elements.id.value = '';
        tab.dirty = true;
        tab.panel.querySelector('#save-state').textContent = 'Deleted request — unsaved draft';
        tab.panel.querySelector('#save-request').textContent = 'Save as new request';
        tab.panel.querySelector('.delete-form')?.remove();
      }
    }
  }
  async function open(url) {
    if (blocked()) return;
    const target = new URL(url, location.href);
    if (target.origin !== location.origin || target.pathname !== '/') return;
    const id = target.searchParams.get('id');
    const existing = id && tabs.find(tab => identity(tab.panel) === id);
    if (existing) { select(existing, true); return; }
    busy = true;
    try {
      const response = await fetch(target.href, { credentials: 'same-origin', redirect: 'error', cache: 'no-store' });
      if (!response.ok) throw new Error('HTTP ' + response.status);
      const page = new DOMParser().parseFromString(await response.text(), 'text/html');
      const panel = page.querySelector('#main');
      if (!panel || !form(panel)) throw new Error('Invalid editor response');
      refresh(page);
      register(panel);
      document.dispatchEvent(new CustomEvent('htmx:load'));
    } catch (error) { alert('Could not open request: ' + error.message); }
    finally { busy = false; }
  }
  function close(tab) {
    if (blocked()) return;
    if ((tab.dirty || uploads(tab)) && !confirm('Close this tab? Unsaved changes and selected files will be lost.')) return;
    if (tabs.length === 1) { open('/').then(() => { if (active !== tab) { tabs.splice(tabs.indexOf(tab), 1); render(); } }); return; }
    const index = tabs.indexOf(tab);
    if (tab === active) select(tabs[index === 0 ? 1 : index - 1], true);
    tabs.splice(index, 1); render();
  }
  document.addEventListener('click', event => {
    const link = event.target.closest('a[href]');
    if (!link || event.ctrlKey || event.metaKey || event.shiftKey || event.altKey || event.button) return;
    const url = new URL(link.href, location.href);
    if (url.origin !== location.origin || url.pathname !== '/') return;
    event.preventDefault(); event.stopImmediatePropagation();
    document.getElementById('collection-menu').hidden = true;
    open(url.href);
  }, true);
  for (const type of ['input', 'change']) document.addEventListener(type, event => {
    if (!event.target.closest('form[action="/request"]') || event.target.name === 'editor_tab') return;
    if (['collection', 'environment'].includes(event.target.name)) event.target.setCustomValidity('');
    revision++; active.dirty = true; render();
  });
  document.addEventListener('htmx:beforeRequest', event => {
    if (blocked()) { event.preventDefault(); return; }
    busy = true; submittedRevision = revision;
  });
  document.addEventListener('htmx:afterRequest', () => { busy = false; });
  document.addEventListener('htmx:beforeSwap', event => {
    if (!event.detail.shouldSwap) return;
    const page = new DOMParser().parseFromString(event.detail.serverResponse, 'text/html');
    const incoming = page.querySelector('#main');
    if (!incoming || !form(incoming)) return;
    event.detail.shouldSwap = false;
    const source = event.detail.requestConfig?.elt;
    const editing = source?.matches('form[action="/request"]');
    const notice = incoming.querySelector('.notice');
    active.panel.querySelector('.notice')?.remove();
    if (notice) form(active.panel).before(notice);
    if (editing) {
      const saved = notice?.textContent.startsWith('Saved to ');
      if (saved) {
        form(active.panel).elements.id.value = identity(incoming);
        if (!form(active.panel).elements.name.value) form(active.panel).elements.name.value = form(incoming).elements.name.value;
        active.dirty = revision !== submittedRevision;
        active.panel.querySelector('#save-state').textContent = active.dirty ? 'Unsaved changes' : 'Saved';
        active.panel.querySelector('#save-request').textContent = 'Save to destination';
        active.panel.querySelector('h1').textContent = title(active);
        active.panel.querySelector('.delete-form')?.remove();
        const deletion = incoming.querySelector('.delete-form');
        if (deletion) active.panel.append(deletion);
      } else if (event.detail.requestConfig.parameters.action === 'send') {
        active.panel.querySelector('.response').replaceWith(incoming.querySelector('.response'));
      }
    }
    refresh(page);
    // Refresh navigation/dialogs only. Never replace a live editor after sidebar actions.
    document.querySelector('aside').replaceWith(page.querySelector('aside'));
    for (const node of document.querySelectorAll('body > dialog, body > #collection-menu')) node.remove();
    for (const node of page.querySelectorAll('body > dialog, body > #collection-menu')) document.body.append(node);
    htmx.process(document.body);
    render();
    document.dispatchEvent(new CustomEvent('htmx:load'));
  });
  window.addEventListener('beforeunload', event => {
    if (busy || tabs.some(tab => tab.dirty || uploads(tab))) { event.preventDefault(); event.returnValue = ''; }
  });
  register(document.querySelector('#main'));
  const warning = document.createElement('small');
  warning.textContent = 'Refresh restores saved tabs only. Unsaved edits, drafts, responses and selected files will be lost.';
  warning.id = 'request-tabs-warning';
  warning.className = 'visually-hidden';
  list.setAttribute('aria-describedby', warning.id);
  strip.title = warning.textContent;
  strip.append(warning);
  restore();
})();
