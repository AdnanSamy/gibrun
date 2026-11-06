// Live DOM nodes retain drafts, response views and FileLists; nothing is persisted.
(() => {
  const tabs = [];
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
  add.type = 'button'; add.textContent = '+ New request';
  add.onclick = () => open('/');
  strip.append(list, add);
  document.querySelector('#main').before(strip);
  function render() {
    list.replaceChildren();
    for (const tab of tabs) {
      const group = document.createElement('div');
      group.setAttribute('role', 'presentation');
      const button = document.createElement('button');
      button.type = 'button'; button.id = 'request-tab-' + tab.key;
      button.setAttribute('role', 'tab');
      button.setAttribute('aria-selected', String(tab === active));
      button.setAttribute('aria-controls', 'main');
      button.tabIndex = tab === active ? 0 : -1;
      button.textContent = title(tab) + (tab.dirty ? ' •' : '');
      button.onclick = () => { if (!blocked()) select(tab); };
      button.onkeydown = event => {
        const index = tabs.indexOf(tab);
        const next = event.key === 'Home' ? 0 : event.key === 'End' ? tabs.length - 1 : event.key === 'ArrowRight' ? (index + 1) % tabs.length : event.key === 'ArrowLeft' ? (index + tabs.length - 1) % tabs.length : -1;
        if (next >= 0) { event.preventDefault(); if (!blocked()) select(tabs[next], true); }
        if (event.key === 'Delete') { event.preventDefault(); close(tab); }
      };
      const remove = document.createElement('button');
      remove.type = 'button'; remove.textContent = '×';
      remove.setAttribute('aria-label', 'Close ' + title(tab));
      remove.onclick = () => close(tab);
      group.append(button, remove); list.append(group);
    }
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
    button.scrollIntoView?.({ block: 'nearest', inline: 'nearest' });
    if (focus) button.focus();
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
    const id = new URL(url, location.href).searchParams.get('id');
    const existing = id && tabs.find(tab => identity(tab.panel) === id);
    if (existing) { select(existing, true); return; }
    busy = true;
    try {
      const response = await fetch(url, { credentials: 'same-origin' });
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
})();
