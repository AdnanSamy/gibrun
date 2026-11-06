// Flat preorder rows retain matching ancestors and descendants without a DOM tree.
function searchVisibility(name, rows, query) {
  const term = query.trim().toLowerCase();
  const all = !term || name.toLowerCase().includes(term);
  const visible = rows.map(() => all);
  const parents = [];
  rows.forEach((row, i) => {
    while (parents.length && rows[parents.at(-1)].depth >= row.depth) parents.pop();
    const match = [row.name, row.method, row.url].some(value => (value || '').toLowerCase().includes(term));
    visible[i] = all || match || parents.some(index => rows[index].matched);
    if (visible[i]) parents.forEach(index => { visible[index] = true; });
    // Only direct matches propagate down, not ancestors retained as context.
    if (row.folder) { row.matched = match; parents.push(i); }
  });
  return { visible, show: all || visible.some(Boolean) };
}
(() => {
  const openStates = new WeakMap();
  function filter() {
    const input = document.getElementById('collection-search');
    if (!input) return;
    let found = false;
    for (const collection of document.querySelectorAll('#collection-results > details')) {
      const nodes = [...collection.querySelectorAll('[data-kind="folder"], [data-kind="request"]')];
      const rows = nodes.map(node => ({ name: node.dataset.name, method: node.querySelector('[data-method]')?.dataset.method, url: node.dataset.searchUrl, depth: Number(node.dataset.depth || 0), folder: node.dataset.kind === 'folder' }));
      const result = searchVisibility(collection.querySelector('summary').textContent, rows, input.value);
      nodes.forEach((node, i) => { node.hidden = !result.visible[i]; });
      collection.hidden = !result.show;
      found ||= result.show;
      if (input.value.trim()) {
        if (!openStates.has(collection)) openStates.set(collection, collection.open);
        if (result.show) collection.open = true;
      } else if (openStates.has(collection)) {
        collection.open = openStates.get(collection);
        openStates.delete(collection);
      }
    }
    document.getElementById('search-empty').hidden = found;
  }
  function clear() {
    const input = document.getElementById('collection-search');
    input.value = '';
    filter();
    input.focus();
  }
  document.addEventListener('input', event => { if (event.target.id === 'collection-search') filter(); });
  document.addEventListener('click', event => { if (event.target.id === 'clear-collection-search') clear(); });
  document.addEventListener('keydown', event => {
    if (event.target.id === 'collection-search' && event.key === 'Escape') { event.preventDefault(); clear(); }
  });
  document.addEventListener('htmx:load', filter);
  filter();
})();

