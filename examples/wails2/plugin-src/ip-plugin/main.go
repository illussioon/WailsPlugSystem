package main

import (
	"flag"
	"log"

	"github.com/illussioon/WailsPlugSystem/plugin"
)

const ipHTML = `
<section class="wailsplugs-ip-card" data-wailsplugs-ip>
  <div class="wailsplugs-ip-copy">
    <span class="wailsplugs-ip-label">Сетевой адрес</span>
    <strong data-ip-value>Ваш IP: определение...</strong>
  </div>
  <button type="button" data-ip-refresh>Обновить IP</button>
</section>
`

const ipCSS = `
.wailsplugs-ip-card {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 24px;
  margin-top: 28px;
  padding: 20px 22px;
  border: 1px solid rgba(96, 165, 250, 0.35);
  border-radius: 16px;
  background: rgba(30, 64, 175, 0.22);
}
.wailsplugs-ip-copy { display: grid; gap: 6px; }
.wailsplugs-ip-label { color: #93c5fd; font-size: 12px; letter-spacing: 0.08em; text-transform: uppercase; }
.wailsplugs-ip-card button {
  border: 0;
  border-radius: 10px;
  padding: 10px 14px;
  color: #eff6ff;
  background: #2563eb;
  cursor: pointer;
}
.wailsplugs-ip-card button:hover { background: #1d4ed8; }
`

const ipJS = `
(() => {
  const card = document.querySelector('[data-wailsplugs-ip]');
  if (!card) return;

  const value = card.querySelector('[data-ip-value]');
  const refresh = card.querySelector('[data-ip-refresh]');

  async function loadIP() {
    value.textContent = 'Ваш IP: запрашиваем...';
    refresh.disabled = true;
    try {
      const response = await fetch('https://api64.ipify.org?format=json', {
        headers: { Accept: 'application/json' },
      });
      if (!response.ok) throw new Error('HTTP ' + response.status);
      const payload = await response.json();
      value.textContent = 'Ваш IP: ' + (payload.ip || 'неизвестен');
    } catch (error) {
      console.error('WailsPlugSystem IP plugin:', error);
      value.textContent = 'Ваш IP: не удалось получить';
    } finally {
      refresh.disabled = false;
    }
  }

  refresh.addEventListener('click', loadIP);
  loadIP();
})();
`

func main() {
	output := flag.String("output", "../../plugins/ip.example.plugs", "output .plugs path")
	flag.Parse()

	definition := plugin.New("example.ip", "Example IP Plugin", "1.0.0").
		Priority(100).
		OnLoad("IP plugin loaded").
		OnUnload("IP plugin unloaded").
		HTML().
		AppendHTML("ip-card", "main", ipHTML).
		AddCSS("ip-style", "ip.css", []byte(ipCSS), plugin.WithConflictKey("example-ip-style")).
		AddJS("ip-script", "ip.js", []byte(ipJS), plugin.WithConflictKey("example-ip-script"))

	if _, err := definition.Build(*output); err != nil {
		log.Fatal(err)
	}
	log.Printf("built IP plugin: %s", *output)
}
