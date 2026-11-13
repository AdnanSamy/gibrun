// Only saved IDs/order/selection persist; live drafts, responses and files never do.
(() => {
  const storageKey = 'gibrun.request-tabs.v1';
  const legacyStorageKey = 'solarclient.request-tabs.v1';
  const maxTabs = 100;
  const form = panel => panel.querySelector('form[action="/request"]');
  const identity = panel => form(panel).elements.id.value;
  const title = tab => form(tab.panel).elements.name.value || 'Untitled request';
  const uploads = tab => [...tab.panel.querySelectorAll('input[type="file"]')].some(input => input.files.length);

  function readSession(storage) {
    try {
      const raw = storage.getItem(storageKey) ?? storage.getItem(legacyStorageKey);
      if (!raw || raw.length > 16000) return null;
      const value = JSON.parse(raw);
      if (value.version !== 1 || !Array.isArray(value.ids) || value.ids.length > maxTabs ||
          !value.ids.every(id => typeof id === 'string' && /^[A-Za-z0-9_-]{1,128}$/.test(id)) ||
          !(value.active === null || value.ids.includes(value.active))) return null;
      return { ids: [...new Set(value.ids)], active: value.active };
    } catch (_) {
      return null;
    }
  }

  function persist(state, storage) {
    if (state.restoring) return;
    const ids = [...new Set(state.tabs.map(tab => identity(tab.panel)).filter(id => /^[A-Za-z0-9_-]{1,128}$/.test(id)))].slice(0, maxTabs);
    const selected = identity(state.active.panel);
    try {
      storage.setItem(storageKey, JSON.stringify({ version: 1, ids, active: ids.includes(selected) ? selected : null }));
    } catch (_) {
      // Storage may be disabled/full.
    }
  }

  async function restore(state, view, services, session) {
    if (!session?.ids.length) {
      state.restoring = false;
      persist(state, services.storage);
      return;
    }
    state.busy = true;
    const initial = state.active;
    const startRevision = state.revision;
    const restored = [];
    try {
      for (const id of session.ids) {
        const existing = state.tabs.find(tab => identity(tab.panel) === id);
        if (existing) {
          restored.push(existing);
          continue;
        }
        try {
          const response = await services.fetch('/?id=' + encodeURIComponent(id), { credentials: 'same-origin', redirect: 'error', cache: 'no-store' });
          if (!response.ok) continue;
          const page = services.parser.parseFromString(await response.text(), 'text/html');
          const panel = page.querySelector('#main');
          // Deleted IDs render a blank editor.
          if (!panel || !form(panel) || identity(panel) !== id) continue;
          const tab = { panel, key: ++state.serial, dirty: false };
          state.tabs.push(tab);
          restored.push(tab);
          services.htmx.process(panel);
        } catch (_) {
          // Deleted/unavailable requests never replace a live draft.
        }
      }
      const untouched = state.revision === startRevision && !uploads(initial);
      const discardInitial = untouched && !restored.includes(initial) && restored.length;
      const remaining = state.tabs.filter(tab => !restored.includes(tab) && !(discardInitial && tab === initial));
      state.tabs.splice(0, state.tabs.length, ...restored, ...remaining);
      if (untouched) select(state, view, services.storage, restored.find(tab => identity(tab.panel) === session.active) || (discardInitial ? restored[0] : initial));
      document.dispatchEvent(new CustomEvent('htmx:load'));
    } finally {
      state.busy = false;
      state.restoring = false;
      render(state, view, services.storage);
    }
  }

  function blocked(busy) {
    if (!busy) return false;
    alert('A request is in progress. Wait for completion, or Cancel the send before changing tabs.');
    return true;
  }

  function updateOverflow({ list, previous, next }) {
    const overflowing = list.scrollWidth > list.clientWidth + 1;
    previous.hidden = !overflowing;
    next.hidden = !overflowing;
    previous.disabled = list.scrollLeft <= 1;
    next.disabled = list.scrollLeft + list.clientWidth >= list.scrollWidth - 1;
  }

  function revealActive(active, view) {
    const button = document.getElementById('request-tab-' + active.key);
    if (!button) return;
    const item = button.parentElement.getBoundingClientRect();
    const viewport = view.list.getBoundingClientRect();
    // Scroll only the strip: scrollIntoView can also move the document vertically.
    if (item.left < viewport.left) view.list.scrollLeft += item.left - viewport.left;
    else if (item.right > viewport.right) view.list.scrollLeft += item.right - viewport.right;
    updateOverflow(view);
  }

  function scrollTabs(view, direction) {
    view.list.scrollLeft += direction * view.list.clientWidth * 0.75;
    updateOverflow(view);
  }

  function wheelTabs(event, view) {
    const { list } = view;
    if (event.ctrlKey || (!event.shiftKey && Math.abs(event.deltaX) <= Math.abs(event.deltaY))) return;
    const delta = (event.deltaX || event.deltaY) * (event.deltaMode === 1 ? 16 : event.deltaMode === 2 ? list.clientWidth : 1);
    const before = list.scrollLeft;
    list.scrollLeft += delta;
    if (list.scrollLeft !== before) event.preventDefault();
    updateOverflow(view);
  }

  function render(state, view, storage) {
    persist(state, storage);
    view.list.replaceChildren();
    for (const tab of state.tabs) {
      const group = document.createElement('div');
      group.setAttribute('role', 'presentation');
      group.className = tab === state.active ? 'request-tab active' : 'request-tab';
      const button = document.createElement('button');
      button.type = 'button';
      button.id = 'request-tab-' + tab.key;
      button.dataset.tabKey = tab.key;
      button.setAttribute('role', 'tab');
      button.setAttribute('aria-selected', String(tab === state.active));
      button.setAttribute('aria-controls', 'main');
      button.tabIndex = tab === state.active ? 0 : -1;
      button.textContent = title(tab) + (tab.dirty ? ' •' : '');
      button.title = button.textContent;
      const remove = document.createElement('button');
      remove.type = 'button';
      remove.textContent = '×';
      remove.dataset.closeKey = tab.key;
      remove.className = 'close-request-tab';
      remove.setAttribute('aria-label', 'Close ' + title(tab));
      group.append(button, remove);
      view.list.append(group);
    }
    updateOverflow(view);
    state.active.panel.setAttribute('role', 'tabpanel');
    state.active.panel.setAttribute('aria-labelledby', 'request-tab-' + state.active.key);
    for (const link of document.querySelectorAll('.request-link')) {
      const selected = link.dataset.id === identity(state.active.panel);
      link.classList.toggle('active', selected);
      if (selected) link.setAttribute('aria-current', 'page');
      else link.removeAttribute('aria-current');
    }
  }

  function select(state, view, storage, tab, focus = false) {
    if (state.active !== tab) {
      state.active.panel.replaceWith(tab.panel);
      state.active = tab;
    }
    render(state, view, storage);
    const button = document.getElementById('request-tab-' + tab.key);
    revealActive(state.active, view);
    if (focus) button.focus({ preventScroll: true });
  }

  function register(state, view, services, panel) {
    const tab = { panel, key: ++state.serial, dirty: !identity(panel) };
    state.tabs.push(tab);
    if (!state.active) state.active = tab;
    select(state, view, services.storage, tab);
    services.htmx.process(panel);
  }

  function refresh(tabs, page) {
    const incoming = form(page.querySelector('#main'));
    const ids = new Set([...page.querySelectorAll('.request-link')].map(link => link.dataset.id));
    for (const tab of tabs) {
      const editor = form(tab.panel);
      for (const name of ['collection', 'environment']) {
        const select = editor.elements[name];
        const value = select.value;
        select.replaceChildren(...[...incoming.elements[name].options].map(option => option.cloneNode(true)));
        if (![...select.options].some(option => option.value === value)) {
          const missing = document.createElement('option');
          missing.value = value;
          missing.disabled = true;
          missing.textContent = 'Removed ' + (name === 'collection' ? 'destination' : 'environment') + ' — choose again';
          select.append(missing);
          select.setCustomValidity('Choose an existing ' + name + ' or explicitly select none.');
        } else {
          select.setCustomValidity('');
        }
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

  async function open(state, view, services, url) {
    if (blocked(state.busy)) return;
    const target = new URL(url, services.location.href);
    if (target.origin !== services.location.origin || target.pathname !== '/') return;
    const id = target.searchParams.get('id');
    const existing = id && state.tabs.find(tab => identity(tab.panel) === id);
    if (existing) {
      select(state, view, services.storage, existing, true);
      return;
    }
    state.busy = true;
    try {
      const response = await services.fetch(target.href, { credentials: 'same-origin', redirect: 'error', cache: 'no-store' });
      if (!response.ok) throw new Error('HTTP ' + response.status);
      const page = services.parser.parseFromString(await response.text(), 'text/html');
      const panel = page.querySelector('#main');
      if (!panel || !form(panel)) throw new Error('Invalid editor response');
      refresh(state.tabs, page);
      register(state, view, services, panel);
      document.dispatchEvent(new CustomEvent('htmx:load'));
    } catch (error) {
      alert('Could not open request: ' + error.message);
    } finally {
      state.busy = false;
    }
  }

  async function close(state, view, services, tab) {
    if (blocked(state.busy)) return;
    if ((tab.dirty || uploads(tab)) && !confirm('Close this tab? Unsaved changes and selected files will be lost.')) return;
    if (state.tabs.length === 1) {
      await open(state, view, services, '/');
      if (state.active !== tab) {
        state.tabs.splice(state.tabs.indexOf(tab), 1);
        render(state, view, services.storage);
      }
      return;
    }
    const index = state.tabs.indexOf(tab);
    if (tab === state.active) select(state, view, services.storage, state.tabs[index === 0 ? 1 : index - 1], true);
    state.tabs.splice(index, 1);
    render(state, view, services.storage);
  }

  function tabIndex(key, index, count) {
    switch (key) {
      case 'Home': return 0;
      case 'End': return count - 1;
      case 'ArrowRight': return (index + 1) % count;
      case 'ArrowLeft': return (index + count - 1) % count;
      default: return -1;
    }
  }

  function tabKeydown(event, state, view, services) {
    const key = event.target.closest('[data-tab-key]')?.dataset.tabKey;
    const tab = state.tabs.find(tab => String(tab.key) === key);
    if (!tab) return;
    const next = tabIndex(event.key, state.tabs.indexOf(tab), state.tabs.length);
    if (next >= 0) {
      event.preventDefault();
      if (!blocked(state.busy)) select(state, view, services.storage, state.tabs[next], true);
    }
    if (event.key === 'Delete') {
      event.preventDefault();
      close(state, view, services, tab);
    }
  }

  function navigate(event, state, view, services) {
    const link = event.target.closest('a[href]');
    if (!link || event.ctrlKey || event.metaKey || event.shiftKey || event.altKey || event.button) return;
    const url = new URL(link.href, services.location.href);
    if (url.origin !== services.location.origin || url.pathname !== '/') return;
    event.preventDefault();
    event.stopImmediatePropagation();
    document.getElementById('collection-menu').hidden = true;
    open(state, view, services, url.href);
  }

  function markDirty(event, state, view, storage) {
    if (!event.target.closest('form[action="/request"]') || event.target.name === 'editor_tab') return;
    if (['collection', 'environment'].includes(event.target.name)) event.target.setCustomValidity('');
    state.revision++;
    state.active.dirty = true;
    render(state, view, storage);
  }

  function beforeRequest(event, state) {
    if (blocked(state.busy)) {
      event.preventDefault();
      return;
    }
    state.busy = true;
    state.submittedRevision = state.revision;
  }

  function beforeSwap(event, state, view, services) {
    if (!event.detail.shouldSwap) return;
    const page = services.parser.parseFromString(event.detail.serverResponse, 'text/html');
    const incoming = page.querySelector('#main');
    if (!incoming || !form(incoming)) return;
    event.detail.shouldSwap = false;
    const source = event.detail.requestConfig?.elt;
    const editing = source?.matches('form[action="/request"]');
    const notice = incoming.querySelector('.notice');
    const active = state.active;
    active.panel.querySelector('.notice')?.remove();
    if (notice) form(active.panel).before(notice);
    if (editing) {
      const saved = notice?.textContent.startsWith('Saved to ');
      if (saved) {
        form(active.panel).elements.id.value = identity(incoming);
        if (!form(active.panel).elements.name.value) form(active.panel).elements.name.value = form(incoming).elements.name.value;
        active.dirty = state.revision !== state.submittedRevision;
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
    refresh(state.tabs, page);
    // Refresh navigation/dialogs only. Never replace a live editor after sidebar actions.
    document.querySelector('aside').replaceWith(page.querySelector('aside'));
    for (const node of document.querySelectorAll('body > dialog, body > #collection-menu')) node.remove();
    for (const node of page.querySelectorAll('body > dialog, body > #collection-menu')) document.body.append(node);
    services.htmx.process(document.body);
    render(state, view, services.storage);
    document.dispatchEvent(new CustomEvent('htmx:load'));
  }

  // Bootstrap owns mutable workspace state; event adapters pass it explicitly.
  const state = { tabs: [], active: undefined, busy: false, restoring: true, serial: 0, revision: 0, submittedRevision: 0 };
  // Access storage lazily: the browser may throw even when reading sessionStorage.
  const storage = {
    getItem: key => sessionStorage.getItem(key),
    setItem: (key, value) => sessionStorage.setItem(key, value),
  };
  const services = { storage, fetch: window.fetch.bind(window), parser: new DOMParser(), htmx, location };
  const session = readSession(storage);
  const strip = document.createElement('div');
  strip.className = 'workspace-tabs';
  const list = document.createElement('div');
  list.setAttribute('role', 'tablist');
  list.setAttribute('aria-label', 'Open requests');
  list.id = 'request-tab-list';
  const add = document.createElement('button');
  add.type = 'button';
  add.textContent = '+';
  add.className = 'new-request-tab';
  add.setAttribute('aria-label', 'New request');
  add.title = 'New request';
  const previous = document.createElement('button');
  const next = document.createElement('button');
  const view = { list, previous, next };
  add.onclick = () => open(state, view, services, '/');
  for (const [button, direction, label] of [[previous, -1, 'Scroll tabs left'], [next, 1, 'Scroll tabs right']]) {
    button.type = 'button';
    button.className = 'tab-scroll';
    button.textContent = direction < 0 ? '‹' : '›';
    button.setAttribute('aria-label', label);
    button.setAttribute('aria-controls', list.id);
    button.hidden = true;
    button.onclick = () => scrollTabs(view, direction);
  }
  strip.append(previous, list, next, add);
  document.querySelector('#main').before(strip);
  list.addEventListener('click', event => {
    const button = event.target.closest('button');
    if (!button) return;
    const tab = state.tabs.find(tab => String(tab.key) === (button.dataset.tabKey || button.dataset.closeKey));
    if (!tab) return;
    if (button.dataset.closeKey) close(state, view, services, tab);
    else if (!blocked(state.busy)) select(state, view, storage, tab);
  });
  list.addEventListener('keydown', event => tabKeydown(event, state, view, services));
  list.addEventListener('scroll', () => updateOverflow(view), { passive: true });
  list.addEventListener('wheel', event => wheelTabs(event, view), { passive: false });
  const observer = new ResizeObserver(() => {
    updateOverflow(view);
    revealActive(state.active, view);
  });
  observer.observe(list);
  document.addEventListener('htmx:beforeCleanupElement', function cleanup(event) {
    if (event.detail.elt === strip || event.detail.elt.contains(strip)) {
      observer.disconnect();
      document.removeEventListener('htmx:beforeCleanupElement', cleanup);
    }
  });
  document.addEventListener('click', event => navigate(event, state, view, services), true);
  for (const type of ['input', 'change']) {
    document.addEventListener(type, event => markDirty(event, state, view, storage));
  }
  document.addEventListener('htmx:beforeRequest', event => beforeRequest(event, state));
  document.addEventListener('htmx:afterRequest', () => { state.busy = false; });
  document.addEventListener('htmx:beforeSwap', event => beforeSwap(event, state, view, services));
  window.addEventListener('beforeunload', event => {
    if (state.busy || state.tabs.some(tab => tab.dirty || uploads(tab))) {
      event.preventDefault();
      event.returnValue = '';
    }
  });
  register(state, view, services, document.querySelector('#main'));
  const warning = document.createElement('small');
  warning.textContent = 'Refresh restores saved tabs only. Unsaved edits, drafts, responses and selected files will be lost.';
  warning.id = 'request-tabs-warning';
  warning.className = 'visually-hidden';
  list.setAttribute('aria-describedby', warning.id);
  strip.title = warning.textContent;
  strip.append(warning);
  restore(state, view, services, session);
})();
