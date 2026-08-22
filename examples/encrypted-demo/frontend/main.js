const report = document.querySelector('#plugin-report');
const status = document.querySelector('#host-status');
const refresh = document.querySelector('#refresh-report');
const reload = document.querySelector('#reload-plugins');

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
  const app = window.go.main.App;
  const plugins = await app.GetPluginReport();
  const securityStatus = await app.GetSecurityStatus();
  report.innerHTML = plugins.map((plugin) => `
    <article class="report-card">
      <span class="priority">priority ${escapeHTML(plugin.priority)}</span>
      <h3>${escapeHTML(plugin.name)} <small>(${escapeHTML(plugin.id)})</small></h3>
      <p>Version ${escapeHTML(plugin.version)} · ${escapeHTML(plugin.files)} assets</p>
      <p>Encryption: <strong>${escapeHTML(plugin.encryption || 'none')}</strong></p>
      <p>Permissions: ${escapeHTML((plugin.permissions || []).join(', ') || 'none')}</p>
      <p>SHA-256: <code>${escapeHTML(plugin.sha256)}</code></p>
    </article>
  `).join('');
  status.textContent = securityStatus || `${plugins.length} plugin(s) active`;
}

async function reloadPlugins() {
  if (!window.go?.main?.App?.ReloadPlugins) return;
  status.textContent = 'Decrypting and verifying...';
  try {
    await window.go.main.App.ReloadPlugins();
  } catch (error) {
    // The host keeps the previous safe snapshot and exposes the failure status.
  }
  await refreshReport();
}

refresh?.addEventListener('click', refreshReport);
reload?.addEventListener('click', reloadPlugins);
window.addEventListener('DOMContentLoaded', refreshReport);
