(function () {
  const page = document.body.dataset.page;

  function text(id, value) {
    const element = document.getElementById(id);
    if (element && value !== undefined && value !== null) {
      element.textContent = value;
    }
  }

  async function fetchJSON(url) {
    const response = await fetch(url, { headers: { Accept: "application/json" } });
    if (!response.ok) {
      throw new Error("request failed: " + response.status);
    }
    return response.json();
  }

  async function refreshOverview() {
    const snapshot = await fetchJSON("/api/v1/cluster");
    text("leader-indicator", snapshot.leader_id || "electing");
    text("rendered-at", new Date(snapshot.generated_at).toLocaleString());
    text("ingress-sync", snapshot.ingress && snapshot.ingress.dns_synced ? "SYNCED" : "PENDING");
    text("ingress-ip", snapshot.ingress ? (snapshot.ingress.desired_ip || "-") : "-");
    if (!Array.isArray(snapshot.nodes)) return;
    snapshot.nodes.forEach((node) => {
      const card = document.querySelector('[data-node-card="' + node.node_id + '"]');
      if (!card) return;
      const reason = card.querySelector('[data-field="reason"]');
      const cpu = card.querySelector('[data-field="cpu"]');
      const mem = card.querySelector('[data-field="mem"]');
      const disk = card.querySelector('[data-field="disk"]');
      const heartbeat = card.querySelector('[data-field="heartbeat"]');
      if (reason) reason.textContent = node.reason;
      if (cpu) cpu.textContent = Math.round(node.cpu_pct) + "%";
      if (mem) mem.textContent = Math.round(node.mem_pct) + "%";
      if (disk) disk.textContent = Math.round(node.disk_pct) + "%";
      if (heartbeat && node.last_heartbeat_at) {
        heartbeat.textContent = new Date(node.last_heartbeat_at).toLocaleTimeString();
      }
      card.classList.remove("tone-ok", "tone-warn", "tone-bad", "tone-unknown");
      card.classList.add(statusClass(node.status));
    });
  }

  async function refreshNode() {
    const nodeID = document.querySelector("[data-node-page]")?.dataset.nodePage;
    if (!nodeID) return;
    const detail = await fetchJSON("/api/v1/nodes/" + encodeURIComponent(nodeID));
    text("rendered-at", new Date().toLocaleString());
    text("node-cpu", Math.round(detail.state.cpu_pct) + "%");
    text("node-mem", Math.round(detail.state.mem_pct) + "%");
    text("node-disk", Math.round(detail.state.disk_pct) + "%");
  }

  async function refreshEvents() {
    const events = await fetchJSON("/api/v1/events?limit=20");
    text("rendered-at", new Date().toLocaleString());
    const container = document.getElementById("timeline-events");
    if (!container || !Array.isArray(events)) return;
    container.innerHTML = events.map((event) => `
      <article class="timeline__item">
        <div>
          <p>${event.kind}</p>
          <strong>${event.title}</strong>
        </div>
        <div>
          <span>${event.node_id || ""}</span>
          <small>${new Date(event.created_at).toLocaleString()}</small>
        </div>
      </article>
    `).join("");
  }

  async function submitTestAlert(form) {
    const result = document.getElementById("test-alert-result");
    if (result) {
      result.innerHTML = "<p>正在发送测试告警...</p>";
    }
    const payload = {
      channel: form.channel.value,
      token: form.token ? form.token.value : "",
      note: form.note.value
    };
    const response = await fetch("/api/v1/test-alert", {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        "Accept": "application/json"
      },
      body: JSON.stringify(payload)
    });
    const body = await response.json();
    if (!response.ok) {
      throw new Error(body.error || ("request failed: " + response.status));
    }
    if (!result) return;
    const items = Array.isArray(body.results) ? body.results : [];
    result.innerHTML = `
      <p>发送时间 ${new Date(body.sent_at).toLocaleString()}</p>
      ${items.map((item) => `
        <div class="test-alert-result__item ${statusClass(item.ok ? "healthy" : "critical")}">
          <strong>${item.channel}</strong>
          <span>${item.ok ? "OK" : (item.error || "FAILED")}</span>
        </div>
      `).join("")}
    `;
  }

  function statusClass(status) {
    switch (status) {
      case "healthy":
        return "tone-ok";
      case "degraded":
        return "tone-warn";
      case "critical":
        return "tone-bad";
      default:
        return "tone-unknown";
    }
  }

  const refreshMap = {
    overview: refreshOverview,
    node: refreshNode,
    events: refreshEvents
  };

  if (!refreshMap[page]) return;

  const testForm = document.getElementById("test-alert-form");
  if (testForm) {
    testForm.addEventListener("submit", async (event) => {
      event.preventDefault();
      try {
        await submitTestAlert(testForm);
      } catch (error) {
        const result = document.getElementById("test-alert-result");
        if (result) {
          result.innerHTML = "<p>" + error.message + "</p>";
        }
      }
    });
  }

  refreshMap[page]().catch(() => {});
  window.setInterval(() => {
    refreshMap[page]().catch(() => {});
  }, 15000);
})();