// Delegated listeners survive HTMX body replacements.
(() => {
  let context;
  function filterParents() {
    const form = document.querySelector('#organize-dialog form');
    const collection = form.elements.collection.value;
    for (const option of form.elements.parent.options) {
      option.disabled = !!option.value && option.dataset.collection !== collection;
    }
    if (form.elements.parent.selectedOptions[0]?.disabled) form.elements.parent.value = '';
  }
  document.addEventListener('change', event => {
    if (event.target.id === 'organize-collection') filterParents();
    if (event.target.id === 'environment') document.cookie = 'solar_environment=' + encodeURIComponent(event.target.value) + '; Path=/; SameSite=Strict';
  });
  function closeMenu() {
    const menu = document.getElementById('collection-menu');
    if (menu) menu.hidden = true;
  }
  function openMenu(target, x, y) {
    const menu = document.getElementById('collection-menu');
    const collection = target.closest('details[data-collection]');
    context = target.closest('[data-kind]');
    for (const button of menu.querySelectorAll('[data-organize]')) {
      button.hidden = !context || (button.dataset.organize === 'move' && context.dataset.kind === 'collection') || (button.dataset.organize === 'create' && context.dataset.kind === 'request') || (button.dataset.organize === 'duplicate' && context.dataset.kind !== 'request');
    }
    document.getElementById('context-new-request').href = collection
      ? '/?collection=' + encodeURIComponent(collection.dataset.collection) : '/';
    menu.hidden = false;
    menu.style.left = Math.max(0, Math.min(x, innerWidth - menu.offsetWidth)) + 'px';
    menu.style.top = Math.max(0, Math.min(y, innerHeight - menu.offsetHeight)) + 'px';
    menu.querySelector('[role="menuitem"]').focus();
  }
  function updateSaveLabel() {
    const destination = document.getElementById('collection');
    const button = document.getElementById('save-request');
    if (destination && button) button.textContent = 'Save to ' + destination.selectedOptions[0].textContent;
  }
  function markUnsaved(event) {
    if (!event.target.closest('form[action="/request"]') || ['environment', 'editor_tab'].includes(event.target.name)) return;
    const state = document.getElementById('save-state');
    if (state) state.textContent = 'Unsaved changes';
    updateSaveLabel();
  }
  document.addEventListener('input', markUnsaved);
  document.addEventListener('change', markUnsaved);
  document.addEventListener('htmx:load', updateSaveLabel);
  updateSaveLabel();
  document.addEventListener('contextmenu', event => {
    if (!event.target.closest('aside') || event.target.closest('input,textarea,select,dialog')) return;
    event.preventDefault();
    openMenu(event.target, event.clientX, event.clientY);
  });
  document.addEventListener('click', event => {
    closeMenu();
    const dialogID = event.target.closest('[data-open-dialog]')?.dataset.openDialog;
    if (dialogID) document.getElementById(dialogID)?.showModal();
    if (event.target.closest('[data-new-collection]')) {
      document.getElementById('collection-dialog').showModal();
      document.getElementById('collection-name').focus();
    }
    if (event.target.closest('[data-close-dialog]')) event.target.closest('dialog').close();
    if (event.target.closest('[data-confirm-delete]') && !window.confirm('Delete this request? 0 descendant folders, 1 requests. This cannot be undone.')) event.preventDefault();
    const action = event.target.closest('[data-organize]')?.dataset.organize;
    if (action && context) {
      const dialog = document.getElementById('organize-dialog');
      const form = dialog.querySelector('form');
      form.reset();
      form.elements.id.value = context.dataset.id;
      form.elements.kind.value = action === 'create' ? 'folder' : context.dataset.kind;
      form.elements.action.value = action;
      form.elements.name.value = action === 'create' ? '' : context.dataset.name;
      form.elements.name.required = action === 'create' || action === 'rename';
      form.elements.counts.value = context.dataset.counts;
      form.elements.collection.value = context.closest('details[data-collection]')?.dataset.collection || '';
      filterParents();
      if (action === 'create' && context.dataset.kind === 'folder') form.elements.parent.value = context.dataset.id;
      dialog.querySelector('[data-organize-name]').hidden = !form.elements.name.required;
      dialog.querySelector('[data-organize-destination]').hidden = action !== 'create' && action !== 'move';
      document.getElementById('organize-title').textContent = action + ' ' + form.elements.kind.value;
      dialog.querySelector('[data-organize-warning]').textContent = action === 'delete' ? 'Delete “' + context.dataset.name + '”: ' + context.dataset.counts + '. This cannot be undone.' : action === 'duplicate' ? 'Copy the saved request into the same destination. Unsaved edits and selected uploads are not copied.' : '';
      dialog.showModal();
    }
  });
  document.addEventListener('keydown', event => {
    if (event.key === 'Escape') closeMenu();
    if ((event.key === 'ContextMenu' || (event.shiftKey && event.key === 'F10')) && event.target.closest('aside')) {
      event.preventDefault();
      const rect = event.target.getBoundingClientRect();
      openMenu(event.target, rect.left, rect.bottom);
      return;
    }
    const menu = event.target.closest('[role="menu"]');
    if (menu && ['ArrowDown', 'ArrowUp', 'Home', 'End'].includes(event.key)) {
      event.preventDefault();
      const items = [...menu.querySelectorAll('[role="menuitem"]')].filter(item => !item.hidden);
      const index = items.indexOf(document.activeElement);
      items[event.key === 'Home' ? 0 : event.key === 'End' ? items.length - 1 : (index + (event.key === 'ArrowDown' ? 1 : -1) + items.length) % items.length].focus();
    }
    if (event.target.id === 'body' && event.key === 'Tab' && !event.shiftKey && !event.ctrlKey && !event.altKey && !event.metaKey) {
      event.preventDefault();
      const editor = event.target;
      editor.setRangeText('  ', editor.selectionStart, editor.selectionEnd, 'end');
      editor.dispatchEvent(new Event('input', { bubbles: true }));
    }
  });
  window.addEventListener('resize', closeMenu);
  document.addEventListener('scroll', closeMenu, true);
})();

