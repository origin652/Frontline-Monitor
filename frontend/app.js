(function () {
  const app = document.getElementById("app");
  if (!app) {
    return;
  }

  const REFRESH_INTERVAL_MS = 15000;
  const THEME_KEY = "vps-monitor-theme";
  const THEMES = [
    { id: "graphite", label: "Graphite" },
    { id: "porcelain", label: "Porcelain" },
    { id: "brass", label: "Brass" }
  ];

  let refreshTimer = 0;
  let renderToken = 0;

  document.addEventListener("click", handleDocumentClick);
  document.addEventListener("change", handleDocumentChange);
  document.addEventListener("submit", handleDocumentSubmit);
  window.addEventListener("popstate", () => {
    renderRoute();
  });

  applyTheme(document.documentElement.dataset.theme || document.body.dataset.defaultTheme || "graphite");
  renderRoute();

  function handleDocumentClick(event) {
    const link = event.target.closest("a[data-link]");
    if (!link) {
      return;
    }
    if (
      event.defaultPrevented ||
      event.button !== 0 ||
      event.metaKey ||
      event.ctrlKey ||
      event.shiftKey ||
      event.altKey
    ) {
      return;
    }

    const url = new URL(link.href, window.location.origin);
    if (url.origin !== window.location.origin) {
      return;
    }

    event.preventDefault();
    if (url.pathname !== window.location.pathname) {
      window.history.pushState({}, "", url.pathname);
    }
    renderRoute();
  }

  function handleDocumentChange(event) {
    if (!event.target.matches("[data-theme-select]")) {
      return;
    }
    applyTheme(event.target.value);
  }

  async function handleDocumentSubmit(event) {
    if (event.target.id !== "test-alert-form") {
      return;
    }
    event.preventDefault();
    try {
      await submitTestAlert(event.target);
    } catch (error) {
      const result = document.getElementById("test-alert-result");
      if (result) {
        result.innerHTML = '<p>' + escapeHTML(error.message || "发送失败") + "</p>";
      }
    }
  }

  async function renderRoute() {
    window.clearTimeout(refreshTimer);
    const token = ++renderToken;
    const route = resolveRoute(window.location.pathname);
    document.body.dataset.page = route.page;

    app.innerHTML = renderShell({
      page: route.page,
      title: routeTitle(route),
      localNodeID: "...",
      leaderID: "加载中",
      generatedAt: new Date().toISOString(),
      content: renderStatePanel("加载中", "正在从 API 拉取最新集群状态。", route)
    });
    syncThemeSelect();

    try {
      const view = await loadRoute(route);
      if (token !== renderToken) {
        return;
      }

      document.title = view.title + " · VPS Monitor";
      app.innerHTML = renderShell(view);
      syncThemeSelect();
      scheduleRefresh();
    } catch (error) {
      if (token !== renderToken) {
        return;
      }

      document.title = routeTitle(route) + " · VPS Monitor";
      app.innerHTML = renderShell({
        page: route.page,
        title: routeTitle(route),
        localNodeID: "-",
        leaderID: "不可用",
        generatedAt: new Date().toISOString(),
        content: renderStatePanel("加载失败", error.message || "接口请求失败。", route)
      });
      syncThemeSelect();
      scheduleRefresh();
    }
  }

  function scheduleRefresh() {
    window.clearTimeout(refreshTimer);
    refreshTimer = window.setTimeout(() => {
      const form = document.getElementById("test-alert-form");
      if (form) {
        const note = form.querySelector('textarea[name="note"]');
        const token = form.querySelector('input[name="token"]');
        const hasDraft =
          form.contains(document.activeElement) ||
          (note && note.value.trim() !== "") ||
          (token && token.value.trim() !== "");
        if (hasDraft) {
          scheduleRefresh();
          return;
        }
      }
      renderRoute();
    }, REFRESH_INTERVAL_MS);
  }

  function resolveRoute(pathname) {
    const normalized = pathname.replace(/\/+$/, "") || "/";
    if (normalized === "/") {
      return { page: "overview" };
    }
    if (normalized === "/events") {
      return { page: "events" };
    }
    const nodeMatch = normalized.match(/^\/nodes\/([^/]+)$/);
    if (nodeMatch) {
      return { page: "node", nodeID: decodeURIComponent(nodeMatch[1]) };
    }
    return { page: "not-found" };
  }

  function routeTitle(route) {
    switch (route.page) {
      case "overview":
        return "总览";
      case "events":
        return "事件";
      case "node":
        return "节点 " + route.nodeID;
      default:
        return "未找到页面";
    }
  }

  function routeMainClass(route) {
    switch (route.page) {
      case "overview":
        return "overview";
      case "node":
        return "node-page";
      default:
        return "events-page";
    }
  }

  async function loadRoute(route) {
    switch (route.page) {
      case "overview":
        return loadOverviewView();
      case "events":
        return loadEventsView();
      case "node":
        return loadNodeView(route.nodeID);
      default:
        return {
          page: "not-found",
          title: "未找到页面",
          localNodeID: "-",
          leaderID: "",
          generatedAt: new Date().toISOString(),
          content: renderStatePanel("页面不存在", "这个路径没有对应的前端页面。", route)
        };
    }
  }

  async function loadOverviewView() {
    const snapshot = await fetchJSON("/api/v1/cluster");
    const nodes = Array.isArray(snapshot.nodes) ? snapshot.nodes : [];
    const historyEntries = await Promise.all(
      nodes.map(async (node) => {
        try {
          const points = await fetchHistory(node.node_id, "cpu_pct");
          return [node.node_id, points];
        } catch (error) {
          return [node.node_id, []];
        }
      })
    );
    const historyMap = Object.fromEntries(historyEntries);

    return {
      page: "overview",
      title: "总览",
      localNodeID: snapshot.node_id || "-",
      leaderID: snapshot.leader_id || "",
      generatedAt: snapshot.generated_at || new Date().toISOString(),
      content: renderOverview(snapshot, historyMap)
    };
  }

  async function loadNodeView(nodeID) {
    const [detail, snapshot, memHistory, diskHistory] = await Promise.all([
      fetchJSON("/api/v1/nodes/" + encodeURIComponent(nodeID)),
      fetchJSON("/api/v1/cluster"),
      fetchHistory(nodeID, "mem_pct").catch(() => []),
      fetchHistory(nodeID, "disk_pct").catch(() => [])
    ]);

    return {
      page: "node",
      title: "节点 " + nodeID,
      localNodeID: snapshot.node_id || "-",
      leaderID: snapshot.leader_id || "",
      generatedAt: snapshot.generated_at || new Date().toISOString(),
      content: renderNodePage(nodeID, detail, snapshot, memHistory, diskHistory)
    };
  }

  async function loadEventsView() {
    const [snapshot, incidents, events, meta] = await Promise.all([
      fetchJSON("/api/v1/cluster"),
      fetchJSON("/api/v1/incidents?limit=30"),
      fetchJSON("/api/v1/events?limit=40"),
      fetchJSON("/api/v1/meta").catch(() => ({
        test_alert_channels: [],
        test_alert_requires_token: false
      }))
    ]);

    return {
      page: "events",
      title: "事件",
      localNodeID: snapshot.node_id || meta.node_id || "-",
      leaderID: snapshot.leader_id || meta.leader_id || "",
      generatedAt: snapshot.generated_at || new Date().toISOString(),
      content: renderEventsPage(snapshot, incidents, events, meta)
    };
  }

  function renderShell(view) {
    return `
      <div class="shell">
        <header class="masthead">
          <div class="masthead__brand">
            <p class="eyebrow">Signal Atelier · Same-Repo Frontend + Go API</p>
            <h1>Frontline Monitor</h1>
            <p class="masthead__lede">前端已从后端模板里拆出，当前页面由独立前端入口驱动，后端只负责 API 和静态分发。</p>
          </div>
          <div class="masthead__controls">
            <label class="theme-switcher" for="theme-select">
              <span class="theme-switcher__label">主题</span>
              <select id="theme-select" data-theme-select>
                ${renderThemeOptions()}
              </select>
            </label>
            <div class="masthead__status">
              <div class="status-chip">
                <span class="status-chip__label">当前节点</span>
                <strong>${escapeHTML(view.localNodeID || "-")}</strong>
              </div>
              <div class="status-chip">
                <span class="status-chip__label">Leader</span>
                <strong>${escapeHTML(view.leaderID || "选举中")}</strong>
              </div>
              <div class="status-chip">
                <span class="status-chip__label">最近刷新</span>
                <strong>${escapeHTML(formatDateTime(view.generatedAt))}</strong>
              </div>
            </div>
          </div>
        </header>
        <div class="navband">
          <nav class="navstrip">
            <a href="/" data-link class="${view.page === "overview" ? "is-active" : ""}">总览</a>
            <a href="/events" data-link class="${view.page === "events" ? "is-active" : ""}">事件</a>
            ${view.page === "node" ? '<span class="navstrip__ghost">节点详情</span>' : ""}
          </nav>
          <p class="navband__context">${escapeHTML(view.title)}</p>
        </div>
        ${view.content}
      </div>
    `;
  }

  function renderThemeOptions() {
    const selectedTheme = getCurrentTheme();
    return THEMES.map((theme) => {
      const selected = theme.id === selectedTheme ? " selected" : "";
      return `<option value="${theme.id}"${selected}>${theme.label}</option>`;
    }).join("");
  }

  function renderStatePanel(title, description, route) {
    return `
      <main class="${routeMainClass(route)}">
        <section class="panel">
          <div class="empty-rune">
            <strong>${escapeHTML(title)}</strong>
            <p>${escapeHTML(description)}</p>
          </div>
        </section>
      </main>
    `;
  }

  function renderOverview(snapshot, historyMap) {
    const nodes = Array.isArray(snapshot.nodes) ? snapshot.nodes : [];
    const incidents = Array.isArray(snapshot.incidents) ? snapshot.incidents : [];
    const events = Array.isArray(snapshot.events) ? snapshot.events : [];
    const counts = summarizeNodes(nodes);

    return `
      <main class="overview">
        <section class="command-deck">
          <div class="command-deck__copy">
            <p class="eyebrow">总览</p>
            <h2>${snapshot.ingress && snapshot.ingress.active_node_id ? "入口当前指向 " + escapeHTML(snapshot.ingress.active_node_id) : "入口正在等待新的 active node"}</h2>
            <p class="command-deck__lede">这个页面完全由前端自己消费 API 后拼出来。先看入口与 DNS，再看节点健康、活跃 incident 和互探矩阵。</p>
          </div>
          <div class="command-deck__stats">
            ${renderSummaryCard("Ingress 节点", snapshot.ingress && snapshot.ingress.active_node_id ? snapshot.ingress.active_node_id : "待选举", "当前对外流量落点")}
            ${renderSummaryCard("DNS 同步", snapshot.ingress && snapshot.ingress.dns_synced ? "已同步" : "待同步", snapshot.ingress && snapshot.ingress.dns_synced_at ? timeAgo(snapshot.ingress.dns_synced_at) : "尚未同步")}
            ${renderSummaryCard("活跃 Incident", String(incidents.length), "当前需要处理的异常")}
            ${renderSummaryCard("Critical 节点", String(counts.critical), counts.healthy + " 个节点处于稳定状态")}
            ${renderSummaryCard("Ingress IP", snapshot.ingress && snapshot.ingress.desired_ip ? snapshot.ingress.desired_ip : "-", "期望的对外地址")}
            ${renderSummaryCard("最近事件", String(events.length), "这一轮同步写入的轨迹")}
          </div>
        </section>

        <section class="overview-grid">
          <div class="overview-grid__main">
            <section class="panel stage-panel">
              <div class="section-heading">
                <div>
                  <p class="eyebrow">节点主舞台</p>
                  <h3>节点健康主舞台</h3>
                </div>
                <p>节点卡片由前端直接根据 <code>/api/v1/cluster</code> 与 <code>/api/v1/history</code> 组合而成，后端不再渲染任何 HTML。</p>
              </div>
              <div class="node-stage">
                ${nodes.map((node) => renderNodeCard(node, historyMap[node.node_id] || [])).join("")}
              </div>
            </section>

            <section class="panel matrix-panel">
              <div class="section-heading">
                <div>
                  <p class="eyebrow">节点互探矩阵</p>
                  <h3>节点互探矩阵</h3>
                </div>
                <p>这里继续用当前 API 可推导的数据回答谁能被看见、谁已经开始变薄或完全掉线。</p>
              </div>
              <div class="matrix">
                ${renderProbeMatrix(nodes)}
              </div>
            </section>
          </div>

          <aside class="overview-grid__rail">
            <section class="panel rail-panel">
              <div class="section-heading">
                <div>
                  <p class="eyebrow">活跃 Incident</p>
                  <h3>现在最该处理的异常</h3>
                </div>
                <p>只展示当前仍打开的 incident，避免优先级被普通事件冲淡。</p>
              </div>
              <div class="stack-list">
                ${renderIncidentTimeline(incidents, {
                  emptyTitle: "当前没有活动 incident",
                  emptyText: "这是安静时段，不是空白时段。页面会继续等待下一次异常。",
                  showNode: true
                })}
              </div>
            </section>

            <section class="panel rail-panel">
              <div class="section-heading">
                <div>
                  <p class="eyebrow">最近事件</p>
                  <h3>最近状态轨迹</h3>
                </div>
                <p>用于复盘 leader、DNS 与状态变化是如何串起来发生的。</p>
              </div>
              <div class="stack-list">
                ${renderEventTimeline(events, {
                  emptyTitle: "还没有事件",
                  emptyText: "第一轮 leader 评估完成后，状态流会从这里开始堆起来。",
                  compact: true
                })}
              </div>
            </section>
          </aside>
        </section>
      </main>
    `;
  }

  function renderNodePage(nodeID, detail, snapshot, memHistory, diskHistory) {
    const state = detail.state || {};
    const cpuHistory = Array.isArray(detail.history) ? detail.history : [];
    const services = Array.isArray(state.services) ? state.services : [];
    const probes = Array.isArray(detail.probes) ? detail.probes : [];
    const incidents = Array.isArray(detail.incidents) ? detail.incidents : [];
    const serviceIssues = countServiceIssues(services);
    const ingressRole = snapshot.ingress && snapshot.ingress.active_node_id === nodeID ? "ACTIVE" : "STANDBY";

    return `
      <main class="node-page">
        <section class="node-command status-surface" data-status="${normalizeStatus(state.status)}">
          <div class="node-command__copy">
            <p class="eyebrow">节点档案</p>
            <h2>${escapeHTML(nodeID)} · ${escapeHTML(statusLabel(state.status))}</h2>
            <p>${escapeHTML(state.reason || "无说明")}</p>
          </div>
          <div class="node-command__facts">
            ${renderSummaryCard("Heartbeat", timeAgo(state.last_heartbeat_at), "最后一次心跳")}
            ${renderSummaryCard("Leader", snapshot.leader_id || "选举中", "当前决策节点")}
            ${renderSummaryCard("Ingress", ingressRole, "当前入口角色")}
            ${renderSummaryCard("Incident 记录", String(incidents.length), "近期历史长度")}
          </div>
        </section>

        <section class="node-grid">
          <article class="panel panel--evidence">
            <div class="section-heading">
              <div>
                <p class="eyebrow">判定证据</p>
                <h3>为什么它会被判成现在这样</h3>
              </div>
              <p>现在这些证据完全由前端根据 JSON 数据渲染，而不是后端模板拼接。</p>
            </div>
            <div class="evidence-list">
              ${renderEvidenceList(state.primary_evidence)}
            </div>
            <details class="raw-toggle">
              <summary>展开原始观测证据</summary>
              <pre>${escapeHTML(JSON.stringify(probes, null, 2))}</pre>
            </details>
          </article>

          <article class="panel panel--metrics">
            <div class="section-heading">
              <div>
                <p class="eyebrow">资源曲线</p>
                <h3>资源曲线与当前占用</h3>
              </div>
              <p>CPU 直接使用节点详情里的历史，内存与磁盘则通过 <code>/api/v1/history</code> 补齐。</p>
            </div>
            <div class="metric-stack">
              ${renderMetricBar("CPU", state.cpu_pct, cpuHistory)}
              ${renderMetricBar("MEM", state.mem_pct, memHistory)}
              ${renderMetricBar("DISK", state.disk_pct, diskHistory)}
            </div>
          </article>

          <article class="panel">
            <div class="section-heading">
              <div>
                <p class="eyebrow">服务</p>
                <h3>关键服务面</h3>
              </div>
              <p>${serviceIssues} 项服务处于异常或不稳定状态。</p>
            </div>
            <div class="service-list">
              ${renderServiceList(services)}
            </div>
          </article>

          <article class="panel">
            <div class="section-heading">
              <div>
                <p class="eyebrow">互探轨迹</p>
                <h3>互探路径证据</h3>
              </div>
              <p>最近 ${probes.length} 条观测，直接用来判断从哪一层开始失真。</p>
            </div>
            <div class="probe-trail">
              ${renderProbeList(probes)}
            </div>
          </article>

          <article class="panel panel--wide">
            <div class="section-heading">
              <div>
                <p class="eyebrow">Incident 历史</p>
                <h3>这个节点最近发生过什么</h3>
              </div>
              <p>打开、恢复与残留模式被放在一条线上，便于判断它是偶发还是反复。</p>
            </div>
            <div class="timeline">
              ${renderIncidentTimeline(incidents, {
                emptyTitle: "这个节点还没有 incident 历史",
                emptyText: "如果你保持当前平稳，这块应该继续空着。",
                showNode: false
              })}
            </div>
          </article>
        </section>
      </main>
    `;
  }

  function renderEventsPage(snapshot, incidents, events, meta) {
    const incidentList = Array.isArray(incidents) ? incidents : [];
    const eventList = Array.isArray(events) ? events : [];
    const activeIncidentCount = incidentList.filter((incident) => incident.status === "active").length;

    return `
      <main class="events-page">
        <section class="events-command">
          <div class="events-command__copy">
            <p class="eyebrow">事件时间线</p>
            <h2>从事件流看集群如何做决定</h2>
            <p>这个页现在完全由独立前端入口消费 <code>/api/v1/incidents</code>、<code>/api/v1/events</code> 与轻量元信息 API 渲染。</p>
          </div>
          <div class="events-command__stats">
            ${renderSummaryCard("Ingress", snapshot.ingress && snapshot.ingress.active_node_id ? snapshot.ingress.active_node_id : "待选举", "当前入口落点")}
            ${renderSummaryCard("DNS", snapshot.ingress && snapshot.ingress.dns_synced ? "已同步" : "待同步", "域名回源状态")}
            ${renderSummaryCard("活跃 Incident", String(activeIncidentCount), "当前仍在打开的异常")}
            ${renderSummaryCard("最近事件", String(eventList.length), "最近一次采样保留下来的轨迹")}
          </div>
        </section>

        <section class="events-layout">
          <article class="panel panel--compose">
            <div class="section-heading">
              <div>
                <p class="eyebrow">告警测试</p>
                <h3>手动打一次告警</h3>
              </div>
              <p>告警能力信息来自新的 <code>/api/v1/meta</code>，后端不再把它注入模板。</p>
            </div>
            ${renderTestAlertPanel(meta)}
          </article>

          <div class="events-stack">
            <article class="panel">
              <div class="section-heading">
                <div>
                  <p class="eyebrow">Incident 轨迹</p>
                  <h3>问题如何打开与恢复</h3>
                </div>
                <p>这条线只看 incident 的演化，不让普通事件把优先级稀释掉。</p>
              </div>
              <div class="timeline">
                ${renderIncidentTimeline(incidentList, {
                  emptyTitle: "还没有 incident 历史",
                  emptyText: "这代表系统刚启动，或者最近确实很稳。",
                  showNode: true
                })}
              </div>
            </article>

            <article class="panel">
              <div class="section-heading">
                <div>
                  <p class="eyebrow">集群事件</p>
                  <h3>Leader、DNS 与状态切换</h3>
                </div>
                <p>这一段用来复盘状态变化是如何发生的，而不是把所有日志都塞进来。</p>
              </div>
              <div class="timeline">
                ${renderEventTimeline(eventList, {
                  emptyTitle: "事件流暂时空着",
                  emptyText: "它会在 leader 评估、DNS 切换和 incident 打开时开始积累。",
                  compact: false
                })}
              </div>
            </article>
          </div>
        </section>
      </main>
    `;
  }

  function renderNodeCard(node, history) {
    const summary = node.last_probe_summary || {};
    const loadLabel = Number.isFinite(Number(node.load1)) ? Number(node.load1).toFixed(2) : "-";

    return `
      <article class="node-card status-surface" data-status="${normalizeStatus(node.status)}">
        <div class="node-card__header">
          <div>
            <p class="eyebrow">Node ${escapeHTML(node.node_id || "-")}</p>
            <h3>${escapeHTML(node.node_id || "-")}</h3>
          </div>
          <div class="node-card__actions">
            <span class="status-pill">${escapeHTML(statusLabel(node.status))}</span>
            <a href="/nodes/${encodeURIComponent(node.node_id || "")}" data-link class="node-card__link">进入节点</a>
          </div>
        </div>
        <p class="node-card__reason">${escapeHTML(node.reason || "无说明")}</p>
        <div class="node-card__metrics">
          ${renderMiniMetric("CPU", formatPercent(node.cpu_pct))}
          ${renderMiniMetric("MEM", formatPercent(node.mem_pct))}
          ${renderMiniMetric("DISK", formatPercent(node.disk_pct))}
          ${renderMiniMetric("Heartbeat", timeAgo(node.last_heartbeat_at))}
        </div>
        <div class="trendline trendline--node">${sparkline(history)}</div>
        <div class="node-card__footer">
          <span>${countServiceIssues(node.services)} 项服务异常</span>
          <span>${(summary.successful_peers || 0)} / ${(summary.total_peers || 0)} 个节点确认</span>
          <span>Load ${escapeHTML(loadLabel)}</span>
        </div>
      </article>
    `;
  }

  function renderProbeMatrix(nodes) {
    if (!Array.isArray(nodes) || nodes.length === 0) {
      return emptyRune("暂无节点数据", "等第一轮集群状态写入后，这里会出现互探矩阵。");
    }

    return nodes
      .map((source) => {
        const cells = nodes
          .map((target) => {
            if (source.node_id === target.node_id) {
              return `
                <div class="matrix__cell" data-status="self">
                  <span>${escapeHTML(target.node_id || "-")}</span>
                  <strong>SELF</strong>
                </div>
              `;
            }

            let label = "WAIT";
            let status = "unknown";
            if (target.last_probe_summary && target.last_probe_summary.reachable) {
              label = "OPEN";
              status = "healthy";
            } else if (normalizeStatus(target.status) === "critical") {
              label = "DROP";
              status = "critical";
            } else if (normalizeStatus(target.status) === "degraded") {
              label = "THIN";
              status = "degraded";
            }

            return `
              <div class="matrix__cell" data-status="${status}">
                <span>${escapeHTML(target.node_id || "-")}</span>
                <strong>${label}</strong>
              </div>
            `;
          })
          .join("");

        return `
          <div class="matrix__row">
            <div class="matrix__label">${escapeHTML(source.node_id || "-")}</div>
            ${cells}
          </div>
        `;
      })
      .join("");
  }

  function renderIncidentTimeline(incidents, options) {
    const list = Array.isArray(incidents) ? incidents : [];
    if (list.length === 0) {
      return emptyRune(options.emptyTitle, options.emptyText);
    }

    return list
      .map((incident) => {
        const eyebrow = options.showNode
          ? (incident.node_id || "-") + " · " + (incident.rule_key || "incident")
          : incident.rule_key || "incident";
        return renderTimelineItem({
          status: incident.severity || incident.status,
          eyebrow: eyebrow,
          title: incident.summary || "未命名 incident",
          description: "",
          rightTop: incident.status || "",
          rightBottom: formatDateTime(incident.opened_at)
        });
      })
      .join("");
  }

  function renderEventTimeline(events, options) {
    const list = Array.isArray(events) ? events : [];
    if (list.length === 0) {
      return emptyRune(options.emptyTitle, options.emptyText);
    }

    return list
      .map((event) =>
        renderTimelineItem({
          status: event.severity,
          eyebrow: event.kind || "event",
          title: event.title || "未命名事件",
          description: options.compact ? truncate(event.body || "", 88) : "",
          rightTop: event.node_id || "-",
          rightBottom: options.compact ? timeAgo(event.created_at) : formatDateTime(event.created_at)
        })
      )
      .join("");
  }

  function renderMetricBar(label, value, history) {
    return `
      <div class="metric-bar">
        <div class="metric-bar__header">
          <span>${escapeHTML(label)}</span>
          <strong>${escapeHTML(formatPercent(value))}</strong>
        </div>
        <div class="meter"><i style="width: ${escapeHTML(formatPercent(value))}"></i></div>
        <div class="trendline">${sparkline(history)}</div>
      </div>
    `;
  }

  function renderServiceList(services) {
    const list = Array.isArray(services) ? services : [];
    if (list.length === 0) {
      return emptyRune("没有配置服务检查", "在 monitor.yaml 的 checks.services 里加上 systemd 服务名。");
    }

    return list
      .map(
        (service) => `
          <div class="service-row status-surface" data-status="${normalizeStatus(service.status)}">
            <strong>${escapeHTML(service.name || "service")}</strong>
            <span>${escapeHTML(service.status || "unknown")}</span>
            <small>${escapeHTML(service.detail || "")}</small>
          </div>
        `
      )
      .join("");
  }

  function renderProbeList(probes) {
    const list = Array.isArray(probes) ? probes : [];
    if (list.length === 0) {
      return emptyRune("还没有探测数据", "下一轮 prober 跑完，这里会出现路径证据。");
    }

    return list
      .map(
        (probe) => `
          <div class="probe-row status-surface" data-status="${probeTone(probe)}">
            <strong>${escapeHTML((probe.source_node_id || "?") + " → " + (probe.target_node_id || "?"))}</strong>
            <span>22 ${probe.tcp_22_ok ? "OPEN" : "DROP"}</span>
            <span>443 ${probe.tcp_443_ok ? "OPEN" : "DROP"}</span>
            <span>HTTP ${probe.http_ok ? "OK" : "MISS"}</span>
            <small>${escapeHTML(timeAgo(probe.collected_at))}</small>
          </div>
        `
      )
      .join("");
  }

  function renderEvidenceList(items) {
    const list = Array.isArray(items) ? items : [];
    if (list.length === 0) {
      return '<p class="evidence-item">等待第一轮评估产出规则命中信息。</p>';
    }
    return list.map((item) => '<p class="evidence-item">' + escapeHTML(item) + "</p>").join("");
  }

  function renderTestAlertPanel(meta) {
    const channels = Array.isArray(meta.test_alert_channels) ? meta.test_alert_channels : [];
    const requiresToken = Boolean(meta.test_alert_requires_token);

    if (channels.length === 0) {
      return emptyRune("没有启用任何告警渠道", "先在 monitor.yaml 里开启 Telegram、SMTP 或 webhook 之后，这里才会出现测试入口。");
    }

    return `
      <form class="test-alert-form" id="test-alert-form">
        <label>
          <span>渠道</span>
          <select name="channel">
            <option value="all">all</option>
            ${channels.map((channel) => `<option value="${escapeHTML(channel)}">${escapeHTML(channel)}</option>`).join("")}
          </select>
        </label>
        ${requiresToken ? `
          <label>
            <span>Token</span>
            <input type="password" name="token" placeholder="MONITOR_TEST_ALERT_TOKEN">
          </label>
        ` : ""}
        <label class="test-alert-form__wide">
          <span>备注</span>
          <textarea name="note" rows="4" placeholder="这条消息用于验证 Telegram / SMTP 是否真的能送达。"></textarea>
        </label>
        <button type="submit" class="test-alert-form__submit">发送测试告警</button>
      </form>
      <div class="test-alert-result" id="test-alert-result">
        <p>这里只会发送测试消息，不会创建真实 incident。</p>
      </div>
    `;
  }

  function renderTimelineItem(options) {
    const description = options.description
      ? `<p>${escapeHTML(options.description)}</p>`
      : "";
    const classNames = options.status ? "timeline-item status-surface" : "timeline-item";
    const statusAttr = options.status ? ` data-status="${normalizeStatus(options.status)}"` : "";

    return `
      <article class="${classNames}"${statusAttr}>
        <div>
          <p>${escapeHTML(options.eyebrow || "")}</p>
          <strong>${escapeHTML(options.title || "")}</strong>
          ${description}
        </div>
        <div>
          <span>${escapeHTML(options.rightTop || "")}</span>
          <small>${escapeHTML(options.rightBottom || "")}</small>
        </div>
      </article>
    `;
  }

  function renderSummaryCard(label, value, description) {
    return `
      <article class="summary-card">
        <span>${escapeHTML(label)}</span>
        <strong>${escapeHTML(value)}</strong>
        <small>${escapeHTML(description)}</small>
      </article>
    `;
  }

  function renderMiniMetric(label, value) {
    return `
      <div class="mini-metric">
        <span>${escapeHTML(label)}</span>
        <strong>${escapeHTML(value)}</strong>
      </div>
    `;
  }

  async function submitTestAlert(form) {
    const result = document.getElementById("test-alert-result");
    if (result) {
      result.innerHTML = "<p>正在发送测试告警...</p>";
    }

    const payload = {
      channel: form.channel ? form.channel.value : "all",
      token: form.token ? form.token.value : "",
      note: form.note ? form.note.value : ""
    };

    const response = await fetch("/api/v1/test-alert", {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        Accept: "application/json"
      },
      body: JSON.stringify(payload)
    });

    const data = await response.json();
    if (!response.ok) {
      throw new Error(data.error || ("request failed: " + response.status));
    }

    if (!result) {
      return;
    }

    const items = Array.isArray(data.results) ? data.results : [];
    result.innerHTML = `
      <p>发送时间 ${escapeHTML(formatDateTime(data.sent_at))}</p>
      ${items
        .map(
          (item) => `
            <div class="test-alert-result__item status-surface" data-status="${item.ok ? "healthy" : "critical"}">
              <strong>${escapeHTML(item.channel || "channel")}</strong>
              <span>${escapeHTML(item.ok ? "OK" : item.error || "FAILED")}</span>
            </div>
          `
        )
        .join("")}
    `;
  }

  async function fetchJSON(url) {
    const response = await fetch(url, {
      headers: {
        Accept: "application/json"
      }
    });
    if (!response.ok) {
      throw new Error("request failed: " + response.status);
    }
    return response.json();
  }

  async function fetchHistory(nodeID, metric) {
    const query = new URLSearchParams({
      node_id: nodeID,
      metric: metric
    });
    return fetchJSON("/api/v1/history?" + query.toString());
  }

  function syncThemeSelect() {
    const select = document.querySelector("[data-theme-select]");
    if (select) {
      select.value = getCurrentTheme();
    }
  }

  function applyTheme(theme) {
    const nextTheme = THEMES.some((item) => item.id === theme) ? theme : "graphite";
    document.documentElement.dataset.theme = nextTheme;
    try {
      window.localStorage.setItem(THEME_KEY, nextTheme);
    } catch (error) {
      // Ignore storage failures.
    }
    return nextTheme;
  }

  function getCurrentTheme() {
    return document.documentElement.dataset.theme || "graphite";
  }

  function summarizeNodes(nodes) {
    return (Array.isArray(nodes) ? nodes : []).reduce(
      (counts, node) => {
        counts[normalizeStatus(node.status)] += 1;
        return counts;
      },
      { healthy: 0, degraded: 0, critical: 0, unknown: 0 }
    );
  }

  function countServiceIssues(services) {
    return (Array.isArray(services) ? services : []).filter((service) => normalizeStatus(service.status) !== "healthy").length;
  }

  function normalizeStatus(status) {
    const value = String(status || "").trim().toLowerCase();
    if (["healthy", "running", "active", "ok", "up", "synced"].includes(value)) {
      return "healthy";
    }
    if (["degraded", "warning", "partial", "starting", "restarting"].includes(value)) {
      return "degraded";
    }
    if (["critical", "failed", "fail", "inactive", "down", "error", "unhealthy", "dead", "exited", "stopped", "drop", "miss"].includes(value)) {
      return "critical";
    }
    return "unknown";
  }

  function statusLabel(status) {
    switch (normalizeStatus(status)) {
      case "healthy":
        return "稳定";
      case "degraded":
        return "降级";
      case "critical":
        return "严重";
      default:
        return "未知";
    }
  }

  function probeTone(probe) {
    if (probe && probe.tcp_22_ok && probe.tcp_443_ok && probe.http_ok) {
      return "healthy";
    }
    if (probe && (probe.tcp_22_ok || probe.tcp_443_ok || probe.http_ok)) {
      return "degraded";
    }
    return "critical";
  }

  function formatPercent(value) {
    const parsed = Number(value);
    if (!Number.isFinite(parsed)) {
      return "-";
    }
    return Math.round(parsed) + "%";
  }

  function formatDateTime(value) {
    if (!value) {
      return "从未";
    }
    const date = new Date(value);
    if (Number.isNaN(date.getTime())) {
      return "从未";
    }
    return date.toLocaleString("zh-CN", { hour12: false });
  }

  function timeAgo(value) {
    if (!value) {
      return "无信号";
    }
    const date = new Date(value);
    if (Number.isNaN(date.getTime())) {
      return "无信号";
    }
    let diff = Math.abs(Date.now() - date.getTime());
    if (diff < 60000) {
      return Math.max(1, Math.round(diff / 1000)) + " 秒前";
    }
    if (diff < 3600000) {
      return Math.max(1, Math.round(diff / 60000)) + " 分钟前";
    }
    return Math.max(1, Math.round(diff / 3600000)) + " 小时前";
  }

  function truncate(value, limit) {
    const input = String(value || "");
    if (input.length <= limit) {
      return input;
    }
    return input.slice(0, limit) + "...";
  }

  function sparkline(points) {
    const list = Array.isArray(points) ? points : [];
    if (list.length === 0) {
      return '<svg viewBox="0 0 100 24" preserveAspectRatio="none"><path d="M0 20 L100 20"></path></svg>';
    }

    let maxValue = 0;
    for (const point of list) {
      maxValue = Math.max(maxValue, Number(point.value) || 0);
    }
    if (maxValue === 0) {
      maxValue = 1;
    }

    const path = list
      .map((point, index) => {
        const x = (index / Math.max(1, list.length - 1)) * 100;
        const y = 22 - ((Number(point.value) || 0) / maxValue) * 20;
        return (index === 0 ? "M" : "L") + x.toFixed(2) + " " + y.toFixed(2);
      })
      .join(" ");

    return `<svg viewBox="0 0 100 24" preserveAspectRatio="none"><path d="${path}"></path></svg>`;
  }

  function emptyRune(title, description) {
    return `
      <div class="empty-rune">
        <strong>${escapeHTML(title)}</strong>
        <p>${escapeHTML(description)}</p>
      </div>
    `;
  }

  function escapeHTML(value) {
    return String(value == null ? "" : value)
      .replace(/&/g, "&amp;")
      .replace(/</g, "&lt;")
      .replace(/>/g, "&gt;")
      .replace(/"/g, "&quot;")
      .replace(/'/g, "&#39;");
  }
})();
