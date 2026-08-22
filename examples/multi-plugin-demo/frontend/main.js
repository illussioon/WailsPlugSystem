const report = document.querySelector('#plugin-report');
const status = document.querySelector('#host-status');
const refresh = document.querySelector('#refresh-report');

function escapeHTML(value) {
  return String(value ?? '').replace(/[&<>'"]/g, (char) => ({
    '&': '&amp;', '<': '&lt;', '>': '&gt;', "'": '&#39;', '"': '&quot;'
  })[char]);
}

async function refreshReport() {
  if (!window.go?.main?.App?.GetPluginReport) {
    status.textContent = 'Wails bindings are not available';
    return;
  }
  const plugins = await window.go.main.App.GetPluginReport();
  report.innerHTML = plugins.map((plugin) => `
    <article class="report-card">
      <span class="priority">priority ${escapeHTML(plugin.priority)}</span>
      <h3>${escapeHTML(plugin.name)} <small>(${escapeHTML(plugin.id)})</small></h3>
      <p>Version ${escapeHTML(plugin.version)} · ${escapeHTML(plugin.files)} archive files</p>
      <p>Permissions: ${escapeHTML((plugin.permissions || []).join(', ') || 'none')}</p>
      <p>SHA-256: <code>${escapeHTML(plugin.sha256)}</code></p>
      <p>Lifecycle: load “${escapeHTML(plugin.on_load)}” · unload “${escapeHTML(plugin.on_unload)}”</p>
    </article>
  `).join('');
  status.textContent = `${plugins.length} plugins active`;
}

refresh?.addEventListener('click', refreshReport);
window.addEventListener('DOMContentLoaded', refreshReport);
