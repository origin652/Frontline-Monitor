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
  let lastRouteKey = "";
  let currentSearchItems = [];
  let currentSearchQuery = "";
  let searchPanelOpen = false;

  document.addEventListener("click", handleDocumentClick);
  document.addEventListener("change", handleDocumentChange);
  document.addEventListener("input", handleDocumentInput);
  document.addEventListener("focusin", handleDocumentFocusIn);
  document.addEventListener("keydown", handleDocumentKeydown);
  document.addEventListener("submit", handleDocumentSubmit);
  window.addEventListener("popstate", () => {
    renderRoute();
  });

  applyTheme(document.documentElement.dataset.theme || document.body.dataset.defaultTheme || "graphite");
  renderRoute();

  function handleDocumentClick(event) {
    const searchRoot = event.target.closest("[data-search-root]");
    if (!searchRoot) {
      closeSearchPanel();
    }

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

  function handleDocumentInput(event) {
    if (!event.target.matches("[data-global-search]")) {
      return;
    }
    currentSearchQuery = event.target.value || "";
    searchPanelOpen = true;
    syncSearchUI();
  }

  function handleDocumentFocusIn(event) {
    if (!event.target.matches("[data-global-search]")) {
      return;
    }
    searchPanelOpen = true;
    syncSearchUI();
  }

  function handleDocumentKeydown(event) {
    if ((event.ctrlKey || event.metaKey) && !event.shiftKey && String(event.key).toLowerCase() === "k") {
      event.preventDefault();
      focusSearchInput();
      return;
    }

    if (!event.target.matches("[data-global-search]")) {
      return;
    }

    if (event.key === "Escape") {
      event.preventDefault();
      closeSearchPanel();
      event.target.blur();
      return;
    }

    if (event.key === "Enter") {
      const first = filterSearchItems(currentSearchItems, currentSearchQuery)[0];
      if (!first) {
        return;
      }
      event.preventDefault();
      navigateTo(first.href);
    }
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

  async function renderRoute(options) {
    const settings = options || {};
    window.clearTimeout(refreshTimer);
    const token = ++renderToken;
    const route = resolveRoute(window.location.pathname);
    const routeKey = buildRouteKey(route);
    const backgroundRefresh = Boolean(settings.backgroundRefresh);
    const shouldShowLoading = !backgroundRefresh && (!app.innerHTML || routeKey !== lastRouteKey);
    document.body.dataset.page = route.page;

    if (shouldShowLoading) {
      currentSearchItems = [];
      currentSearchQuery = "";
      searchPanelOpen = false;
      app.innerHTML = renderShell({
        page: route.page,
        title: routeTitle(route),
        localNodeID: "...",
        leaderID: "加载中",
        generatedAt: new Date().toISOString(),
        content: renderStatePanel("加载中", "正在从 API 拉取最新集群状态。", route)
      });
      syncThemeSelect();
    }

    try {
      const view = await loadRoute(route);
      if (token !== renderToken) {
        return;
      }

      document.title = view.title + " · VPS Monitor";
      currentSearchItems = Array.isArray(view.searchIndex) ? view.searchIndex : [];
      if (!backgroundRefresh) {
        currentSearchQuery = "";
        searchPanelOpen = false;
      }
      app.innerHTML = renderShell(view);
      lastRouteKey = routeKey;
      syncThemeSelect();
      syncSearchUI();
      scheduleRefresh();
    } catch (error) {
      if (token !== renderToken) {
        return;
      }

      if (backgroundRefresh && app.innerHTML && routeKey === lastRouteKey) {
        scheduleRefresh();
        return;
      }

      document.title = routeTitle(route) + " · VPS Monitor";
      currentSearchItems = [];
      currentSearchQuery = "";
      searchPanelOpen = false;
      app.innerHTML = renderShell({
        page: route.page,
        title: routeTitle(route),
        localNodeID: "-",
        leaderID: "不可用",
        generatedAt: new Date().toISOString(),
        content: renderStatePanel("加载失败", error.message || "接口请求失败。", route)
      });
      lastRouteKey = routeKey;
      syncThemeSelect();
      syncSearchUI();
      scheduleRefresh();
    }
  }

  function scheduleRefresh() {
    window.clearTimeout(refreshTimer);
    refreshTimer = window.setTimeout(() => {
      if (document.activeElement && document.activeElement.matches("[data-global-search]")) {
        scheduleRefresh();
        return;
      }

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
      renderRoute({ backgroundRefresh: true });
    }, REFRESH_INTERVAL_MS);
  }

  function buildRouteKey(route) {
    if (!route || !route.page) {
      return "unknown";
    }
    return route.page + ":" + (route.nodeID || "");
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
      currentNodeID: snapshot.node_id || "",
      leaderID: snapshot.leader_id || "",
      generatedAt: snapshot.generated_at || new Date().toISOString(),
      content: renderOverview(snapshot, historyMap),
      searchIndex: buildSearchIndex({
        snapshot: snapshot,
        incidents: snapshot.incidents,
        events: snapshot.events
      })
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
      currentNodeID: nodeID,
      leaderID: snapshot.leader_id || "",
      generatedAt: snapshot.generated_at || new Date().toISOString(),
      content: renderNodePage(nodeID, detail, snapshot, memHistory, diskHistory),
      searchIndex: buildSearchIndex({
        snapshot: snapshot,
        incidents: detail.incidents,
        events: snapshot.events,
        currentNodeID: nodeID
      })
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
      currentNodeID: snapshot.node_id || meta.node_id || "",
      leaderID: snapshot.leader_id || meta.leader_id || "",
      generatedAt: snapshot.generated_at || new Date().toISOString(),
      content: renderEventsPage(snapshot, incidents, events, meta),
      searchIndex: buildSearchIndex({
        snapshot: snapshot,
        incidents: incidents,
        events: events
      })
    };
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

  function renderShell(view) {
    const localNodeHref = isUsableNodeID(view.localNodeID)
      ? "/nodes/" + encodeURIComponent(view.localNodeID)
      : "";
    const contextNodeID = view.currentNodeID || "";
    const contextNodeHref = isUsableNodeID(contextNodeID)
      ? "/nodes/" + encodeURIComponent(contextNodeID)
      : localNodeHref;
    const headingLabel = view.page === "node"
      ? "Node Detail"
      : view.page === "events"
        ? "Incident Center"
        : "Command Center";
    const headingTitle = view.page === "node" && contextNodeID
      ? contextNodeID
      : sidebarContextLabel(view.page);

    return `
      <div class="obs-shell">
        <aside class="obs-sidebar">
          <div class="obs-sidebar__brand">
            <div>
              <h1>Obsidian</h1>
              <p>vps-monitor</p>
            </div>
            <span class="obs-sidebar__context">${escapeHTML(sidebarContextLabel(view.page))}</span>
          </div>

          <nav class="obs-sidebar__nav">
            ${renderObsidianNavLink("/", "dashboard", "Observatory", view.page === "overview")}
            ${localNodeHref ? renderObsidianNavLink(localNodeHref, "dns", "Local Node", view.page === "node" && contextNodeID === view.localNodeID) : ""}
            ${contextNodeHref && contextNodeID && contextNodeID !== view.localNodeID
              ? renderObsidianNavLink(contextNodeHref, "lan", "Node " + contextNodeID, view.page === "node")
              : ""}
            ${renderObsidianNavLink("/events", "warning", "Incidents", view.page === "events")}
          </nav>

          <div class="obs-sidebar__footer">
            ${localNodeHref
              ? `<a href="${localNodeHref}" data-link class="obs-sidebar__cta">${renderIcon("rocket_launch")}<span>Open Local Node</span></a>`
              : ""}
            <div class="obs-sidebar__status">
              <div class="obs-sidebar__status-item">
                <span>Current Node</span>
                <strong>${escapeHTML(view.localNodeID || "-")}</strong>
              </div>
              <div class="obs-sidebar__status-item">
                <span>Leader</span>
                <strong>${escapeHTML(view.leaderID || "Electing")}</strong>
              </div>
            </div>
            <div class="obs-sidebar__links">
              <span>Self-hosted control console</span>
              <span>${escapeHTML(formatDateTime(view.generatedAt))}</span>
            </div>
          </div>
        </aside>

        <header class="obs-topbar">
          <div class="obs-topbar__heading">
            <span>${escapeHTML(headingLabel)}</span>
            <strong>${escapeHTML(headingTitle)}</strong>
          </div>
          <div class="obs-topbar__search${searchPanelOpen ? " is-open" : ""}" data-search-root>
            <label class="obs-search-field" for="global-search">
              ${renderIcon("search")}
              <input
                id="global-search"
                type="search"
                value="${escapeHTML(currentSearchQuery)}"
                placeholder="Search nodes, incidents, or telemetry..."
                autocomplete="off"
                spellcheck="false"
                data-global-search
              >
            </label>
            <div class="obs-search-panel" data-search-panel>
              ${renderSearchPanel(currentSearchItems, currentSearchQuery)}
            </div>
          </div>
          <div class="obs-topbar__actions">
            ${renderObsidianMeta("Updated", formatDateTime(view.generatedAt))}
            <label class="obs-theme-switcher" for="theme-select">
              ${renderIcon("palette")}
              <select id="theme-select" data-theme-select>
                ${renderThemeOptions()}
              </select>
            </label>
            <div class="obs-topbar__operator">
              <span>Cluster Leader</span>
              <strong>${escapeHTML(view.leaderID || "Electing")}</strong>
            </div>
          </div>
        </header>

        <div class="obs-main">
          ${view.content}
        </div>

        <nav class="obs-mobile-nav">
          ${renderObsidianMobileLink("/", "grid_view", "Dashboard", view.page === "overview")}
          ${contextNodeHref
            ? renderObsidianMobileLink(
                contextNodeHref,
                "dns",
                contextNodeID && contextNodeID !== view.localNodeID ? "Node" : "Local",
                view.page === "node"
              )
            : ""}
          ${renderObsidianMobileLink("/events", "warning", "Incidents", view.page === "events")}
        </nav>
      </div>
    `;
  }

  function renderSearchPanel(items, query) {
    const trimmed = String(query || "").trim();
    const quickItems = (Array.isArray(items) ? items : []).filter((item) => item.pinned).slice(0, 5);
    const results = filterSearchItems(items, trimmed).slice(0, 8);

    if (trimmed === "") {
      return `
        <div class="obs-search-empty">
          <strong>Quick Access</strong>
          <p>Type a node ID, incident summary, rule key, status, or event title.</p>
          <div class="obs-search-results">
            ${quickItems.map((item) => renderSearchResult(item)).join("")}
          </div>
        </div>
      `;
    }

    if (results.length === 0) {
      return `
        <div class="obs-search-empty">
          <strong>No Matches</strong>
          <p>No nodes, incidents, or events matched “${escapeHTML(trimmed)}”.</p>
        </div>
      `;
    }

    return `<div class="obs-search-results">${results.map((item) => renderSearchResult(item)).join("")}</div>`;
  }

  function renderSearchResult(item) {
    const statusAttr = item.status ? ` data-status="${escapeHTML(item.status)}"` : "";
    return `
      <a href="${escapeHTML(item.href)}" data-link class="obs-search-item"${statusAttr}>
        <div class="obs-search-item__icon">${renderIcon(item.icon || "search")}</div>
        <div class="obs-search-item__body">
          <div class="obs-search-item__head">
            <span>${escapeHTML(item.kind)}</span>
            <strong>${escapeHTML(item.title)}</strong>
          </div>
          <p>${escapeHTML(item.subtitle || "")}</p>
        </div>
      </a>
    `;
  }

  function filterSearchItems(items, query) {
    const needle = normalizeSearchText(query);
    if (!needle) {
      return [];
    }

    const tokens = needle.split(" ").filter(Boolean);
    return (Array.isArray(items) ? items : [])
      .map((item, index) => {
        const haystack = normalizeSearchText(
          [item.kind, item.title, item.subtitle, item.keywords]
            .filter(Boolean)
            .join(" ")
        );
        let score = 0;
        if (normalizeSearchText(item.title || "").startsWith(needle)) {
          score += 120;
        }
        if (haystack.includes(needle)) {
          score += 80;
        }
        for (const token of tokens) {
          if (haystack.includes(token)) {
            score += 20;
          } else {
            score -= 40;
          }
        }
        return { item: item, score: score, index: index };
      })
      .filter((entry) => entry.score > 0)
      .sort((left, right) => {
        if (right.score !== left.score) {
          return right.score - left.score;
        }
        return left.index - right.index;
      })
      .map((entry) => entry.item);
  }

  function syncSearchUI() {
    const input = document.querySelector("[data-global-search]");
    if (input && input.value !== currentSearchQuery) {
      input.value = currentSearchQuery;
    }

    const root = document.querySelector("[data-search-root]");
    if (root) {
      root.classList.toggle("is-open", searchPanelOpen);
    }

    const panel = document.querySelector("[data-search-panel]");
    if (panel) {
      panel.innerHTML = renderSearchPanel(currentSearchItems, currentSearchQuery);
    }
  }

  function closeSearchPanel() {
    if (!searchPanelOpen) {
      return;
    }
    searchPanelOpen = false;
    syncSearchUI();
  }

  function focusSearchInput() {
    const input = document.querySelector("[data-global-search]");
    if (!input) {
      return;
    }
    searchPanelOpen = true;
    input.focus();
    input.select();
    syncSearchUI();
  }

  function navigateTo(pathname) {
    if (!pathname || pathname === window.location.pathname) {
      searchPanelOpen = false;
      syncSearchUI();
      return;
    }
    window.history.pushState({}, "", pathname);
    currentSearchQuery = "";
    searchPanelOpen = false;
    renderRoute();
  }

  function renderStatePanel(title, description, route) {
    const pageTitle = route && route.page === "events"
      ? "Incident Center"
      : route && route.page === "node"
        ? "Node Detail"
        : "Observatory";

    return `
      <main class="obs-page">
        <section class="obs-state-panel">
          <p class="obs-state-panel__eyebrow">${escapeHTML(pageTitle)}</p>
          <h1>${escapeHTML(title)}</h1>
          <p>${escapeHTML(description)}</p>
        </section>
      </main>
    `;
  }

  function renderOverview(snapshot, historyMap) {
    const nodes = Array.isArray(snapshot.nodes) ? snapshot.nodes : [];
    const incidents = Array.isArray(snapshot.incidents) ? snapshot.incidents : [];
    const counts = summarizeNodes(nodes);
    const activeIncidents = activeIncidentCount(incidents);
    const summary = clusterSummary(snapshot, nodes, incidents, counts);
    const metrics = clusterAverages(nodes);
    const focusHistory = primaryHistory(snapshot, nodes, historyMap);
    const cpuPeak = highestNodeMetric(nodes, "cpu_pct");
    const memPeak = highestNodeMetric(nodes, "mem_pct");
    const diskPeak = highestNodeMetric(nodes, "disk_pct");

    return `
      <main class="obs-page obs-page--overview">
        <section class="obs-hero status-surface" data-status="${summary.tone}">
          <div class="obs-hero__copy">
            <p class="obs-kicker">
              <span class="obs-live-dot" data-status="${summary.tone}"></span>
              REAL-TIME PULSE
            </p>
            <h1 class="obs-hero__title">SYSTEM <span>${escapeHTML(summary.word)}</span></h1>
            <p class="obs-hero__text">${escapeHTML(summary.description)}</p>
          </div>
          <div class="obs-hero__aside">
            <div>
              <p class="obs-aside__label">Cluster Coverage</p>
              <strong class="obs-aside__value">${escapeHTML(clusterCoverageText(counts, nodes.length))}</strong>
            </div>
            <div class="obs-aside__spark">${sparkline(focusHistory, { area: true, tone: summary.tone })}</div>
            <div class="obs-hero__facts">
              <div class="obs-hero__fact">
                <span>Leader</span>
                <strong>${escapeHTML(snapshot.leader_id || "Electing")}</strong>
              </div>
              <div class="obs-hero__fact">
                <span>Active Incidents</span>
                <strong>${escapeHTML(String(activeIncidents))}</strong>
              </div>
            </div>
          </div>
        </section>

        <section class="obs-metric-grid">
          ${renderObsidianMetricCard({
            label: "CPU LOAD",
            value: formatPercent(metrics.cpu),
            caption: cpuPeak ? "Peak " + cpuPeak.nodeID + " · " + formatPercent(cpuPeak.value) : "Waiting for CPU history",
            icon: "memory",
            size: "wide",
            visual: sparkline(focusHistory, { area: true, tone: "healthy" })
          })}
          ${renderObsidianMetricCard({
            label: "MEMORY",
            value: formatPercent(metrics.mem),
            caption: memPeak ? "Peak " + memPeak.nodeID + " · " + formatPercent(memPeak.value) : "No memory sample yet",
            icon: "data_usage",
            visual: renderObsidianBarColumns(nodes, "mem_pct")
          })}
          ${renderObsidianMetricCard({
            label: "STORAGE",
            value: formatPercent(metrics.disk),
            caption: diskPeak ? "Peak " + diskPeak.nodeID + " · " + formatPercent(diskPeak.value) : "No storage sample yet",
            icon: "storage",
            tone: diskPeak && diskPeak.value >= 85 ? "critical" : "healthy",
            visual: renderObsidianMeter(metrics.disk, diskPeak)
          })}
        </section>

        <section class="obs-section obs-section--table">
          <div class="obs-section__head">
            <div>
              <p class="obs-section__eyebrow">Deployment Nodes</p>
              <h2>Active Deployment Nodes</h2>
            </div>
            <p>${escapeHTML(nodes.length + " nodes · " + counts.healthy + " stable · " + activeIncidents + " active incidents")}</p>
          </div>
          <div class="obs-node-table">
            <div class="obs-node-table__head">
              <span>Node Identity</span>
              <span>Status</span>
              <span>Peer Reach</span>
              <span>Heartbeat</span>
              <span>Uptime</span>
              <span>Action</span>
            </div>
            <div class="obs-node-table__body">
              ${nodes.length > 0
                ? nodes.map((node) => renderObsidianNodeRow(node, snapshot)).join("")
                : emptyRune("No nodes yet", "The table will populate after the first node reports telemetry.")}
            </div>
          </div>
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
    const tone = normalizeStatus(state.status);
    const visibility = nodeVisibilityPct(state);
    const uptimeCompact = formatUptimeCompact(state.uptime_s);

    return `
      <main class="obs-page obs-page--node">
        <section class="obs-node-top">
          <div class="obs-node-top__copy">
            <p class="obs-kicker">
              <span class="obs-live-dot" data-status="${tone}"></span>
              ${escapeHTML(nodeID)} · ${escapeHTML(nodeRoleLabel(snapshot, nodeID))}
            </p>
            <h1>${escapeHTML(nodeID)}</h1>
            <div class="obs-node-meta">
              <span>${renderIcon("monitor_heart")} ${escapeHTML(statusLabel(state.status))}</span>
              <span>${renderIcon("schedule")} Heartbeat ${escapeHTML(timeAgo(state.last_heartbeat_at))}</span>
              <span>${renderIcon("speed")} Load ${escapeHTML(formatLoad(state.load1))}</span>
            </div>
          </div>
          <nav class="obs-tabs">
            <a href="#node-overview" class="is-active">Overview</a>
            <a href="#node-resources">Resources</a>
            <a href="#node-stream">Logs</a>
          </nav>
        </section>

        <section id="node-overview" class="obs-node-hero status-surface" data-status="${tone}">
          <div>
            <p class="obs-section__eyebrow">Core Node Identifier</p>
            <h2>${escapeHTML(nodeID)}</h2>
            <p class="obs-node-hero__reason">${escapeHTML(state.reason || "No current reason provided.")}</p>
          </div>
          <div class="obs-node-hero__score">
            <span class="obs-node-hero__percent">${escapeHTML(String(visibility))}<small>%</small></span>
            <span class="obs-node-hero__pill">UPTIME ${escapeHTML(uptimeCompact)}</span>
          </div>
        </section>

        <section class="obs-node-grid">
          <article id="node-resources" class="obs-section obs-chart-panel">
            <div class="obs-section__head">
              <div>
                <p class="obs-section__eyebrow">Process Load</p>
                <h2>${escapeHTML(formatPercent(state.cpu_pct))}</h2>
              </div>
              <p>${escapeHTML(formatLoad(state.load1) + " load1")}</p>
            </div>
            <div class="obs-chart-panel__chart">
              ${sparkline(cpuHistory, { area: true, tone: tone })}
            </div>
            <div class="obs-chart-panel__axis">
              <span>-24h</span>
              <span>-12h</span>
              <span>Now</span>
            </div>
          </article>

          <article class="obs-section obs-memory-panel">
            <div class="obs-section__head">
              <div>
                <p class="obs-section__eyebrow">Resource Utilization</p>
                <h2>${escapeHTML(formatPercent(state.mem_pct))}</h2>
              </div>
              <p>${escapeHTML(countServiceIssues(services) + " service issues")}</p>
            </div>
            <div class="obs-resource-stack">
              ${renderMetricBar("MEMORY", state.mem_pct, memHistory)}
              ${renderMetricBar("DISK", state.disk_pct, diskHistory)}
              ${renderMetricBar("CPU", state.cpu_pct, cpuHistory)}
            </div>
          </article>
        </section>

        <section id="node-stream" class="obs-section obs-log-panel">
          <div class="obs-section__head">
            <div>
              <p class="obs-section__eyebrow">Probe Evidence</p>
              <h2>Realtime Evidence Stream</h2>
            </div>
            <p>${escapeHTML(probes.length + " probe records")}</p>
          </div>
          ${renderObsidianProbeLog(probes, state.primary_evidence)}
        </section>

        <section class="obs-split">
          <article class="obs-section">
            <div class="obs-section__head">
              <div>
                <p class="obs-section__eyebrow">Services</p>
                <h2>Key Service Surface</h2>
              </div>
            </div>
            <div class="service-list">
              ${renderServiceList(services)}
            </div>
          </article>

          <article class="obs-section">
            <div class="obs-section__head">
              <div>
                <p class="obs-section__eyebrow">System Metadata</p>
                <h2>Node Metadata</h2>
              </div>
            </div>
            <div class="obs-info-grid">
              ${renderObsidianInfoCard("Uptime", formatUptimeLong(state.uptime_s))}
              ${renderObsidianInfoCard("Leader", snapshot.leader_id || "Electing")}
              ${renderObsidianInfoCard("Heartbeat", timeAgo(state.last_heartbeat_at))}
              ${renderObsidianInfoCard("Peer Reach", (state.last_probe_summary.successful_peers || 0) + "/" + (state.last_probe_summary.total_peers || 0))}
            </div>
          </article>
        </section>

        <section class="obs-split">
          <article class="obs-section">
            <div class="obs-section__head">
              <div>
                <p class="obs-section__eyebrow">Decision Evidence</p>
                <h2>Primary Evidence</h2>
              </div>
            </div>
            <div class="evidence-list">
              ${renderEvidenceList(state.primary_evidence)}
            </div>
          </article>

          <article class="obs-section">
            <div class="obs-section__head">
              <div>
                <p class="obs-section__eyebrow">Probe Trail</p>
                <h2>Recent Probe Path</h2>
              </div>
            </div>
            <div class="probe-trail">
              ${renderProbeList(probes)}
            </div>
          </article>
        </section>

        <section class="obs-section">
          <div class="obs-section__head">
            <div>
              <p class="obs-section__eyebrow">Incident History</p>
              <h2>Node Incident Archive</h2>
            </div>
          </div>
          <div class="timeline">
            ${renderIncidentTimeline(incidents, {
              emptyTitle: "No incident history",
              emptyText: "This node has not opened any tracked incidents yet.",
              showNode: false
            })}
          </div>
        </section>
      </main>
    `;
  }

  function renderEventsPage(snapshot, incidents, events, meta) {
    const incidentList = Array.isArray(incidents) ? incidents : [];
    const eventList = Array.isArray(events) ? events : [];
    const activeIncidents = incidentList.filter((incident) => incident.status === "active");
    const resolvedIncidents = incidentList.filter((incident) => incident.status !== "active");
    const featuredIncident = activeIncidents[0] || null;
    const remainingActive = activeIncidents.slice(featuredIncident ? 1 : 0, 5);
    const criticalActive = activeIncidents.filter((incident) => normalizeStatus(incident.severity) === "critical").length;
    const degradedActive = activeIncidents.filter((incident) => normalizeStatus(incident.severity) === "degraded").length;

    return `
      <main class="obs-page obs-page--events">
        <section class="obs-events-header">
          <div class="obs-events-header__copy">
            <p class="obs-section__eyebrow">Operational Status</p>
            <h1>Incident <span>Center</span></h1>
          </div>
          <div class="obs-events-header__counts">
            ${renderObsidianCountCard("Active Critical", String(criticalActive))}
            ${renderObsidianCountCard("Degraded", String(degradedActive))}
          </div>
        </section>

        <section class="obs-events-grid">
          <div class="obs-events-main">
            ${featuredIncident
              ? renderObsidianIncidentHero(featuredIncident)
              : `
                <section class="obs-section obs-incidents-empty">
                  <p class="obs-section__eyebrow">Incident Feed</p>
                  <h2>System Stable</h2>
                  <p>No active incidents at the moment. The center is waiting for the next state transition.</p>
                </section>
              `}

            <div class="obs-incident-grid">
              ${remainingActive.length > 0
                ? remainingActive.map((incident) => renderObsidianIncidentCard(incident)).join("")
                : `
                  <div class="obs-incident-card obs-incident-card--muted">
                    <p class="obs-section__eyebrow">Degraded Feed</p>
                    <h3>No secondary incidents</h3>
                    <p>Additional active incidents will stack here when the cluster opens more than one issue at a time.</p>
                  </div>
                `}
            </div>
          </div>

          <aside class="obs-events-side">
            <section class="obs-event-rail">
              <div class="obs-event-rail__head">
                <p class="obs-section__eyebrow">Recent Events</p>
                <h2>Cluster Timeline</h2>
              </div>
              <div class="obs-event-feed">
                ${renderObsidianEventFeed(eventList)}
              </div>
            </section>

            <section class="obs-compose-panel">
              <div class="obs-compose-panel__head">
                <p class="obs-section__eyebrow">Alert Test</p>
                <h2>Manual Dispatch</h2>
              </div>
              ${renderTestAlertPanel(meta)}
            </section>
          </aside>
        </section>

        <section class="obs-section obs-archive-panel">
          <div class="obs-section__head">
            <div>
              <p class="obs-section__eyebrow">Resolution Archives</p>
              <h2>Closed Incident History</h2>
            </div>
          </div>
          ${renderObsidianResolvedRows(resolvedIncidents)}
        </section>
      </main>
    `;
  }

  function renderProbeMatrix(nodes) {
    if (!Array.isArray(nodes) || nodes.length === 0) {
      return emptyRune("No node visibility yet", "The matrix will appear after the first probe cycle completes.");
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
          <div class="matrix__row" style="grid-template-columns: 96px repeat(${nodes.length}, minmax(0, 1fr));">
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
      .map((incident) =>
        renderTimelineItem({
          status: incident.severity || incident.status,
          eyebrow: options.showNode
            ? (incident.node_id || "-") + " · " + (incident.rule_key || "incident")
            : incident.rule_key || "incident",
          title: incident.summary || "Unnamed incident",
          description: truncate(incident.detail || "", 118),
          rightTop: incident.status || "",
          rightBottom: formatDateTime(incident.resolved_at || incident.opened_at)
        })
      )
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
          title: event.title || "Unnamed event",
          description: truncate(event.body || "", options.compact ? 84 : 132),
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
        <div class="metric-bar__meter"><i style="width: ${escapeHTML(formatPercent(value))}"></i></div>
        <div class="metric-bar__chart">${sparkline(history, { area: true })}</div>
      </div>
    `;
  }

  function renderServiceList(services) {
    const list = Array.isArray(services) ? services : [];
    if (list.length === 0) {
      return emptyRune("No service checks", "Add systemd services in monitor.yaml to populate this section.");
    }

    return list
      .map(
        (service) => `
          <div class="service-row status-surface" data-status="${normalizeStatus(service.status)}">
            <div>
              <strong>${escapeHTML(service.name || "service")}</strong>
              <span>${escapeHTML(service.status || "unknown")}</span>
            </div>
            <small>${escapeHTML(service.detail || "No detail")}</small>
          </div>
        `
      )
      .join("");
  }

  function renderProbeList(probes) {
    const list = Array.isArray(probes) ? probes : [];
    if (list.length === 0) {
      return emptyRune("No probe data", "The next probe cycle will populate this evidence trail.");
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
      return '<p class="evidence-item">Waiting for the evaluation engine to publish rule evidence.</p>';
    }
    return list.map((item) => '<p class="evidence-item">' + escapeHTML(item) + "</p>").join("");
  }

  function renderTestAlertPanel(meta) {
    const channels = Array.isArray(meta.test_alert_channels) ? meta.test_alert_channels : [];
    const requiresToken = Boolean(meta.test_alert_requires_token);

    if (channels.length === 0) {
      return emptyRune("No alert channels enabled", "Enable Telegram, SMTP or a webhook in monitor.yaml to unlock manual dispatch.");
    }

    return `
      <form class="test-alert-form" id="test-alert-form">
        <label>
          <span>Channel</span>
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
          <span>Note</span>
          <textarea name="note" rows="4" placeholder="This message validates whether Telegram, SMTP or webhook delivery really arrives."></textarea>
        </label>
        <button type="submit" class="test-alert-form__submit">Send Test Alert</button>
      </form>
      <div class="test-alert-result" id="test-alert-result">
        <p>This panel sends a test alert only. It does not create a real incident.</p>
      </div>
    `;
  }

  function renderTimelineItem(options) {
    const description = options.description
      ? `<p class="timeline-item__description">${escapeHTML(options.description)}</p>`
      : "";
    const classNames = options.status ? "timeline-item status-surface" : "timeline-item";
    const statusAttr = options.status ? ` data-status="${normalizeStatus(options.status)}"` : "";

    return `
      <article class="${classNames}"${statusAttr}>
        <div class="timeline-item__body">
          <span>${escapeHTML(options.eyebrow || "")}</span>
          <strong>${escapeHTML(options.title || "")}</strong>
          ${description}
        </div>
        <div class="timeline-item__meta">
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

  function renderObsidianNavLink(href, iconName, label, active) {
    return `
      <a href="${href}" data-link class="obs-nav-item${active ? " is-active" : ""}">
        ${renderIcon(iconName, active)}
        <span>${escapeHTML(label)}</span>
      </a>
    `;
  }

  function renderObsidianMobileLink(href, iconName, label, active) {
    return `
      <a href="${href}" data-link class="obs-mobile-nav__item${active ? " is-active" : ""}">
        ${renderIcon(iconName, active)}
        <span>${escapeHTML(label)}</span>
      </a>
    `;
  }

  function renderObsidianMeta(label, value) {
    return `
      <div class="obs-meta">
        <span>${escapeHTML(label)}</span>
        <strong>${escapeHTML(value || "-")}</strong>
      </div>
    `;
  }

  function renderObsidianMetricCard(options) {
    const classes = ["obs-metric-card"];
    if (options.size === "wide") {
      classes.push("obs-metric-card--wide");
    }
    const statusAttr = options.tone ? ` data-status="${escapeHTML(options.tone)}"` : "";

    return `
      <article class="${classes.join(" ")}"${statusAttr}>
        <div class="obs-metric-card__head">
          <div class="obs-metric-card__copy">
            <span>${escapeHTML(options.label)}</span>
            <strong>${escapeHTML(options.value)}</strong>
          </div>
          <div class="obs-metric-card__icon">${renderIcon(options.icon || "monitoring")}</div>
        </div>
        <p class="obs-metric-card__caption">${escapeHTML(options.caption || "")}</p>
        <div class="obs-metric-card__visual">
          ${options.visual || ""}
        </div>
      </article>
    `;
  }

  function renderObsidianBarColumns(nodes, metricKey) {
    const list = Array.isArray(nodes) ? nodes : [];
    if (list.length === 0) {
      return '<div class="obs-column-bars obs-column-bars--empty"></div>';
    }

    return `
      <div class="obs-column-bars">
        ${list
          .map((node) => {
            const value = Math.max(8, Math.round(Number(node[metricKey]) || 0));
            return `<i title="${escapeHTML(node.node_id || "-")}" style="height:${Math.min(100, value)}%"></i>`;
          })
          .join("")}
      </div>
    `;
  }

  function renderObsidianMeter(value, peak) {
    const width = Math.max(0, Math.min(100, Math.round(Number(value) || 0)));
    const peakLabel = peak ? peak.nodeID + " · " + formatPercent(peak.value) : "No peak";

    return `
      <div class="obs-meter">
        <div class="obs-meter__track"><i style="width:${width}%"></i></div>
        <div class="obs-meter__meta">
          <span>Average</span>
          <strong>${escapeHTML(peakLabel)}</strong>
        </div>
      </div>
    `;
  }

  function renderObsidianNodeRow(node, snapshot) {
    const summary = node.last_probe_summary || {};
    const badges = [];
    if (snapshot && snapshot.leader_id === node.node_id) {
      badges.push(renderNodeBadge("LEADER", true));
    }
    if (snapshot && snapshot.ingress && snapshot.ingress.active_node_id === node.node_id) {
      badges.push(renderNodeBadge("INGRESS", false));
    }
    if (snapshot && snapshot.node_id === node.node_id) {
      badges.push(renderNodeBadge("LOCAL", false));
    }

    return `
      <article class="obs-node-row status-surface" data-status="${normalizeStatus(node.status)}">
        <div class="obs-node-row__identity">
          <div class="obs-node-row__icon">${renderIcon("dns", normalizeStatus(node.status) === "healthy")}</div>
          <div>
            <div class="obs-node-row__title">
              <strong>${escapeHTML(node.node_id || "-")}</strong>
              ${badges.join("")}
            </div>
            <p>${escapeHTML(truncate(node.reason || "No summary", 96))}</p>
          </div>
        </div>
        <div class="obs-node-row__status">
          <span class="obs-live-dot" data-status="${normalizeStatus(node.status)}"></span>
          <span>${escapeHTML(statusLabel(node.status))}</span>
        </div>
        <div class="obs-node-row__metric">
          <strong>${escapeHTML((summary.successful_peers || 0) + "/" + (summary.total_peers || 0))}</strong>
          <span>Peer Reach</span>
        </div>
        <div class="obs-node-row__metric">
          <strong>${escapeHTML(timeAgo(node.last_heartbeat_at))}</strong>
          <span>Load ${escapeHTML(formatLoad(node.load1))}</span>
        </div>
        <div class="obs-node-row__metric">
          <strong>${escapeHTML(formatUptimeLong(node.uptime_s))}</strong>
          <span>CPU ${escapeHTML(formatPercent(node.cpu_pct))}</span>
        </div>
        <a href="/nodes/${encodeURIComponent(node.node_id || "")}" data-link class="obs-node-row__action">Open</a>
      </article>
    `;
  }

  function renderNodeBadge(label, accent) {
    return `<span class="obs-node-badge${accent ? " is-accent" : ""}">${escapeHTML(label)}</span>`;
  }

  function renderObsidianIncidentHero(incident) {
    const tone = normalizeStatus(incident.severity);
    return `
      <section class="obs-incident-hero" data-status="${tone}">
        <div class="obs-incident-hero__bar">
          <div>
            <span>${renderIcon("warning", true)} ${escapeHTML(featuredIncidentLabel(tone))}</span>
            <strong>${escapeHTML(timeAgo(incident.opened_at))}</strong>
          </div>
        </div>
        <div class="obs-incident-hero__body">
          <div class="obs-incident-hero__copy">
            <h2>${escapeHTML(incident.summary || "Unnamed incident")}</h2>
            <p>${escapeHTML(incident.detail || "No incident detail supplied.")}</p>
          </div>
          <div class="obs-incident-hero__stats">
            ${renderObsidianInfoCard("Node", incident.node_id || "-")}
            ${renderObsidianInfoCard("Rule", incident.rule_key || "-")}
          </div>
        </div>
      </section>
    `;
  }

  function renderObsidianIncidentCard(incident) {
    return `
      <article class="obs-incident-card" data-status="${normalizeStatus(incident.severity)}">
        <p class="obs-section__eyebrow">${escapeHTML((incident.status || "active") + " · " + (incident.node_id || "-"))}</p>
        <h3>${escapeHTML(incident.summary || "Unnamed incident")}</h3>
        <p>${escapeHTML(truncate(incident.detail || "", 116))}</p>
        <div class="obs-incident-card__meta">
          <span>${escapeHTML(incident.rule_key || "incident")}</span>
          <strong>${escapeHTML(timeAgo(incident.opened_at))}</strong>
        </div>
      </article>
    `;
  }

  function renderObsidianCountCard(label, value) {
    return `
      <article class="obs-events-count">
        <span>${escapeHTML(label)}</span>
        <strong>${escapeHTML(value)}</strong>
      </article>
    `;
  }

  function renderObsidianEventFeed(events) {
    const list = Array.isArray(events) ? events : [];
    if (list.length === 0) {
      return emptyRune("No recent events", "The event rail will populate after leader and DNS activity is recorded.");
    }

    return list
      .map((event) => `
        <article class="obs-event-item" data-status="${normalizeStatus(event.severity)}">
          <div class="obs-event-item__kind">${escapeHTML(event.kind || "event")}</div>
          <div class="obs-event-item__body">
            <strong>${escapeHTML(event.title || "Unnamed event")}</strong>
            <p>${escapeHTML(truncate(event.body || "", 120))}</p>
          </div>
          <div class="obs-event-item__meta">
            <span>${escapeHTML(event.node_id || "-")}</span>
            <time>${escapeHTML(formatDateTime(event.created_at))}</time>
          </div>
        </article>
      `)
      .join("");
  }

  function renderObsidianResolvedRows(incidents) {
    const list = Array.isArray(incidents) ? incidents : [];
    if (list.length === 0) {
      return emptyRune("No resolved incidents", "Closed incident history will accumulate here after the first recovery.");
    }

    return `
      <div class="obs-archive-table">
        ${list
          .map((incident) => {
            const downtime = incident.resolved_at
              ? durationBetween(incident.opened_at, incident.resolved_at)
              : "-";
            return `
              <article class="obs-archive-row">
                <div class="obs-archive-row__status">
                  <span class="obs-live-dot" data-status="healthy"></span>
                  <strong>Resolved</strong>
                </div>
                <span>${escapeHTML(incident.id || "-")}</span>
                <span>${escapeHTML(incident.summary || "Unnamed incident")}</span>
                <span>${escapeHTML(incident.node_id || "-")}</span>
                <span>${escapeHTML(downtime)}</span>
              </article>
            `;
          })
          .join("")}
      </div>
    `;
  }

  function renderObsidianProbeLog(probes, evidence) {
    const lines = [];
    const probeList = Array.isArray(probes) ? probes : [];
    const evidenceList = Array.isArray(evidence) ? evidence : [];

    probeList.slice(0, 10).forEach((probe) => {
      const level = probeTone(probe);
      lines.push(`
        <div class="obs-log-line" data-status="${level}">
          <span class="obs-log-line__time">${escapeHTML(formatDateTime(probe.collected_at))}</span>
          <span class="obs-log-line__level">${escapeHTML(level.toUpperCase())}</span>
          <span class="obs-log-line__body">${escapeHTML((probe.source_node_id || "?") + " → " + (probe.target_node_id || "?") + " · 22 " + (probe.tcp_22_ok ? "OPEN" : "DROP") + " · 443 " + (probe.tcp_443_ok ? "OPEN" : "DROP") + " · HTTP " + (probe.http_ok ? "OK" : "MISS"))}</span>
        </div>
      `);
    });

    evidenceList.slice(0, 6).forEach((item) => {
      lines.push(`
        <div class="obs-log-line" data-status="unknown">
          <span class="obs-log-line__time">evidence</span>
          <span class="obs-log-line__level">RULE</span>
          <span class="obs-log-line__body">${escapeHTML(item)}</span>
        </div>
      `);
    });

    if (lines.length === 0) {
      return emptyRune("No streaming evidence", "Probe data and rule evidence will appear here when the next monitoring cycle lands.");
    }

    return `<div class="obs-log-stream">${lines.join("")}</div>`;
  }

  function renderObsidianInfoCard(label, value) {
    return `
      <div class="obs-info-card">
        <span>${escapeHTML(label)}</span>
        <strong>${escapeHTML(value || "-")}</strong>
      </div>
    `;
  }

  function buildSearchIndex(options) {
    const snapshot = options && options.snapshot ? options.snapshot : {};
    const nodes = Array.isArray(snapshot.nodes) ? snapshot.nodes : [];
    const incidents = Array.isArray(options && options.incidents) ? options.incidents : [];
    const events = Array.isArray(options && options.events) ? options.events : [];
    const currentNodeID = options && options.currentNodeID ? options.currentNodeID : snapshot.node_id || "";
    const items = [];
    const seen = new Set();

    function add(item) {
      if (!item || !item.href) {
        return;
      }
      const key = [item.kind, item.href, item.title].join("|");
      if (seen.has(key)) {
        return;
      }
      seen.add(key);
      items.push(item);
    }

    add({
      kind: "View",
      title: "Observatory",
      subtitle: "Cluster overview dashboard",
      href: "/",
      icon: "dashboard",
      pinned: true,
      keywords: "overview observatory dashboard cluster home"
    });
    add({
      kind: "View",
      title: "Incidents",
      subtitle: activeIncidentCount(incidents) + " active incidents",
      href: "/events",
      icon: "warning",
      pinned: true,
      keywords: "incidents events alerts center"
    });

    if (isUsableNodeID(snapshot.node_id)) {
      add({
        kind: "View",
        title: "Local Node",
        subtitle: snapshot.node_id,
        href: "/nodes/" + encodeURIComponent(snapshot.node_id),
        icon: "dns",
        pinned: true,
        keywords: "local current node " + snapshot.node_id
      });
    }

    if (isUsableNodeID(currentNodeID) && currentNodeID !== snapshot.node_id) {
      add({
        kind: "View",
        title: "Current Node",
        subtitle: currentNodeID,
        href: "/nodes/" + encodeURIComponent(currentNodeID),
        icon: "lan",
        pinned: true,
        keywords: "current node detail " + currentNodeID
      });
    }

    nodes.forEach((node) => {
      const probeSummary = node.last_probe_summary || {};
      add({
        kind: "Node",
        title: node.node_id || "-",
        subtitle: truncate(node.reason || statusLabel(node.status), 88),
        href: "/nodes/" + encodeURIComponent(node.node_id || ""),
        icon: "dns",
        status: normalizeStatus(node.status),
        keywords: [
          node.node_id,
          node.reason,
          node.status,
          node.rule_key,
          "cpu " + formatPercent(node.cpu_pct),
          "memory " + formatPercent(node.mem_pct),
          "disk " + formatPercent(node.disk_pct),
          "peers " + (probeSummary.successful_peers || 0) + "/" + (probeSummary.total_peers || 0)
        ].join(" ")
      });
    });

    incidents.forEach((incident) => {
      add({
        kind: "Incident",
        title: incident.summary || "Unnamed incident",
        subtitle: (incident.node_id || "-") + " · " + (incident.rule_key || "incident"),
        href: "/events",
        icon: "warning",
        status: normalizeStatus(incident.severity || incident.status),
        keywords: [
          incident.id,
          incident.node_id,
          incident.rule_key,
          incident.summary,
          incident.detail,
          incident.status,
          incident.severity
        ].join(" ")
      });
    });

    events.forEach((event) => {
      add({
        kind: "Event",
        title: event.title || "Unnamed event",
        subtitle: (event.kind || "event") + " · " + (event.node_id || "-"),
        href: "/events",
        icon: "history",
        status: normalizeStatus(event.severity),
        keywords: [
          event.kind,
          event.node_id,
          event.title,
          event.body,
          event.severity
        ].join(" ")
      });
    });

    return items;
  }

  function clusterSummary(snapshot, nodes, incidents, counts) {
    if (!Array.isArray(nodes) || nodes.length === 0) {
      return {
        tone: "unknown",
        word: "WAITING",
        description: "The dashboard is ready. It will settle once the first node reports telemetry."
      };
    }
    const total = nodes.length;
    if (counts.critical > 0 || incidents.some((incident) => normalizeStatus(incident.severity) === "critical")) {
      return {
        tone: "critical",
        word: "ALERT",
        description: "Critical nodes or incidents are open. Start with the failing node, then verify ingress and peer visibility."
      };
    }
    if (counts.degraded > 0 || activeIncidentCount(incidents) > 0) {
      return {
        tone: "degraded",
        word: "DEGRADED",
        description: "The cluster is still serving, but at least one layer is thinning. Review degraded nodes before the next transition escalates."
      };
    }
    return {
      tone: "healthy",
      word: "STABLE",
      description: "All " + total + " nodes are within expected thresholds. No active incidents are diluting operator attention."
    };
  }

  function clusterAverages(nodes) {
    const list = Array.isArray(nodes) ? nodes : [];
    if (list.length === 0) {
      return { cpu: 0, mem: 0, disk: 0 };
    }

    const totals = list.reduce(
      (acc, node) => {
        acc.cpu += Number(node.cpu_pct) || 0;
        acc.mem += Number(node.mem_pct) || 0;
        acc.disk += Number(node.disk_pct) || 0;
        return acc;
      },
      { cpu: 0, mem: 0, disk: 0 }
    );

    return {
      cpu: totals.cpu / list.length,
      mem: totals.mem / list.length,
      disk: totals.disk / list.length
    };
  }

  function clusterCoverageText(counts, total) {
    if (!total) {
      return "0 / 0";
    }
    return Math.min(total, counts.healthy + counts.degraded) + " / " + total;
  }

  function primaryHistory(snapshot, nodes, historyMap) {
    const map = historyMap || {};
    const preferred = snapshot && snapshot.node_id ? map[snapshot.node_id] : null;
    if (Array.isArray(preferred) && preferred.length > 0) {
      return preferred;
    }

    for (const node of nodes || []) {
      const points = map[node.node_id];
      if (Array.isArray(points) && points.length > 0) {
        return points;
      }
    }

    return [];
  }

  function highestNodeMetric(nodes, metricKey) {
    let selected = null;
    for (const node of Array.isArray(nodes) ? nodes : []) {
      const value = Number(node[metricKey]);
      if (!Number.isFinite(value)) {
        continue;
      }
      if (!selected || value > selected.value) {
        selected = { nodeID: node.node_id || "-", value: value };
      }
    }
    return selected;
  }

  function activeIncidentCount(incidents) {
    return (Array.isArray(incidents) ? incidents : []).filter((incident) => (incident.status || "active") === "active").length;
  }

  function normalizeSearchText(value) {
    return String(value || "")
      .toLowerCase()
      .replace(/\s+/g, " ")
      .trim();
  }

  function nodeVisibilityPct(state) {
    const summary = state && state.last_probe_summary ? state.last_probe_summary : {};
    if (summary.total_peers > 0) {
      return Math.max(0, Math.min(100, Math.round((summary.successful_peers / summary.total_peers) * 100)));
    }
    return normalizeStatus(state && state.status) === "healthy" ? 100 : 0;
  }

  function nodeRoleLabel(snapshot, nodeID) {
    if (snapshot && snapshot.ingress && snapshot.ingress.active_node_id === nodeID) {
      return "ACTIVE";
    }
    return "STANDBY";
  }

  function formatLoad(value) {
    const parsed = Number(value);
    if (!Number.isFinite(parsed)) {
      return "-";
    }
    return parsed.toFixed(2);
  }

  function formatUptimeCompact(value) {
    const seconds = Number(value);
    if (!Number.isFinite(seconds) || seconds <= 0) {
      return "0H";
    }
    const hours = Math.floor(seconds / 3600);
    if (hours < 24) {
      return hours + "H";
    }
    const days = Math.floor(hours / 24);
    return days + "D";
  }

  function formatUptimeLong(value) {
    const seconds = Number(value);
    if (!Number.isFinite(seconds) || seconds <= 0) {
      return "0m";
    }
    const days = Math.floor(seconds / 86400);
    const hours = Math.floor((seconds % 86400) / 3600);
    const minutes = Math.floor((seconds % 3600) / 60);
    if (days > 0) {
      return days + "d " + hours + "h " + minutes + "m";
    }
    if (hours > 0) {
      return hours + "h " + minutes + "m";
    }
    return minutes + "m";
  }

  function durationBetween(from, to) {
    const start = new Date(from);
    const end = new Date(to);
    if (Number.isNaN(start.getTime()) || Number.isNaN(end.getTime())) {
      return "-";
    }
    const diff = Math.max(0, end.getTime() - start.getTime());
    const minutes = Math.round(diff / 60000);
    if (minutes < 60) {
      return minutes + "m";
    }
    const hours = Math.floor(minutes / 60);
    const restMinutes = minutes % 60;
    return hours + "h " + restMinutes + "m";
  }

  function sidebarContextLabel(page) {
    switch (page) {
      case "events":
        return "Incident Center";
      case "node":
        return "Node Detail";
      default:
        return "Observatory";
    }
  }

  function isUsableNodeID(value) {
    const input = String(value || "").trim();
    return input !== "" && input !== "-" && input !== "...";
  }

  function featuredIncidentLabel(tone) {
    switch (tone) {
      case "critical":
        return "Critical Incident";
      case "degraded":
        return "Degraded Incident";
      default:
        return "Active Incident";
    }
  }

  function renderIcon(name, filled) {
    const fill = filled ? 1 : 0;
    return `<span class="material-symbols-outlined" style="font-variation-settings:'FILL' ${fill},'wght' 500,'GRAD' 0,'opsz' 24;">${escapeHTML(name)}</span>`;
  }

  function sparkline(points, options) {
    const settings = options || {};
    const list = Array.isArray(points) ? points : [];
    if (list.length === 0) {
      return '<svg viewBox="0 0 100 40" preserveAspectRatio="none"><path d="M0 34 L100 34"></path></svg>';
    }

    let maxValue = 0;
    for (const point of list) {
      maxValue = Math.max(maxValue, Number(point.value) || 0);
    }
    if (maxValue <= 0) {
      maxValue = 1;
    }

    const commands = [];
    const area = [];
    list.forEach((point, index) => {
      const x = (index / Math.max(1, list.length - 1)) * 100;
      const y = 34 - ((Number(point.value) || 0) / maxValue) * 26;
      commands.push((index === 0 ? "M" : "L") + x.toFixed(2) + " " + y.toFixed(2));
      area.push((index === 0 ? "M" : "L") + x.toFixed(2) + " " + y.toFixed(2));
    });
    area.push("L100 40 L0 40 Z");

    sparkline.counter = (sparkline.counter || 0) + 1;
    const gradientID = "sparkline-fill-" + sparkline.counter;

    return `
      <svg viewBox="0 0 100 40" preserveAspectRatio="none">
        <defs>
          <linearGradient id="${gradientID}" x1="0" x2="0" y1="0" y2="1">
            <stop offset="0%" stop-color="currentColor" stop-opacity="${settings.area ? "0.26" : "0"}"></stop>
            <stop offset="100%" stop-color="currentColor" stop-opacity="0"></stop>
          </linearGradient>
        </defs>
        <g data-tone="${escapeHTML(settings.tone || "healthy")}">
          ${settings.area ? `<path d="${area.join(" ")}" fill="url(#${gradientID})"></path>` : ""}
          <path d="${commands.join(" ")}"></path>
        </g>
      </svg>
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

  function sparkline(points, options) {
    const settings = options || {};
    const list = Array.isArray(points) ? points : [];
    if (list.length === 0) {
      return '<svg viewBox="0 0 100 40" preserveAspectRatio="none"><path d="M0 34 L100 34"></path></svg>';
    }

    let maxValue = 0;
    for (const point of list) {
      maxValue = Math.max(maxValue, Number(point.value) || 0);
    }
    if (maxValue <= 0) {
      maxValue = 1;
    }

    const commands = [];
    const area = [];
    list.forEach((point, index) => {
      const x = (index / Math.max(1, list.length - 1)) * 100;
      const y = 34 - ((Number(point.value) || 0) / maxValue) * 26;
      commands.push((index === 0 ? "M" : "L") + x.toFixed(2) + " " + y.toFixed(2));
      area.push((index === 0 ? "M" : "L") + x.toFixed(2) + " " + y.toFixed(2));
    });
    area.push("L100 40 L0 40 Z");

    sparkline.counter = (sparkline.counter || 0) + 1;
    const gradientID = "sparkline-fill-" + sparkline.counter;

    return `
      <svg viewBox="0 0 100 40" preserveAspectRatio="none">
        <defs>
          <linearGradient id="${gradientID}" x1="0" x2="0" y1="0" y2="1">
            <stop offset="0%" stop-color="currentColor" stop-opacity="${settings.area ? "0.26" : "0"}"></stop>
            <stop offset="100%" stop-color="currentColor" stop-opacity="0"></stop>
          </linearGradient>
        </defs>
        <g data-tone="${escapeHTML(settings.tone || "healthy")}">
          ${settings.area ? `<path d="${area.join(" ")}" fill="url(#${gradientID})"></path>` : ""}
          <path d="${commands.join(" ")}"></path>
        </g>
      </svg>
    `;
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