// Token text is always inserted as text nodes, never interpreted as markup.
function highlightJSON(target, text) {
  const fragment = document.createDocumentFragment();
  const tokens = /"(?:\\[^]|[^"\\])*"|-?\d+(?:\.\d+)?(?:[eE][+-]?\d+)?|\b(?:true|false|null)\b/g;
  let end = 0;
  for (const match of text.matchAll(tokens)) {
    fragment.append(document.createTextNode(text.slice(end, match.index)));
    const span = document.createElement('span');
    span.className = match[0][0] === '"' ? 'json-string' : /^-?\d/.test(match[0]) ? 'json-number' : 'json-literal';
    span.textContent = match[0];
    fragment.append(span);
    end = match.index + match[0].length;
  }
  fragment.append(document.createTextNode(text.slice(end)));
  target.replaceChildren(fragment);
}
(() => {
  function editor() {
    const area = document.getElementById('body');
    if (!area) return;
    const pre = document.getElementById('body-highlight');
    // ponytail: large documents remain plain; add a worker for larger highlighting.
    const enabled = document.getElementById('body-type').value === 'json' && area.value.length <= 200000;
    area.parentElement.classList.toggle('highlighted', enabled);
    if (enabled) highlightJSON(pre, area.value + '\n');
    pre.scrollTop = area.scrollTop;
    pre.scrollLeft = area.scrollLeft;
  }
  function load() {
    editor();
    for (const pre of document.querySelectorAll('.response-content pre')) {
      if (pre.textContent.length > 200000) continue;
      try { JSON.parse(pre.textContent); highlightJSON(pre, pre.textContent); } catch (_) { /* Plain response fallback. */ }
    }
  }
  document.addEventListener('input', event => { if (event.target.id === 'body') editor(); });
  document.addEventListener('change', event => {
    if (event.target.id === 'body-type') editor();
    if (event.target.id !== 'body-fields') return;
    let fields;
    try { fields = JSON.parse(event.target.value); if (!Array.isArray(fields)) return; } catch (_) { return; }
    const container = document.getElementById('body-uploads');
    if ([...container.querySelectorAll('input[type="file"]')].some(input => input.files.length) && !confirm('Changing form fields clears selected uploads. Continue?')) return;
    container.replaceChildren();
    fields.forEach((field, i) => {
      if (!field?.file) return;
      const label = document.createElement('label');
      label.htmlFor = 'upload-' + i;
      label.textContent = field.key + ': ' + field.file + ' — reselect file';
      const input = document.createElement('input');
      input.type = 'file'; input.id = label.htmlFor; input.name = 'upload_' + i;
      container.append(label, input);
    });
  });
  document.addEventListener('scroll', event => { if (event.target.id === 'body') {
    const pre = document.getElementById('body-highlight');
    pre.scrollTop = event.target.scrollTop; pre.scrollLeft = event.target.scrollLeft;
  } }, true);
  document.addEventListener('htmx:load', load);
  load();
})();
