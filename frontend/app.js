(function () {
  const app = document.getElementById("app");
  if (!app) {
    return;
  }

  const REFRESH_INTERVAL_MS = 15000;
  const THEME_KEY = "vps-monitor-theme";
  const LANGUAGE_KEY = "vps-monitor-language";
  const THEMES = [
    { id: "graphite", label: "Graphite" },
    { id: "porcelain", label: "Porcelain" },
    { id: "brass", label: "Brass" }
  ];
  const LANGUAGES = [
    { id: "zh", label: "中文" },
    { id: "en", label: "English" }
  ];
  const UI_TEXT = {
    zh: {
      "Observatory": "总览",
      "Local Node": "本机节点",
      "Incidents": "事件",
      "Admin Control": "管理",
      "Admin": "管理",
      "Login": "登录",
      "Command Center": "控制台",
      "Node Detail": "节点详情",
      "Incident Center": "事件中心",
      "Open Local Node": "打开本机节点",
      "Current Node": "当前节点",
      "Leader": "Leader",
      "Self-hosted control console": "自托管控制台",
      "Admin Session": "管理员会话",
      "Cluster Leader": "集群 Leader",
      "Authorized": "已授权",
      "Electing": "选举中",
      "Quick Access": "快速访问",
      "No Matches": "没有匹配项",
      "Search nodes, incidents, or telemetry...": "搜索节点、事件或观测数据...",
      "Type a node ID, incident summary, rule key, status, or event title.": "输入节点 ID、事件摘要、规则键、状态或事件标题。",
      "Authentication": "认证",
      "Bootstrap": "初始化",
      "Session": "会话",
      "Password": "密码",
      "Current Password": "当前密码",
      "New Password": "新密码",
      "Runtime Checks": "运行时检测",
      "Current Checks": "当前检测项",
      "Node Names": "节点名称",
      "Current Names": "当前名称",
      "Name": "名称",
      "Type": "类型",
      "Node Scope": "节点范围",
      "All nodes": "全部节点",
      "Only selected nodes": "仅选中节点",
      "All except selected nodes": "排除选中节点",
      "Target Nodes": "目标节点",
      "Service Name": "服务名",
      "Container Name": "容器名",
      "Scheme": "协议",
      "Host Mode": "主机模式",
      "Port": "端口",
      "Path": "路径",
      "Expect Status": "期望状态码",
      "Timeout": "超时",
      "Label": "标签",
      "Enabled": "启用",
      "Join the next collection cycle immediately": "保存后下一轮立即生效",
      "Node": "节点",
      "Node ": "节点 ",
      "Display Name": "显示名称",
      "No checks yet": "还没有检测项",
      "Create the first runtime check to replace static monitor.yaml service lists.": "先创建第一条运行时检测项，用来替代静态 monitor.yaml 服务列表。",
      "No nodes configured": "还没有配置节点",
      "Cluster peers will appear here after configuration loads.": "配置加载完成后，集群节点会出现在这里。",
      "System Stable": "系统稳定",
      "Cluster Timeline": "集群时间线",
      "Manual Dispatch": "手动派发",
      "Closed Incident History": "已关闭事件历史",
      "Active Critical": "活跃严重",
      "Degraded": "降级",
      "Active Incidents": "活跃事件",
      "Updated": "更新时间",
      "Loading": "加载中",
      "Unavailable": "不可用",
      "Deployment Nodes": "部署节点",
      "Active Deployment Nodes": "活跃部署节点",
      "Node Identity": "节点标识",
      "Status": "状态",
      "Peer Reach": "互探可达",
      "Heartbeat": "心跳",
      "Uptime": "运行时间",
      "Action": "操作",
      "View": "视图",
      "Incident": "事件",
      "Event": "事件流",
      "administrator already initialized": "管理员已经初始化",
      "administrator is not initialized": "管理员尚未初始化",
      "invalid password": "密码错误",
      "current password is invalid": "当前密码错误",
      "admin login required": "需要管理员登录",
      "Language": "语言",
      "Hardware Profile": "硬件配置",
      "Machine Specifications": "机器规格",
      "CPU": "CPU",
      "Cores": "核心数",
      "Memory": "内存",
      "Disk": "磁盘",
      "OS": "操作系统",
      "Kernel": "内核"
    },
    en: {
      "总览": "Overview",
      "管理": "Admin",
      "事件": "Events",
      "节点 ": "Node ",
      "未找到页面": "Page Not Found",
      "加载中": "Loading",
      "不可用": "Unavailable",
      "加载失败": "Load Failed",
      "接口请求失败。": "API request failed.",
      "页面不存在": "Page Not Found",
      "这个路径没有对应的前端页面。": "There is no frontend page for this path.",
      "入口当前指向 ": "Ingress currently points to ",
      "入口正在等待新的 active node": "Ingress is waiting for a new active node",
      "Ingress 节点": "Ingress Node",
      "待选举": "Pending",
      "当前对外流量落点": "Current external traffic target",
      "DNS 同步": "DNS Sync",
      "已同步": "Synced",
      "待同步": "Pending",
      "尚未同步": "Not synced yet",
      "活跃 Incident": "Active Incidents",
      "当前需要处理的异常": "Issues needing attention now",
      "Critical 节点": "Critical Nodes",
      "Ingress IP": "Ingress IP",
      "期望的对外地址": "Desired public address",
      "最近事件": "Recent Events",
      "这一轮同步写入的轨迹": "Events written in this sync round",
      "节点主舞台": "Node Stage",
      "节点健康主舞台": "Node Health Stage",
      "节点互探矩阵": "Peer Probe Matrix",
      "活跃 Incident": "Active Incidents",
      "现在最该处理的异常": "Highest-priority active issues",
      "当前没有活动 incident": "No Active Incidents",
      "最近状态轨迹": "Recent State Timeline",
      "还没有事件": "No Events Yet",
      "节点档案": "Node Profile",
      "无说明": "No details",
      "最后一次心跳": "Last heartbeat",
      "当前决策节点": "Current decision node",
      "当前入口角色": "Current ingress role",
      "近期历史长度": "Recent history size",
      "判定证据": "Decision Evidence",
      "为什么它会被判成现在这样": "Why it is classified this way",
      "展开原始观测证据": "Expand raw observation evidence",
      "资源曲线": "Resource Curves",
      "资源曲线与当前占用": "Resource Curves and Current Usage",
      "服务": "Services",
      "关键服务面": "Critical Service Surface",
      "互探轨迹": "Probe Trail",
      "互探路径证据": "Probe Path Evidence",
      "Incident 历史": "Incident History",
      "这个节点最近发生过什么": "What happened on this node recently",
      "这个节点还没有 incident 历史": "This node has no incident history yet",
      "事件时间线": "Event Timeline",
      "从事件流看集群如何做决定": "How the cluster decides from the event stream",
      "当前入口落点": "Current ingress target",
      "域名回源状态": "DNS origin status",
      "当前仍在打开的异常": "Incidents still open",
      "最近一次采样保留下来的轨迹": "Events kept from the latest sampling round",
      "告警测试": "Alert Test",
      "手动打一次告警": "Trigger a Manual Alert",
      "Incident 轨迹": "Incident Timeline",
      "问题如何打开与恢复": "How incidents opened and recovered",
      "集群事件": "Cluster Events",
      "Leader、DNS 与状态切换": "Leader, DNS, and state transitions",
      "事件流暂时空着": "Event stream is empty",
      "设置管理员密码": "Set Administrator Password",
      "初始化管理员": "Initialize Admin",
      "管理员登录": "Admin Login",
      "登录": "Login",
      "管理员会话": "Admin Session",
      "退出登录": "Log Out",
      "修改管理员密码": "Change Admin Password",
      "更新密码": "Update Password",
      "检测项编辑器": "Check Editor",
      "保存检测项": "Save Check",
      "清空": "Clear",
      "已生效检测项": "Active Checks",
      "节点显示名称": "Node Display Names",
      "保存节点名称": "Save Node Name",
      "当前节点名称映射": "Current Node Name Mappings",
      "编辑": "Edit",
      "删除": "Delete",
      "恢复默认": "Reset",
      "进入节点": "Open Node",
      "暂无节点数据": "No Node Data Yet",
      "未命名 incident": "Unnamed incident",
      "未命名事件": "Unnamed event",
      "没有配置服务检查": "No Service Checks Configured",
      "还没有探测数据": "No Probe Data Yet",
      "没有启用任何告警渠道": "No Alert Channels Enabled",
      "渠道": "Channel",
      "备注": "Note",
      "发送测试告警": "Send Test Alert",
      "管理员已初始化。": "Administrator initialized.",
      "初始化失败": "Initialization failed",
      "正在初始化管理员...": "Initializing administrator...",
      "正在登录...": "Signing in...",
      "登录成功。": "Login successful.",
      "登录失败": "Login failed",
      "正在退出...": "Signing out...",
      "已退出。": "Logged out.",
      "退出失败": "Logout failed",
      "正在更新密码...": "Updating password...",
      "密码已更新。": "Password updated.",
      "更新密码失败": "Password update failed",
      "正在保存检测项...": "Saving check...",
      "检测项已保存。": "Check saved.",
      "保存检测项失败": "Failed to save check",
      "请选择节点。": "Select a node.",
      "正在保存节点名称...": "Saving node name...",
      "节点名称已保存。": "Node name saved.",
      "保存节点名称失败": "Failed to save node name",
      "正在删除检测项...": "Deleting check...",
      "检测项已删除。": "Check deleted.",
      "删除检测项失败": "Failed to delete check",
      "正在恢复默认名称...": "Restoring default name...",
      "已恢复默认名称。": "Default name restored.",
      "恢复默认名称失败": "Failed to restore default name",
      "无法读取检测项内容。": "Unable to read check data.",
      "检测项不存在或已刷新。": "The check no longer exists or the page has refreshed.",
      "无法读取节点名称内容。": "Unable to read node name data.",
      "节点映射不存在或已刷新。": "The node mapping no longer exists or the page has refreshed.",
      "正在发送测试告警...": "Sending test alert...",
      "发送时间 ": "Sent at ",
      "留空则恢复默认名称": "Leave empty to restore the default name",
      "等待第一轮评估产出规则命中信息。": "Waiting for the first evaluation pass to produce rule-hit evidence.",
      "先在 monitor.yaml 里开启 Telegram、SMTP 或 webhook 之后，这里才会出现测试入口。": "Enable Telegram, SMTP, or webhook in monitor.yaml first. The test entry point appears after that.",
      "这条消息用于验证 Telegram / SMTP 是否真的能送达。": "Use this note to verify Telegram or SMTP delivery.",
      "这里只会发送测试消息，不会创建真实 incident。": "This only sends a test message. It does not create a real incident.",
      "Language": "Language"
    }
  };

  let refreshTimer = 0;
  let renderToken = 0;
  let lastRouteKey = "";
  let currentSearchItems = [];
  let currentSearchQuery = "";
  let searchPanelOpen = false;
  let openControlMenu = "";
  let openAdminSelectMenuID = "";
  let sidebarDrawerOpen = false;
  let activeMeta = {};
  let currentAdminChecks = [];
  let currentAdminNodes = [];
  let currentAdminMembers = [];
  let activeAdminCheckID = "";
  let activeAdminNodeID = "";

  document.addEventListener("click", handleDocumentClick);
  document.addEventListener("change", handleDocumentChange);
  document.addEventListener("input", handleDocumentInput);
  document.addEventListener("focusin", handleDocumentFocusIn);
  document.addEventListener("keydown", handleDocumentKeydown);
  document.addEventListener("submit", handleDocumentSubmit);
  window.addEventListener("popstate", () => {
    renderRoute();
  });
  window.addEventListener("resize", handleWindowResize);

  applyTheme(document.documentElement.dataset.theme || document.body.dataset.defaultTheme || "graphite");
  applyLanguage(getStoredLanguage());
  renderRoute();

  // Sparkline tooltip
  const sparklineTooltip = document.createElement("div");
  sparklineTooltip.className = "sparkline-tooltip";
  document.body.appendChild(sparklineTooltip);

  document.addEventListener("mousemove", function (event) {
    const svg = event.target.closest("svg");
    if (!svg || !svg.closest(".obs-chart-panel__chart, .obs-aside__spark, .metric-bar__chart")) {
      sparklineTooltip.classList.remove("is-visible");
      return;
    }
    const rect = svg.getBoundingClientRect();
    const xRatio = (event.clientX - rect.left) / rect.width;
    const path = svg.querySelector("g > path:last-child");
    if (!path) return;
    const pathLen = path.getTotalLength();
    const point = path.getPointAtLength(xRatio * pathLen);
    const maxY = 34;
    const minY = 8;
    const pct = Math.max(0, Math.min(100, ((maxY - point.y) / (maxY - minY)) * 100));
    sparklineTooltip.textContent = pct.toFixed(1) + "%";
    sparklineTooltip.style.left = (event.clientX + 10) + "px";
    sparklineTooltip.style.top = (event.clientY - 28) + "px";
    sparklineTooltip.classList.add("is-visible");
  });

  document.addEventListener("mouseleave", function () {
    sparklineTooltip.classList.remove("is-visible");
  }, true);

  function handleDocumentClick(event) {
    const searchRoot = event.target.closest("[data-search-root]");
    if (!searchRoot) {
      closeSearchPanel();
    }
    if (!event.target.closest("[data-control-menu]")) {
      closeControlMenu();
    }
    if (!event.target.closest("[data-admin-select]")) {
      closeAdminSelectMenu();
    }

    const controlOption = event.target.closest("[data-control-option]");
    if (controlOption) {
      event.preventDefault();
      handleControlOption(controlOption);
      return;
    }

    const controlTrigger = event.target.closest("[data-control-trigger]");
    if (controlTrigger) {
      event.preventDefault();
      closeSearchPanel();
      toggleControlMenu(controlTrigger.dataset.controlTrigger || "");
      return;
    }

    const adminSelectOption = event.target.closest("[data-admin-select-option]");
    if (adminSelectOption) {
      event.preventDefault();
      handleAdminSelectOption(adminSelectOption);
      return;
    }

    const adminSelectTrigger = event.target.closest("[data-admin-select-trigger]");
    if (adminSelectTrigger) {
      event.preventDefault();
      closeSearchPanel();
      closeControlMenu();
      const root = adminSelectTrigger.closest("[data-admin-select]");
      toggleAdminSelectMenu(root ? (root.dataset.adminSelectId || "") : "");
      return;
    }

    const sidebarToggle = event.target.closest("[data-sidebar-toggle]");
    if (sidebarToggle) {
      event.preventDefault();
      toggleSidebarDrawer();
      return;
    }

    const sidebarDismiss = event.target.closest("[data-sidebar-dismiss]");
    if (sidebarDismiss) {
      event.preventDefault();
      toggleSidebarDrawer(false);
      return;
    }

    const actionButton = event.target.closest("[data-admin-action]");
    if (actionButton) {
      event.preventDefault();
      handleAdminAction(actionButton);
      return;
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
    toggleSidebarDrawer(false);
    if (url.pathname !== window.location.pathname) {
      window.history.pushState({}, "", url.pathname);
    }
    renderRoute();
  }

  function handleDocumentChange(event) {
    if (event.target.matches("[data-check-type]") || event.target.matches("[data-check-scope]")) {
      syncAdminCheckForm();
    }
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
    if (event.key === "Escape" && openAdminSelectMenuID) {
      event.preventDefault();
      closeAdminSelectMenu();
      return;
    }

    if (event.key === "Escape" && openControlMenu) {
      event.preventDefault();
      closeControlMenu();
      return;
    }

    if (event.key === "Escape" && sidebarDrawerOpen) {
      event.preventDefault();
      toggleSidebarDrawer(false);
      return;
    }

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

  function handleWindowResize() {
    if (openAdminSelectMenuID) {
      closeAdminSelectMenu();
    }
    if (openControlMenu) {
      closeControlMenu();
    }
    if (window.innerWidth > 980 && sidebarDrawerOpen) {
      toggleSidebarDrawer(false);
    }
  }

  function closeControlMenu() {
    if (!openControlMenu) {
      return;
    }
    openControlMenu = "";
    syncControlMenus();
  }

  function toggleControlMenu(menuID) {
    if (!menuID) {
      closeControlMenu();
      return;
    }
    openControlMenu = openControlMenu === menuID ? "" : menuID;
    syncControlMenus();
  }

  function syncControlMenus() {
    document.querySelectorAll("[data-control-menu]").forEach((menu) => {
      const menuID = menu.dataset.controlMenu || "";
      const isOpen = menuID === openControlMenu;
      menu.classList.toggle("is-open", isOpen);
      const trigger = menu.querySelector("[data-control-trigger]");
      if (trigger) {
        trigger.setAttribute("aria-expanded", isOpen ? "true" : "false");
      }
      const panel = menu.querySelector(".obs-theme-switcher__panel");
      if (panel) {
        panel.setAttribute("aria-hidden", isOpen ? "false" : "true");
      }
    });
  }

  function closeAdminSelectMenu() {
    if (!openAdminSelectMenuID) {
      return;
    }
    openAdminSelectMenuID = "";
    syncAdminSelectMenus();
  }

  function toggleAdminSelectMenu(menuID) {
    if (!menuID) {
      closeAdminSelectMenu();
      return;
    }
    openAdminSelectMenuID = openAdminSelectMenuID === menuID ? "" : menuID;
    syncAdminSelectMenus();
  }

  function syncAdminSelectMenus() {
    document.querySelectorAll("[data-admin-select]").forEach((root) => {
      const menuID = root.dataset.adminSelectId || "";
      const isOpen = menuID !== "" && menuID === openAdminSelectMenuID;
      root.classList.toggle("is-open", isOpen);

      const trigger = root.querySelector("[data-admin-select-trigger]");
      if (trigger) {
        trigger.setAttribute("aria-expanded", isOpen ? "true" : "false");
      }
      const panel = root.querySelector(".admin-select__panel");
      if (panel) {
        panel.setAttribute("aria-hidden", isOpen ? "false" : "true");
      }

      const input = root.querySelector('input[type="hidden"][name]');
      const value = input && "value" in input ? String(input.value || "") : "";
      let selectedLabel = "";

      root.querySelectorAll("[data-admin-select-option]").forEach((button) => {
        const active = (button.dataset.adminSelectValue || "") === value;
        button.classList.toggle("is-active", active);
        button.setAttribute("aria-checked", active ? "true" : "false");
        const check = button.querySelector(".admin-select__check");
        if (check) {
          check.hidden = !active;
        }
        if (active) {
          const labelNode = button.querySelector(".admin-select__label");
          selectedLabel = labelNode ? (labelNode.textContent || "") : "";
        }
      });

      const current = root.querySelector("[data-admin-select-current]");
      if (current) {
        current.textContent = selectedLabel;
      }
    });
  }

  function handleAdminSelectOption(button) {
    const root = button.closest("[data-admin-select]");
    if (!root) {
      return;
    }
    const value = button.dataset.adminSelectValue || "";
    const input = root.querySelector('input[type="hidden"][name]');
    if (!input) {
      return;
    }
    const previous = String(input.value || "");
    if (previous !== value) {
      input.value = value;
      input.dispatchEvent(new Event("change", { bubbles: true }));
      input.dispatchEvent(new Event("input", { bubbles: true }));
    }
    openAdminSelectMenuID = "";
    syncAdminSelectMenus();
  }

  function handleControlOption(button) {
    const kind = button.dataset.controlKind || "";
    const value = button.dataset.controlValue || "";
    if (!kind || !value) {
      return;
    }
    if (kind === "theme") {
      applyTheme(value);
      syncThemeSelect();
      closeControlMenu();
      return;
    }
    if (kind === "language") {
      applyLanguage(value);
      openControlMenu = "";
      renderRoute();
    }
  }

  async function handleDocumentSubmit(event) {
    if (event.target.id === "test-alert-form") {
      event.preventDefault();
      try {
        await submitTestAlert(event.target);
      } catch (error) {
        const result = document.getElementById("test-alert-result");
        if (result) {
          result.innerHTML = '<p>' + escapeHTML(error.message || "发送失败") + "</p>";
        }
      }
      return;
    }
    if (event.target.id === "admin-bootstrap-form") {
      event.preventDefault();
      await submitAdminBootstrap(event.target);
      return;
    }
    if (event.target.id === "admin-login-form") {
      event.preventDefault();
      await submitAdminLogin(event.target);
      return;
    }
    if (event.target.id === "admin-logout-form") {
      event.preventDefault();
      await submitAdminLogout();
      return;
    }
    if (event.target.id === "admin-password-form") {
      event.preventDefault();
      await submitAdminPassword(event.target);
      return;
    }
    if (event.target.id === "admin-check-form") {
      event.preventDefault();
      await submitAdminCheck(event.target);
      return;
    }
    if (event.target.id === "admin-node-name-form") {
      event.preventDefault();
      await submitAdminNodeName(event.target);
      return;
    }
  }

  function setAdminNotice(message, isError) {
    const node = document.getElementById("admin-notice");
    if (!node) {
      return;
    }
    node.className = "admin-notice" + (isError ? " is-error" : "");
    node.textContent = localizeText(message || "");
  }

  function setAdminCheckNotice(message, isError) {
    const node = document.getElementById("admin-check-notice");
    if (!node) {
      return;
    }
    node.className = "admin-notice" + (isError ? " is-error" : "");
    node.textContent = localizeText(message || "");
  }

  function setAdminNodeNotice(message, isError) {
    const node = document.getElementById("admin-node-notice");
    if (!node) {
      return;
    }
    node.className = "admin-notice" + (isError ? " is-error" : "");
    node.textContent = localizeText(message || "");
  }

  function setAdminMemberNotice(message, isError) {
    const node = document.getElementById("admin-member-notice");
    if (!node) {
      return;
    }
    node.className = "admin-notice" + (isError ? " is-error" : "");
    node.textContent = localizeText(message || "");
  }

  function handleAdminAction(button) {
    const action = button.dataset.adminAction || "";
    if (action === "edit-check") {
      fillAdminCheckForm(button.dataset.checkId || "");
      return;
    }
    if (action === "clear-check-form") {
      resetAdminCheckForm();
      return;
    }
    if (action === "delete-check") {
      deleteAdminCheck(button.dataset.checkId || "");
      return;
    }
    if (action === "edit-node-name") {
      fillAdminNodeNameForm(button.dataset.nodeId || "");
      return;
    }
    if (action === "clear-node-form") {
      resetAdminNodeNameForm();
      return;
    }
    if (action === "reset-node-name") {
      deleteAdminNodeName(button.dataset.nodeId || "");
      return;
    }
    if (action === "promote-member") {
      updateAdminMemberRole(button.dataset.nodeId || "", "voter");
      return;
    }
    if (action === "demote-member") {
      updateAdminMemberRole(button.dataset.nodeId || "", "nonvoter");
      return;
    }
    if (action === "remove-member") {
      deleteAdminMember(button.dataset.nodeId || "");
    }
  }

  function bindDirectFormSubmit(formID, handler) {
    const form = document.getElementById(formID);
    if (!form || form.dataset.boundSubmit === "true") {
      return;
    }
    form.addEventListener("submit", async (event) => {
      event.preventDefault();
      event.stopPropagation();
      await handler(form, event);
    });
    form.dataset.boundSubmit = "true";
  }

  function bindPageFormHandlers() {
    bindDirectFormSubmit("test-alert-form", async (form) => {
      try {
        await submitTestAlert(form);
      } catch (error) {
        const result = document.getElementById("test-alert-result");
        if (result) {
          result.innerHTML = '<p>' + escapeHTML(error.message || "发送失败") + "</p>";
        }
      }
    });
    bindDirectFormSubmit("admin-bootstrap-form", async (form) => {
      await submitAdminBootstrap(form);
    });
    bindDirectFormSubmit("admin-login-form", async (form) => {
      await submitAdminLogin(form);
    });
    bindDirectFormSubmit("admin-logout-form", async () => {
      await submitAdminLogout();
    });
    bindDirectFormSubmit("admin-password-form", async (form) => {
      await submitAdminPassword(form);
    });
    bindDirectFormSubmit("admin-check-form", async (form) => {
      await submitAdminCheck(form);
    });
    bindDirectFormSubmit("admin-node-name-form", async (form) => {
      await submitAdminNodeName(form);
    });
  }

  async function renderRoute(options) {
    const settings = options || {};
    window.clearTimeout(refreshTimer);
    const token = ++renderToken;
    const route = resolveRoute(window.location.pathname);
    const routeKey = buildRouteKey(route);
    const backgroundRefresh = Boolean(settings.backgroundRefresh);
    const shouldShowLoading = !backgroundRefresh && (!app.innerHTML || routeKey !== lastRouteKey);
    if (!backgroundRefresh && routeKey !== lastRouteKey) {
      sidebarDrawerOpen = false;
    }
    document.body.dataset.page = route.page;

    if (shouldShowLoading) {
      currentSearchItems = [];
      currentSearchQuery = "";
      searchPanelOpen = false;
      app.innerHTML = localizeMarkup(renderShell({
        page: route.page,
        title: routeTitle(route),
        localNodeID: "...",
        leaderID: "加载中",
        generatedAt: new Date().toISOString(),
        content: renderStatePanel("加载中", "正在从 API 拉取最新集群状态。", route)
      }));
      openControlMenu = "";
      openAdminSelectMenuID = "";
      syncThemeSelect();
      syncLanguageSelect();
      syncControlMenus();
      syncSidebarDrawer();
    }

    try {
      const view = await loadRoute(route);
      if (token !== renderToken) {
        return;
      }

      document.title = formatDocumentTitle(view.title);
      currentSearchItems = Array.isArray(view.searchIndex) ? view.searchIndex : [];
      if (!backgroundRefresh) {
        currentSearchQuery = "";
        searchPanelOpen = false;
      }
      app.innerHTML = localizeMarkup(renderShell(view));
      lastRouteKey = routeKey;
      openControlMenu = "";
      openAdminSelectMenuID = "";
      syncThemeSelect();
      syncLanguageSelect();
      syncControlMenus();
      syncSearchUI();
      bindPageFormHandlers();
      syncAdminCheckForm();
      syncAdminSelectMenus();
      syncAdminRowSelection();
      syncSidebarDrawer();
      scheduleRefresh();
    } catch (error) {
      if (token !== renderToken) {
        return;
      }

      if (backgroundRefresh && app.innerHTML && routeKey === lastRouteKey) {
        scheduleRefresh();
        return;
      }

      document.title = formatDocumentTitle(routeTitle(route));
      currentSearchItems = [];
      currentSearchQuery = "";
      searchPanelOpen = false;
      app.innerHTML = localizeMarkup(renderShell({
        page: route.page,
        title: routeTitle(route),
        localNodeID: "-",
        leaderID: "不可用",
        meta: activeMeta,
        generatedAt: new Date().toISOString(),
        content: renderStatePanel("加载失败", error.message || "接口请求失败。", route)
      }));
      lastRouteKey = routeKey;
      openControlMenu = "";
      openAdminSelectMenuID = "";
      bindPageFormHandlers();
      syncThemeSelect();
      syncLanguageSelect();
      syncControlMenus();
      syncSearchUI();
      syncSidebarDrawer();
      scheduleRefresh();
    }
  }

  function toggleSidebarDrawer(force) {
    sidebarDrawerOpen = typeof force === "boolean" ? force : !sidebarDrawerOpen;
    syncSidebarDrawer();
  }

  function syncSidebarDrawer() {
    document.body.dataset.sidebarOpen = sidebarDrawerOpen ? "true" : "false";
    document.body.classList.toggle("has-sidebar-open", sidebarDrawerOpen);
    document.querySelectorAll("[data-sidebar-toggle]").forEach((button) => {
      button.setAttribute("aria-expanded", sidebarDrawerOpen ? "true" : "false");
    });
  }

  function scheduleRefresh() {
    window.clearTimeout(refreshTimer);
    refreshTimer = window.setTimeout(() => {
      if (resolveRoute(window.location.pathname).page === "admin") {
        scheduleRefresh();
        return;
      }

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

  function getStoredLanguage() {
    try {
      return window.localStorage.getItem(LANGUAGE_KEY) || "zh";
    } catch (error) {
      return "zh";
    }
  }

  function applyLanguage(language) {
    const nextLanguage = LANGUAGES.some((item) => item.id === language) ? language : "zh";
    document.documentElement.lang = nextLanguage === "en" ? "en" : "zh-CN";
    document.documentElement.dataset.language = nextLanguage;
    try {
      window.localStorage.setItem(LANGUAGE_KEY, nextLanguage);
    } catch (error) {
      // Ignore storage failures.
    }
    return nextLanguage;
  }

  function getCurrentLanguage() {
    return document.documentElement.dataset.language || "zh";
  }

  function controlOptions(kind) {
    return kind === "theme" ? THEMES : LANGUAGES;
  }

  function currentControlOption(kind) {
    const currentValue = kind === "theme" ? getCurrentTheme() : getCurrentLanguage();
    return controlOptions(kind).find((option) => option.id === currentValue) || controlOptions(kind)[0];
  }

  function renderControlMenu(kind, surface, iconName, label) {
    const current = currentControlOption(kind);
    const menuID = kind + "-" + surface;
    const isOpen = openControlMenu === menuID;
    const options = controlOptions(kind);
    return `
      <div class="obs-theme-switcher${surface === "drawer" ? " obs-theme-switcher--drawer" : ""}${isOpen ? " is-open" : ""}" data-control-menu="${menuID}">
        <button
          type="button"
          class="obs-theme-switcher__trigger"
          data-control-trigger="${menuID}"
          aria-haspopup="menu"
          aria-expanded="${isOpen ? "true" : "false"}"
          aria-label="${escapeHTML(label)}"
        >
          ${renderIcon(iconName)}
          <span class="obs-theme-switcher__value" data-control-current="${kind}">${escapeHTML(current.label)}</span>
          <span class="obs-theme-switcher__chevron">${renderIcon("expand_more")}</span>
        </button>
        <div class="obs-theme-switcher__panel" role="menu" aria-label="${escapeHTML(label)}" aria-hidden="${isOpen ? "false" : "true"}">
          ${options
            .map((option) => {
              const active = option.id === current.id;
              return `
                <button
                  type="button"
                  class="obs-theme-switcher__option${active ? " is-active" : ""}"
                  role="menuitemradio"
                  aria-checked="${active ? "true" : "false"}"
                  data-control-option
                  data-control-kind="${kind}"
                  data-control-value="${option.id}"
                >
                  <span>${escapeHTML(option.label)}</span>
                  <span class="obs-theme-switcher__check"${active ? "" : " hidden"}>${renderIcon("check")}</span>
                </button>
              `;
            })
            .join("")}
        </div>
      </div>
    `;
  }

  function renderAdminSelect(name, options, initialValue, inputAttributes, label) {
    const list = Array.isArray(options)
      ? options
        .map((item) => Array.isArray(item) ? item : [item && item.value, item && item.label])
        .filter((item) => item && item.length >= 2 && item[0] !== undefined && item[0] !== null)
        .map((item) => [String(item[0]), String(item[1])])
      : [];

    let value = String(initialValue == null ? "" : initialValue);
    if (!list.some((item) => item[0] === value)) {
      value = list.length > 0 ? list[0][0] : "";
    }

    const selected = list.find((item) => item[0] === value);
    const selectedLabel = selected ? selected[1] : "";
    const defaultValue = list.length > 0 ? list[0][0] : "";
    const menuID = "admin-select-" + String(name || "");
    const isOpen = menuID === openAdminSelectMenuID;
    const extraAttributes = inputAttributes ? " " + inputAttributes : "";

    return `
      <div class="admin-select${isOpen ? " is-open" : ""}" data-admin-select data-admin-select-id="${escapeHTML(menuID)}">
        <input type="hidden" name="${escapeHTML(name || "")}" value="${escapeHTML(value)}" data-default-value="${escapeHTML(defaultValue)}"${extraAttributes}>
        <button
          type="button"
          class="admin-select__trigger"
          data-admin-select-trigger
          aria-haspopup="menu"
          aria-expanded="${isOpen ? "true" : "false"}"
          aria-label="${escapeHTML(label || String(name || ""))}"
        >
          <span class="admin-select__value" data-admin-select-current>${escapeHTML(selectedLabel)}</span>
          <span class="admin-select__chevron">${renderIcon("expand_more")}</span>
        </button>
        <div class="admin-select__panel" role="menu" aria-label="${escapeHTML(label || String(name || ""))}" aria-hidden="${isOpen ? "false" : "true"}">
          ${list
            .map((item) => {
              const active = item[0] === value;
              return `
                <button
                  type="button"
                  class="admin-select__option${active ? " is-active" : ""}"
                  role="menuitemradio"
                  aria-checked="${active ? "true" : "false"}"
                  data-admin-select-option
                  data-admin-select-value="${escapeHTML(item[0])}"
                >
                  <span class="admin-select__label">${escapeHTML(item[1])}</span>
                  <span class="admin-select__check"${active ? "" : " hidden"}>${renderIcon("check")}</span>
                </button>
              `;
            })
            .join("")}
        </div>
      </div>
    `;
  }

  function localizeText(text) {
    let output = String(text == null ? "" : text);
    const replacements = UI_TEXT[getCurrentLanguage()] || {};
    const entries = Object.entries(replacements).sort((a, b) => b[0].length - a[0].length);
    for (const [source, target] of entries) {
      output = output.split(source).join(target);
    }
    return output;
  }

  function localizeMarkup(markup) {
    return localizeText(markup);
  }

  function formatDocumentTitle(title) {
    return localizeText(title) + " · VPS Monitor";
  }

  function resolveRoute(pathname) {
    const normalized = pathname.replace(/\/+$/, "") || "/";
    if (normalized === "/") {
      return { page: "overview" };
    }
    if (normalized === "/admin") {
      return { page: "admin" };
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
      case "admin":
        return "管理";
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
      case "admin":
        return "events-page";
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
      case "admin":
        return loadAdminView();
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
    const [snapshot, meta] = await Promise.all([
      fetchJSON("/api/v1/cluster"),
      fetchJSON("/api/v1/meta").catch(() => ({ is_admin: false, admin_initialized: false, test_alert_channels: [] }))
    ]);
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
      localNodeName: nodeLabelFrom(snapshot.node_name, snapshot.node_id),
      currentNodeID: snapshot.node_id || "",
      currentNodeName: nodeLabelFrom(snapshot.node_name, snapshot.node_id),
      leaderID: snapshot.leader_id || "",
      leaderName: nodeLabelFrom(snapshot.leader_name, snapshot.leader_id),
      meta: meta,
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
    const [detail, snapshot, memHistory, diskHistory, meta] = await Promise.all([
      fetchJSON("/api/v1/nodes/" + encodeURIComponent(nodeID)),
      fetchJSON("/api/v1/cluster"),
      fetchHistory(nodeID, "mem_pct").catch(() => []),
      fetchHistory(nodeID, "disk_pct").catch(() => []),
      fetchJSON("/api/v1/meta").catch(() => ({ is_admin: false, admin_initialized: false, test_alert_channels: [] }))
    ]);

    return {
      page: "node",
      title: "节点 " + nodeLabelFrom(detail && detail.state && detail.state.node_name, nodeID),
      localNodeID: snapshot.node_id || "-",
      localNodeName: nodeLabelFrom(snapshot.node_name, snapshot.node_id),
      currentNodeID: nodeID,
      currentNodeName: nodeLabelFrom(detail && detail.state && detail.state.node_name, nodeID),
      leaderID: snapshot.leader_id || "",
      leaderName: nodeLabelFrom(snapshot.leader_name, snapshot.leader_id),
      meta: meta,
      generatedAt: snapshot.generated_at || new Date().toISOString(),
      content: renderNodePage(nodeID, detail, snapshot, memHistory, diskHistory, meta),
      searchIndex: buildSearchIndex({
        snapshot: snapshot,
        incidents: detail.incidents,
        events: snapshot.events,
        currentNodeID: nodeID,
        currentNodeName: nodeLabelFrom(detail && detail.state && detail.state.node_name, nodeID)
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
        admin_initialized: false,
        is_admin: false
      }))
    ]);

    return {
      page: "events",
      title: "事件",
      localNodeID: snapshot.node_id || meta.node_id || "-",
      localNodeName: nodeLabelFrom(snapshot.node_name || meta.node_name, snapshot.node_id || meta.node_id),
      currentNodeID: snapshot.node_id || meta.node_id || "",
      currentNodeName: nodeLabelFrom(snapshot.node_name || meta.node_name, snapshot.node_id || meta.node_id),
      leaderID: snapshot.leader_id || meta.leader_id || "",
      leaderName: nodeLabelFrom(snapshot.leader_name || meta.leader_name, snapshot.leader_id || meta.leader_id),
      meta: meta,
      generatedAt: snapshot.generated_at || new Date().toISOString(),
      content: renderEventsPage(snapshot, incidents, events, meta),
      searchIndex: buildSearchIndex({
        snapshot: snapshot,
        incidents: incidents,
        events: events
      })
    };
  }

  async function loadAdminView() {
    const meta = await fetchJSON("/api/v1/meta").catch(() => ({
      admin_initialized: false,
      is_admin: false,
      node_id: "-",
      node_name: "-",
      leader_id: "",
      leader_name: ""
    }));
    let checks = [];
    let nodes = [];
    let members = [];
    if (meta.is_admin) {
      [checks, nodes, members] = await Promise.all([
        fetchJSON("/api/v1/admin/checks").catch(() => []),
        fetchJSON("/api/v1/admin/nodes").catch(() => []),
        fetchJSON("/api/v1/admin/members").catch(() => [])
      ]);
    }
    currentAdminChecks = Array.isArray(checks) ? checks : [];
    currentAdminNodes = Array.isArray(nodes) ? nodes : [];
    currentAdminMembers = Array.isArray(members) ? members : [];

    return {
      page: "admin",
      title: meta.is_admin ? "管理后台" : "管理员登录",
      localNodeID: meta.node_id || "-",
      localNodeName: nodeLabelFrom(meta.node_name, meta.node_id),
      currentNodeID: meta.node_id || "",
      currentNodeName: nodeLabelFrom(meta.node_name, meta.node_id),
      leaderID: meta.leader_id || "",
      leaderName: nodeLabelFrom(meta.leader_name, meta.leader_id),
      meta: meta,
      generatedAt: new Date().toISOString(),
      content: renderAdminPage(meta, checks, nodes, members),
      searchIndex: buildSearchIndex({
        snapshot: { node_id: meta.node_id, node_name: meta.node_name, nodes: [], events: [], incidents: [] },
        incidents: [],
        events: []
      })
    };
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
            <h2>${snapshot.ingress && snapshot.ingress.active_node_id ? "入口当前指向 " + escapeHTML(ingressNodeLabel(snapshot.ingress)) : "入口正在等待新的 active node"}</h2>
            <p class="command-deck__lede">这个页面完全由前端自己消费 API 后拼出来。先看入口与 DNS，再看节点健康、活跃 incident 和互探矩阵。</p>
          </div>
          <div class="command-deck__stats">
            ${renderSummaryCard("Ingress 节点", snapshot.ingress && snapshot.ingress.active_node_id ? ingressNodeLabel(snapshot.ingress) : "待选举", "当前对外流量落点")}
            ${renderSummaryCard("DNS 同步", snapshot.ingress && snapshot.ingress.dns_synced ? "已同步" : "待同步", snapshot.ingress && snapshot.ingress.dns_synced_at ? timeAgo(snapshot.ingress.dns_synced_at) : "尚未同步")}
            ${renderSummaryCard("活跃 Incident", String(incidents.length), "当前需要处理的异常")}
            ${renderSummaryCard("Critical 节点", String(counts.critical), formatStableNodesSummary(counts.healthy))}
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
            ${renderSummaryCard("Leader", nodeLabelFrom(snapshot.leader_name, snapshot.leader_id) || "选举中", "当前决策节点")}
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
              <p>${formatServiceIssueSummary(serviceIssues)}</p>
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
              <p>${formatProbeObservationSummary(probes.length)}</p>
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
            ${renderSummaryCard("Ingress", snapshot.ingress && snapshot.ingress.active_node_id ? ingressNodeLabel(snapshot.ingress) : "待选举", "当前入口落点")}
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

  function renderAdminPage(meta, checks, nodes, members) {
    const list = Array.isArray(checks) ? checks : [];
    const nodeList = Array.isArray(nodes) ? nodes : [];
    const memberList = Array.isArray(members) ? members : [];
    const initialized = Boolean(meta && meta.admin_initialized);
    const isAdmin = Boolean(meta && meta.is_admin);

    if (!initialized) {
      return `
        <main class="obs-page obs-page--admin">
          <section class="obs-section admin-panel">
            <div class="obs-section__head">
              <div>
                <p class="obs-section__eyebrow">Bootstrap</p>
                <h2>设置管理员密码</h2>
              </div>
              <p>当前系统还没有管理员。第一次设置完成后，敏感信息和管理接口才会切到受保护状态。</p>
            </div>
            <form id="admin-bootstrap-form" class="admin-form">
              <label>
                <span>Password</span>
                <input type="password" name="password" minlength="8" required>
              </label>
              <button type="submit" class="admin-button">初始化管理员</button>
            </form>
            <p id="admin-notice" class="admin-notice"></p>
          </section>
        </main>
      `;
    }

    if (!isAdmin) {
      return `
        <main class="obs-page obs-page--admin">
          <section class="obs-section admin-panel">
            <div class="obs-section__head">
              <div>
                <p class="obs-section__eyebrow">Authentication</p>
                <h2>管理员登录</h2>
              </div>
              <p>登录后可查看敏感信息、发送测试告警，并管理运行时检测项。</p>
            </div>
            <form id="admin-login-form" class="admin-form">
              <label>
                <span>Password</span>
                <input type="password" name="password" minlength="8" required>
              </label>
              <button type="submit" class="admin-button">登录</button>
            </form>
            <p id="admin-notice" class="admin-notice"></p>
          </section>
        </main>
      `;
    }

    return `
      <main class="obs-page obs-page--admin">
        <section class="obs-split admin-top">
          <article class="obs-section admin-panel">
            <div class="obs-section__head">
              <div>
                <p class="obs-section__eyebrow">Session</p>
                <h2>管理员会话</h2>
              </div>
            </div>
            <div class="admin-actions">
              <form id="admin-logout-form">
                <button type="submit" class="admin-button admin-button--secondary">退出登录</button>
              </form>
            </div>
            <p id="admin-notice" class="admin-notice"></p>
          </article>

          <article class="obs-section admin-panel">
            <div class="obs-section__head">
              <div>
                <p class="obs-section__eyebrow">Password</p>
                <h2>修改管理员密码</h2>
              </div>
            </div>
            <form id="admin-password-form" class="admin-form">
              <label>
                <span>Current Password</span>
                <input type="password" name="current_password" minlength="8" required>
              </label>
              <label>
                <span>New Password</span>
                <input type="password" name="new_password" minlength="8" required>
              </label>
              <button type="submit" class="admin-button">更新密码</button>
            </form>
          </article>
        </section>

        <section class="obs-split admin-content">
          <article class="obs-section admin-panel">
            <div class="obs-section__head">
              <div>
                <p class="obs-section__eyebrow">Runtime Checks</p>
                <h2>检测项编辑器</h2>
              </div>
              <p>支持 <code>systemd</code>、<code>docker</code>、<code>http</code>、<code>tcp</code>，保存后下一轮采集立即生效。</p>
            </div>
            <form id="admin-check-form" class="admin-form admin-form--grid">
              <input type="hidden" name="id">
              <label>
                <span>Name</span>
                <input type="text" name="name" required>
              </label>
              <label>
                <span>Type</span>
                ${renderAdminSelect(
                  "type",
                  [
                    ["systemd", "systemd"],
                    ["docker", "docker"],
                    ["http", "http"],
                    ["tcp", "tcp"]
                  ],
                  "systemd",
                  "data-check-type",
                  "Type"
                )}
              </label>
              <label>
                <span>Node Scope</span>
                ${renderAdminSelect(
                  "scope_mode",
                  [
                    ["all", "All nodes"],
                    ["include_nodes", "Only selected nodes"],
                    ["exclude_nodes", "All except selected nodes"]
                  ],
                  "all",
                  "data-check-scope",
                  "Node Scope"
                )}
              </label>
              <label class="admin-check-field" data-field="service_name">
                <span>Service Name</span>
                <input type="text" name="service_name">
              </label>
              <label class="admin-check-field" data-field="container_name">
                <span>Container Name</span>
                <input type="text" name="container_name">
              </label>
              <label class="admin-check-field" data-field="scheme">
                <span>Scheme</span>
                ${renderAdminSelect(
                  "scheme",
                  [
                    ["http", "http"],
                    ["https", "https"]
                  ],
                  "http",
                  "",
                  "Scheme"
                )}
              </label>
              <label class="admin-check-field" data-field="host_mode">
                <span>Host Mode</span>
                ${renderAdminSelect(
                  "host_mode",
                  [
                    ["peer", "peer"],
                    ["local", "local"]
                  ],
                  "peer",
                  "",
                  "Host Mode"
                )}
              </label>
              <label class="admin-check-field" data-field="port">
                <span>Port</span>
                <input type="number" name="port" min="1" max="65535">
              </label>
              <label class="admin-check-field" data-field="path">
                <span>Path</span>
                <input type="text" name="path" placeholder="/">
              </label>
              <label class="admin-check-field" data-field="expect_status">
                <span>Expect Status</span>
                <input type="number" name="expect_status" min="100" max="599" placeholder="200">
              </label>
              <label class="admin-check-field" data-field="timeout">
                <span>Timeout</span>
                <input type="text" name="timeout" placeholder="3s">
              </label>
              <label class="admin-check-field" data-field="label">
                <span>Label</span>
                <input type="text" name="label">
              </label>
              <fieldset class="admin-check-field admin-check-scope-field" data-field="node_ids">
                <legend>Target Nodes</legend>
                <div class="admin-node-scope-list">
                  ${nodeList.map((node) => `
                    <label class="admin-node-scope-item">
                      <input type="checkbox" name="node_ids" value="${escapeHTML(node.node_id || "")}">
                      <span class="admin-node-scope-item__box" aria-hidden="true"></span>
                      <span class="admin-node-scope-item__label">${escapeHTML(formatAdminNodeOption(node))}</span>
                    </label>
                  `).join("")}
                </div>
              </fieldset>
              <label class="admin-toggle">
                <input type="checkbox" name="enabled" checked>
                <span class="admin-toggle__box" aria-hidden="true"></span>
                <span class="admin-toggle__text">
                  <strong>Enabled</strong>
                  <small>Join the next collection cycle immediately</small>
                </span>
              </label>
              <div class="admin-form__actions">
                <button type="submit" class="admin-button">保存检测项</button>
                <button type="button" class="admin-button admin-button--secondary" data-admin-action="clear-check-form">清空</button>
              </div>
            </form>
            <p id="admin-check-notice" class="admin-notice"></p>
          </article>

          <article class="obs-section admin-panel">
            <div class="obs-section__head">
              <div>
                <p class="obs-section__eyebrow">Current Checks</p>
                <h2>已生效检测项</h2>
              </div>
            </div>
            <div class="admin-check-list">
              ${list.length > 0
                ? list.map((check) => renderAdminCheckRow(check)).join("")
                : emptyRune("No checks yet", "Create the first runtime check to replace static monitor.yaml service lists.")}
            </div>
          </article>
        </section>

        <section class="obs-split admin-content">
          <article class="obs-section admin-panel">
            <div class="obs-section__head">
              <div>
                <p class="obs-section__eyebrow">Node Names</p>
                <h2>节点显示名称</h2>
              </div>
              <p>这里只改页面和 API 的显示名，不会修改 <code>node_id</code>、Raft 身份或路由。留空可恢复默认名称。</p>
            </div>
            <form id="admin-node-name-form" class="admin-form admin-form--grid">
              <label>
                <span>Node</span>
                ${renderAdminSelect(
                  "node_id",
                  nodeList.map((node) => [node.node_id || "", formatAdminNodeOption(node)]),
                  nodeList.length > 0 ? (nodeList[0].node_id || "") : "",
                  "",
                  "Node"
                )}
              </label>
              <label>
                <span>Display Name</span>
                <input type="text" name="display_name" maxlength="80" placeholder="留空则恢复默认名称">
              </label>
              <div class="admin-form__actions">
                <button type="submit" class="admin-button">保存节点名称</button>
                <button type="button" class="admin-button admin-button--secondary" data-admin-action="clear-node-form">清空</button>
              </div>
            </form>
            <p id="admin-node-notice" class="admin-notice"></p>
          </article>

          <article class="obs-section admin-panel">
            <div class="obs-section__head">
              <div>
                <p class="obs-section__eyebrow">Current Names</p>
                <h2>当前节点名称映射</h2>
              </div>
            </div>
            <div class="admin-check-list">
              ${nodeList.length > 0
                ? nodeList.map((node) => renderAdminNodeNameRow(node)).join("")
                : emptyRune("No nodes configured", "Cluster peers will appear here after configuration loads.")}
            </div>
          </article>
        </section>

        <section class="obs-section admin-panel">
          <div class="obs-section__head">
            <div>
              <p class="obs-section__eyebrow">Cluster Membership</p>
              <h2>集群成员</h2>
            </div>
            <p>这里直接管理运行时成员目录和 Raft 角色。新节点 auto-join 后会自动出现在这里，不需要再手改其它节点的 <code>cluster.peers</code>。</p>
          </div>
          <div class="admin-check-list">
            ${memberList.length > 0
              ? memberList.map((member) => renderAdminMemberRow(member, meta)).join("")
              : emptyRune("还没有成员数据", "如果你刚完成动态模式升级，等 leader 完成第一次成员目录同步后这里会出现。")}
          </div>
          <p id="admin-member-notice" class="admin-notice"></p>
        </section>
      </main>
    `;
  }

  function renderAdminCheckRow(check) {
    return `
      <article class="service-row status-surface admin-check-row" data-status="${check.enabled ? "healthy" : "unknown"}" data-check-id="${escapeHTML(check.id || "")}">
        <div class="admin-check-row__head">
          <div>
            <strong>${escapeHTML(check.name || check.type || "check")}</strong>
            <span>${escapeHTML((check.type || "check") + " · " + adminCheckEnabledLabel(check.enabled !== false))}</span>
          </div>
          <div class="admin-check-row__actions">
            <button type="button" class="admin-button admin-button--secondary" data-admin-action="edit-check" data-check-id="${escapeHTML(check.id || "")}">编辑</button>
            <button type="button" class="admin-button admin-button--danger" data-admin-action="delete-check" data-check-id="${escapeHTML(check.id || "")}">删除</button>
          </div>
        </div>
        <small>${escapeHTML(describeAdminCheck(check))}</small>
      </article>
    `;
  }

  function renderAdminNodeNameRow(node) {
    return `
      <article class="service-row status-surface admin-check-row" data-status="healthy" data-node-id="${escapeHTML(node.node_id || "")}">
        <div class="admin-check-row__head">
          <div>
            <strong>${escapeHTML(node.effective_display_name || node.node_id || "-")}</strong>
            <span>${escapeHTML(node.node_id || "-")}</span>
          </div>
          <div class="admin-check-row__actions">
            <button type="button" class="admin-button admin-button--secondary" data-admin-action="edit-node-name" data-node-id="${escapeHTML(node.node_id || "")}">编辑</button>
            ${node.display_name
              ? `<button type="button" class="admin-button admin-button--secondary" data-admin-action="reset-node-name" data-node-id="${escapeHTML(node.node_id || "")}">恢复默认</button>`
              : ""}
          </div>
        </div>
        <small>${escapeHTML(describeAdminNodeName(node))}</small>
      </article>
    `;
  }

  function renderAdminMemberRow(member, meta) {
    const nodeID = member && member.node_id ? member.node_id : "";
    const currentRole = member && member.current_role ? member.current_role : (member && member.desired_role ? member.desired_role : "voter");
    const canPromote = member && member.status === "active" && currentRole !== "voter";
    const canDemote = member && member.status === "active" && currentRole === "voter";
    const canRemove = member && member.status === "active";
    const isCurrentNode = nodeID && meta && nodeID === meta.node_id;

    return `
      <article class="service-row status-surface admin-check-row admin-member-row" data-status="${escapeHTML(adminMemberStatusTone(member))}" data-node-id="${escapeHTML(nodeID)}">
        <div class="admin-check-row__head">
          <div>
            <strong>${escapeHTML(member && member.effective_display_name ? member.effective_display_name : nodeID || "-")}</strong>
            <span>${escapeHTML(nodeID || "-")} · ${escapeHTML(adminMemberBadgeLine(member, meta))}</span>
          </div>
          <div class="admin-check-row__actions">
            ${canPromote
              ? `<button type="button" class="admin-button admin-button--secondary" data-admin-action="promote-member" data-node-id="${escapeHTML(nodeID)}">升为 voter</button>`
              : ""}
            ${canDemote
              ? `<button type="button" class="admin-button admin-button--secondary" data-admin-action="demote-member" data-node-id="${escapeHTML(nodeID)}">降为 nonvoter</button>`
              : ""}
            ${canRemove
              ? `<button type="button" class="admin-button admin-button--danger" data-admin-action="remove-member" data-node-id="${escapeHTML(nodeID)}">${escapeHTML(isCurrentNode ? "移除当前节点" : "移除节点")}</button>`
              : ""}
          </div>
        </div>
        <div class="admin-member-row__facts">
          <span class="admin-member-chip">${escapeHTML("role " + currentRole)}</span>
          <span class="admin-member-chip">${escapeHTML("desired " + (member && member.desired_role ? member.desired_role : currentRole))}</span>
          <span class="admin-member-chip">${escapeHTML("health " + adminMemberHealthLabel(member))}</span>
          <span class="admin-member-chip">${escapeHTML("heartbeat " + adminMemberHeartbeatLabel(member))}</span>
        </div>
        <small>${escapeHTML(describeAdminMember(member))}</small>
      </article>
    `;
  }

  function formatAdminNodeOption(node) {
    const effective = node && node.effective_display_name ? node.effective_display_name : nodeLabelFrom("", node && node.node_id);
    const nodeID = node && node.node_id ? node.node_id : "-";
    return effective === nodeID ? nodeID : effective + " (" + nodeID + ")";
  }

  function adminCheckEnabledLabel(enabled) {
    return getCurrentLanguage() === "en"
      ? (enabled ? "enabled" : "disabled")
      : (enabled ? "启用" : "禁用");
  }

  function describeAdminNodeName(node) {
    if (!node) {
      return "";
    }
    if (node.display_name) {
      return getCurrentLanguage() === "en"
        ? "runtime override · " + (node.config_display_name ? "config " + node.config_display_name + " -> " : "") + (node.effective_display_name || node.display_name)
        : "运行时覆盖 · " + (node.config_display_name ? "配置 " + node.config_display_name + " -> " : "") + (node.effective_display_name || node.display_name);
    }
    if (node.config_display_name) {
      return getCurrentLanguage() === "en"
        ? "using config display_name · " + node.config_display_name
        : "使用配置 display_name · " + node.config_display_name;
    }
    return getCurrentLanguage() === "en"
      ? "using node_id as the default display name"
      : "使用 node_id 作为默认显示名称";
  }

  function adminMemberStatusTone(member) {
    if (!member || member.status !== "active") {
      return "unknown";
    }
    return normalizeStatus(member.health_status || "unknown");
  }

  function adminMemberBadgeLine(member, meta) {
    const parts = [];
    if (member && member.is_leader) {
      parts.push("leader");
    }
    if (member && member.status) {
      parts.push(member.status);
    }
    if (member && member.current_role) {
      parts.push(member.current_role);
    } else if (member && member.desired_role) {
      parts.push(member.desired_role);
    }
    if (member && meta && member.node_id === meta.node_id) {
      parts.push("self");
    }
    return parts.join(" · ");
  }

  function adminMemberHealthLabel(member) {
    if (!member || !member.health_status) {
      return getCurrentLanguage() === "en" ? "unknown" : "未知";
    }
    return statusLabel(member.health_status);
  }

  function adminMemberHeartbeatLabel(member) {
    if (!member || !member.last_heartbeat_at) {
      return getCurrentLanguage() === "en" ? "no signal" : "无信号";
    }
    return timeAgo(member.last_heartbeat_at);
  }

  function describeAdminMember(member) {
    if (!member) {
      return "";
    }
    const parts = [
      member.api_addr ? "API " + member.api_addr : "",
      member.raft_addr ? "Raft " + member.raft_addr : "",
      member.public_ipv4 ? "IP " + member.public_ipv4 : "",
      typeof member.priority === "number" ? "priority " + String(member.priority) : "",
      member.ingress_candidate === false ? "ingress disabled" : "ingress candidate"
    ].filter(Boolean);
    if (member.updated_at) {
      parts.push((getCurrentLanguage() === "en" ? "updated " : "更新于 ") + formatDateTime(member.updated_at));
    }
    if (member.removed_at) {
      parts.push((getCurrentLanguage() === "en" ? "removed " : "移除于 ") + formatDateTime(member.removed_at));
    }
    return parts.join(" · ");
  }

  function describeAdminCheck(check) {
    if (!check) {
      return "";
    }
    const scope = describeAdminCheckScope(check);
    if (check.type === "systemd") {
      return [check.service_name || "", scope].filter(Boolean).join(" · ");
    }
    if (check.type === "docker") {
      return [check.container_name || "", scope].filter(Boolean).join(" · ");
    }
    if (check.type === "http") {
      return [
        (check.host_mode || "peer") + " · " + (check.scheme || "http") + "://" + ":" + (check.port || "-") + (check.path || "/"),
        scope
      ].filter(Boolean).join(" · ");
    }
    if (check.type === "tcp") {
      return ["peer tcp :" + (check.port || "-"), scope].filter(Boolean).join(" · ");
    }
    return "";
  }

  function describeAdminCheckScope(check) {
    if (!check) {
      return "";
    }
    const nodeIDs = Array.isArray(check.node_ids) ? check.node_ids.filter(Boolean) : [];
    if (check.scope_mode === "include_nodes") {
      return nodeIDs.length > 0
        ? (getCurrentLanguage() === "en" ? "only " : "仅 ") + nodeIDs.join(", ")
        : (getCurrentLanguage() === "en" ? "no nodes selected" : "未选择节点");
    }
    if (check.scope_mode === "exclude_nodes") {
      return nodeIDs.length > 0
        ? (getCurrentLanguage() === "en" ? "except " : "排除 ") + nodeIDs.join(", ")
        : (getCurrentLanguage() === "en" ? "all nodes" : "全部节点");
    }
    return getCurrentLanguage() === "en" ? "all nodes" : "全部节点";
  }

  function renderNodeCard(node, history) {
    const summary = node.last_probe_summary || {};
    const loadLabel = Number.isFinite(Number(node.load1)) ? Number(node.load1).toFixed(2) : "-";

    return `
      <article class="node-card status-surface" data-status="${normalizeStatus(node.status)}">
        <div class="node-card__header">
          <div>
            <p class="eyebrow">Node ${escapeHTML(nodeLabel(node))}</p>
            <h3>${escapeHTML(nodeLabel(node))}</h3>
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
                  <span>${escapeHTML(nodeLabel(target))}</span>
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
                <span>${escapeHTML(nodeLabel(target))}</span>
                <strong>${label}</strong>
              </div>
            `;
          })
          .join("");

        return `
          <div class="matrix__row">
            <div class="matrix__label">${escapeHTML(nodeLabel(source))}</div>
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
          ? nodeLabelFrom(incident.node_name, incident.node_id) + " · " + (incident.rule_key || "incident")
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
          rightTop: nodeLabelFrom(event.node_name, event.node_id),
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
            <strong>${escapeHTML(probePathLabel(probe))}</strong>
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
    activeMeta = view.meta || {};
    const localNodeHref = isUsableNodeID(view.localNodeID)
      ? "/nodes/" + encodeURIComponent(view.localNodeID)
      : "";
    const contextNodeID = view.currentNodeID || "";
    const contextNodeName = view.currentNodeName || contextNodeID || "";
    const contextNodeHref = isUsableNodeID(contextNodeID)
      ? "/nodes/" + encodeURIComponent(contextNodeID)
      : localNodeHref;
    const headingLabel = view.page === "node"
      ? "Node Detail"
      : view.page === "events"
        ? "Incident Center"
        : view.page === "admin"
          ? "Admin Control"
          : "Command Center";
    const headingTitle = view.page === "node" && contextNodeID
      ? contextNodeName
      : sidebarContextLabel(view.page);
    const adminLabel = activeMeta.is_admin ? "Admin" : "Login";
    const filingLink = '<a class="obs-filing-link" href="https://icp.gov.moe/?keyword=20268668" target="_blank" rel="noopener noreferrer">萌ICP备20268668号</a>';

    return `
      <div class="obs-shell">
        <button type="button" class="obs-sidebar-backdrop" data-sidebar-dismiss aria-label="Close navigation"></button>
        <aside class="obs-sidebar">
          <div class="obs-sidebar__brand">
            <div>
              <h1>Obsidian</h1>
              <p>vps-monitor</p>
            </div>
            <button type="button" class="obs-sidebar__close" data-sidebar-dismiss aria-label="Close navigation">
              ${renderIcon("close")}
            </button>
            <span class="obs-sidebar__context">${escapeHTML(sidebarContextLabel(view.page))}</span>
          </div>

          <nav class="obs-sidebar__nav">
            ${renderObsidianNavLink("/", "dashboard", "Observatory", view.page === "overview")}
            ${localNodeHref ? renderObsidianNavLink(localNodeHref, "dns", "Local Node", view.page === "node" && contextNodeID === view.localNodeID) : ""}
            ${contextNodeHref && contextNodeID && contextNodeID !== view.localNodeID
              ? renderObsidianNavLink(contextNodeHref, "lan", contextNodeName, view.page === "node")
              : ""}
            ${renderObsidianNavLink("/events", "warning", "Incidents", view.page === "events")}
            ${renderObsidianNavLink("/admin", activeMeta.is_admin ? "shield_lock" : "login", adminLabel, view.page === "admin")}
          </nav>

          <div class="obs-sidebar__footer">
            ${localNodeHref
              ? `<a href="${localNodeHref}" data-link class="obs-sidebar__cta">${renderIcon("rocket_launch")}<span>Open Local Node</span></a>`
              : ""}
            <div class="obs-sidebar__status">
              <div class="obs-sidebar__status-item">
                <span>Current Node</span>
                <strong>${escapeHTML(view.localNodeName || view.localNodeID || "-")}</strong>
              </div>
              <div class="obs-sidebar__status-item">
                <span>Leader</span>
                <strong>${escapeHTML(view.leaderName || view.leaderID || "Electing")}</strong>
              </div>
            </div>
            <div class="obs-sidebar__links">
              <span>Self-hosted control console</span>
              <span>${escapeHTML(formatDateTime(view.generatedAt))}</span>
              <span class="obs-sidebar__filing">${filingLink}</span>
            </div>
            <div class="obs-sidebar__mobile-tools">
              ${renderControlMenu("language", "drawer", "translate", "Language")}
              ${renderControlMenu("theme", "drawer", "palette", "Theme")}
              <div class="obs-topbar__operator obs-topbar__operator--drawer">
                <span>${escapeHTML(activeMeta.is_admin ? "Admin Session" : "Cluster Leader")}</span>
                <strong>${escapeHTML(activeMeta.is_admin ? "Authorized" : (view.leaderName || view.leaderID || "Electing"))}</strong>
              </div>
            </div>
          </div>
        </aside>

        <header class="obs-topbar">
          <div class="obs-topbar__leading">
            <button type="button" class="obs-sidebar-toggle" data-sidebar-toggle aria-label="Open navigation">
              ${renderIcon("menu")}
            </button>
            <div class="obs-topbar__heading">
              <span>${escapeHTML(headingLabel)}</span>
              <strong>${escapeHTML(headingTitle)}</strong>
            </div>
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
            ${renderControlMenu("language", "topbar", "translate", "Language")}
            ${renderControlMenu("theme", "topbar", "palette", "Theme")}
            <div class="obs-topbar__operator">
              <span>${escapeHTML(activeMeta.is_admin ? "Admin Session" : "Cluster Leader")}</span>
              <strong>${escapeHTML(activeMeta.is_admin ? "Authorized" : (view.leaderName || view.leaderID || "Electing"))}</strong>
            </div>
          </div>
        </header>

        <div class="obs-main">
          ${view.content}
          <footer class="obs-main__footer">
            ${filingLink}
          </footer>
        </div>
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
    const leaderLabel = nodeLabelFrom(snapshot.leader_name, snapshot.leader_id);

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
                <strong>${escapeHTML(leaderLabel || "Electing")}</strong>
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
            <p>${escapeHTML(formatFleetSummary(nodes.length, counts.healthy, activeIncidents))}</p>
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
    const nodeName = nodeLabelFrom(state.node_name, nodeID);
    const leaderLabel = nodeLabelFrom(snapshot.leader_name, snapshot.leader_id);

    return `
      <main class="obs-page obs-page--node">
        <section class="obs-node-top">
          <div class="obs-node-top__copy">
            <p class="obs-kicker">
              <span class="obs-live-dot" data-status="${tone}"></span>
              ${escapeHTML(nodeName)} · ${escapeHTML(nodeRoleLabel(snapshot, nodeID))}
            </p>
            <h1>${escapeHTML(nodeName)}</h1>
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
            <h2>${escapeHTML(nodeName)}</h2>
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
              ${renderObsidianInfoCard("Leader", leaderLabel || "Electing")}
              ${renderObsidianInfoCard("Heartbeat", timeAgo(state.last_heartbeat_at))}
              ${renderObsidianInfoCard("Peer Reach", (state.last_probe_summary.successful_peers || 0) + "/" + (state.last_probe_summary.total_peers || 0))}
            </div>
          </article>
        </section>

        ${renderHardwareSpecs(detail.heartbeat)}

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
                  <span>${escapeHTML(nodeLabel(target))}</span>
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
                <span>${escapeHTML(nodeLabel(target))}</span>
                <strong>${label}</strong>
              </div>
            `;
          })
          .join("");

        return `
          <div class="matrix__row" style="grid-template-columns: 96px repeat(${nodes.length}, minmax(0, 1fr));">
            <div class="matrix__label">${escapeHTML(nodeLabel(source))}</div>
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
            ? nodeLabelFrom(incident.node_name, incident.node_id) + " · " + (incident.rule_key || "incident")
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
          rightTop: nodeLabelFrom(event.node_name, event.node_id),
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
      return emptyRune("No runtime checks", "Add systemd, docker, http, or tcp checks from the admin page.");
    }

    return list
      .map(
        (service) => `
          <div class="service-row status-surface" data-status="${normalizeStatus(service.status)}">
            <div>
              <strong>${escapeHTML(service.name || "service")}</strong>
              <span>${escapeHTML((service.type || "check") + " · " + (service.status || "unknown"))}</span>
            </div>
            <small>${escapeHTML((service.target ? service.target + " · " : "") + (service.detail || "No detail"))}</small>
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
            <strong>${escapeHTML(probePathLabel(probe))}</strong>
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
            return `<i title="${escapeHTML(nodeLabel(node))}" style="height:${Math.min(100, value)}%"></i>`;
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
              <strong>${escapeHTML(nodeLabel(node))}</strong>
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
            ${renderObsidianInfoCard("Node", nodeLabelFrom(incident.node_name, incident.node_id))}
            ${renderObsidianInfoCard("Rule", incident.rule_key || "-")}
          </div>
        </div>
      </section>
    `;
  }

  function renderObsidianIncidentCard(incident) {
    return `
      <article class="obs-incident-card" data-status="${normalizeStatus(incident.severity)}">
        <p class="obs-section__eyebrow">${escapeHTML((incident.status || "active") + " · " + nodeLabelFrom(incident.node_name, incident.node_id))}</p>
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
            <span>${escapeHTML(nodeLabelFrom(event.node_name, event.node_id))}</span>
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
                <span>${escapeHTML(nodeLabelFrom(incident.node_name, incident.node_id))}</span>
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
          <span class="obs-log-line__body">${escapeHTML(probePathLabel(probe) + " · 22 " + (probe.tcp_22_ok ? "OPEN" : "DROP") + " · 443 " + (probe.tcp_443_ok ? "OPEN" : "DROP") + " · HTTP " + (probe.http_ok ? "OK" : "MISS"))}</span>
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

  function renderHardwareSpecs(heartbeat) {
    if (!heartbeat || (!heartbeat.cpu_model && !heartbeat.cpu_cores)) return "";
    const memGB = heartbeat.mem_total_mb ? (heartbeat.mem_total_mb / 1024).toFixed(1) : "-";
    const diskGB = heartbeat.disk_total_mb ? (heartbeat.disk_total_mb / 1024).toFixed(1) : "-";
    return `
      <section class="obs-section obs-hw-specs">
        <div class="obs-section__head">
          <div>
            <p class="obs-section__eyebrow">Hardware Profile</p>
            <h2>Machine Specifications</h2>
          </div>
        </div>
        <div class="obs-info-grid obs-info-grid--3col">
          ${renderObsidianInfoCard("CPU", escapeHTML(heartbeat.cpu_model || "-"))}
          ${renderObsidianInfoCard("Cores", String(heartbeat.cpu_cores || "-"))}
          ${renderObsidianInfoCard("Memory", memGB + " GB")}
          ${renderObsidianInfoCard("Disk", diskGB + " GB")}
          ${renderObsidianInfoCard("OS", escapeHTML(heartbeat.os || "-"))}
          ${renderObsidianInfoCard("Kernel", escapeHTML(heartbeat.kernel || "-"))}
        </div>
      </section>
    `;
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
    add({
      kind: "View",
      title: activeMeta.is_admin ? "Admin" : "Login",
      subtitle: activeMeta.is_admin ? "Runtime check management" : "Administrator authentication",
      href: "/admin",
      icon: activeMeta.is_admin ? "shield_lock" : "login",
      pinned: true,
      keywords: "admin login password checks"
    });

    if (isUsableNodeID(snapshot.node_id)) {
      add({
        kind: "View",
        title: "Local Node",
        subtitle: nodeLabelFrom(snapshot.node_name, snapshot.node_id),
        href: "/nodes/" + encodeURIComponent(snapshot.node_id),
        icon: "dns",
        pinned: true,
        keywords: "local current node " + nodeLabelFrom(snapshot.node_name, snapshot.node_id) + " " + snapshot.node_id
      });
    }

    if (isUsableNodeID(currentNodeID) && currentNodeID !== snapshot.node_id) {
      add({
        kind: "View",
        title: "Current Node",
        subtitle: nodeLabelFrom(options && options.currentNodeName, currentNodeID),
        href: "/nodes/" + encodeURIComponent(currentNodeID),
        icon: "lan",
        pinned: true,
        keywords: "current node detail " + nodeLabelFrom(options && options.currentNodeName, currentNodeID) + " " + currentNodeID
      });
    }

    nodes.forEach((node) => {
      const probeSummary = node.last_probe_summary || {};
      add({
        kind: "Node",
        title: nodeLabel(node),
        subtitle: truncate(node.reason || statusLabel(node.status), 88),
        href: "/nodes/" + encodeURIComponent(node.node_id || ""),
        icon: "dns",
        status: normalizeStatus(node.status),
        keywords: [
          node.node_name,
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
        subtitle: nodeLabelFrom(incident.node_name, incident.node_id) + " · " + (incident.rule_key || "incident"),
        href: "/events",
        icon: "warning",
        status: normalizeStatus(incident.severity || incident.status),
        keywords: [
          incident.id,
          incident.node_name,
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
        subtitle: (event.kind || "event") + " · " + nodeLabelFrom(event.node_name, event.node_id),
        href: "/events",
        icon: "history",
        status: normalizeStatus(event.severity),
        keywords: [
          event.kind,
          event.node_name,
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
        word: getCurrentLanguage() === "en" ? "WAITING" : "等待",
        description: getCurrentLanguage() === "en"
          ? "The dashboard is ready. It will settle once the first node reports telemetry."
          : "面板已经准备好，等第一台节点上报遥测后就会稳定下来。"
      };
    }
    const total = nodes.length;
    if (counts.critical > 0 || incidents.some((incident) => normalizeStatus(incident.severity) === "critical")) {
      return {
        tone: "critical",
        word: getCurrentLanguage() === "en" ? "ALERT" : "告警",
        description: getCurrentLanguage() === "en"
          ? "Critical nodes or incidents are open. Start with the failing node, then verify ingress and peer visibility."
          : "存在严重节点或未恢复事件。先处理异常节点，再确认 ingress 与互探可见性。"
      };
    }
    if (counts.degraded > 0 || activeIncidentCount(incidents) > 0) {
      return {
        tone: "degraded",
        word: getCurrentLanguage() === "en" ? "DEGRADED" : "降级",
        description: getCurrentLanguage() === "en"
          ? "The cluster is still serving, but at least one layer is thinning. Review degraded nodes before the next transition escalates."
          : "集群仍在服务，但至少有一层正在变薄。请在下一次升级前先检查降级节点。"
      };
    }
    return {
      tone: "healthy",
      word: getCurrentLanguage() === "en" ? "STABLE" : "稳定",
      description: getCurrentLanguage() === "en"
        ? "All " + total + " nodes are within expected thresholds. No active incidents are diluting operator attention."
        : "全部 " + total + " 个节点都在预期阈值内，目前没有活跃事件分散注意力。"
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
        selected = { nodeID: nodeLabel(node), nodeName: nodeLabel(node), value: value };
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

  function nodeLabelFrom(name, id) {
    const label = String(name || id || "").trim();
    return label || "-";
  }

  function nodeLabel(node) {
    return nodeLabelFrom(node && node.node_name, node && node.node_id);
  }

  function ingressNodeLabel(ingress) {
    return nodeLabelFrom(ingress && ingress.active_node_name, ingress && ingress.active_node_id);
  }

  function probePathLabel(probe) {
    return nodeLabelFrom(probe && probe.source_node_name, probe && probe.source_node_id) +
      " → " +
      nodeLabelFrom(probe && probe.target_node_name, probe && probe.target_node_id);
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
      return getCurrentLanguage() === "en" ? "0h" : "0小时";
    }
    const hours = Math.floor(seconds / 3600);
    if (hours < 24) {
      return getCurrentLanguage() === "en" ? hours + "h" : hours + "小时";
    }
    const days = Math.floor(hours / 24);
    return getCurrentLanguage() === "en" ? days + "d" : days + "天";
  }

  function formatUptimeLong(value) {
    const seconds = Number(value);
    if (!Number.isFinite(seconds) || seconds <= 0) {
      return getCurrentLanguage() === "en" ? "0m" : "0分";
    }
    const days = Math.floor(seconds / 86400);
    const hours = Math.floor((seconds % 86400) / 3600);
    const minutes = Math.floor((seconds % 3600) / 60);
    if (days > 0) {
      return getCurrentLanguage() === "en"
        ? days + "d " + hours + "h " + minutes + "m"
        : days + "天 " + hours + "小时 " + minutes + "分";
    }
    if (hours > 0) {
      return getCurrentLanguage() === "en"
        ? hours + "h " + minutes + "m"
        : hours + "小时 " + minutes + "分";
    }
    return getCurrentLanguage() === "en" ? minutes + "m" : minutes + "分";
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
      return getCurrentLanguage() === "en" ? minutes + "m" : minutes + "分";
    }
    const hours = Math.floor(minutes / 60);
    const restMinutes = minutes % 60;
    return getCurrentLanguage() === "en"
      ? hours + "h " + restMinutes + "m"
      : hours + "小时 " + restMinutes + "分";
  }

  function sidebarContextLabel(page) {
    switch (page) {
      case "admin":
        return "Admin Control";
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
        return getCurrentLanguage() === "en" ? "Critical Incident" : "严重事件";
      case "degraded":
        return getCurrentLanguage() === "en" ? "Degraded Incident" : "降级事件";
      default:
        return getCurrentLanguage() === "en" ? "Active Incident" : "活跃事件";
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

  async function submitAdminBootstrap(form) {
    setAdminNotice("正在初始化管理员...", false);
    try {
      await requestJSON("/api/v1/admin/bootstrap", {
        method: "POST",
        body: {
          password: fieldValue(form, "password")
        }
      });
      setAdminNotice("管理员已初始化。", false);
      renderRoute();
    } catch (error) {
      setAdminNotice(error.message || "初始化失败", true);
    }
  }

  async function submitAdminLogin(form) {
    setAdminNotice("正在登录...", false);
    try {
      await requestJSON("/api/v1/admin/login", {
        method: "POST",
        body: {
          password: fieldValue(form, "password")
        }
      });
      setAdminNotice("登录成功。", false);
      renderRoute();
    } catch (error) {
      setAdminNotice(error.message || "登录失败", true);
    }
  }

  async function submitAdminLogout() {
    setAdminNotice("正在退出...", false);
    try {
      await requestJSON("/api/v1/admin/logout", { method: "POST" });
      setAdminNotice("已退出。", false);
      renderRoute();
    } catch (error) {
      setAdminNotice(error.message || "退出失败", true);
    }
  }

  async function submitAdminPassword(form) {
    setAdminNotice("正在更新密码...", false);
    try {
      await requestJSON("/api/v1/admin/password", {
        method: "POST",
        body: {
          current_password: fieldValue(form, "current_password"),
          new_password: fieldValue(form, "new_password")
        }
      });
      form.reset();
      setAdminNotice("密码已更新。", false);
    } catch (error) {
      setAdminNotice(error.message || "更新密码失败", true);
    }
  }

  async function submitAdminCheck(form) {
    setAdminCheckNotice("正在保存检测项...", false);
    const payload = {
      id: fieldValue(form, "id"),
      name: fieldValue(form, "name"),
      type: fieldValue(form, "type") || "systemd",
      enabled: fieldChecked(form, "enabled"),
      scope_mode: fieldValue(form, "scope_mode") || "all",
      node_ids: fieldValues(form, "node_ids"),
      service_name: fieldValue(form, "service_name"),
      container_name: fieldValue(form, "container_name"),
      scheme: fieldValue(form, "scheme"),
      host_mode: fieldValue(form, "host_mode"),
      port: fieldNumber(form, "port"),
      path: fieldValue(form, "path"),
      expect_status: fieldNumber(form, "expect_status"),
      timeout: fieldValue(form, "timeout"),
      label: fieldValue(form, "label")
    };
    const method = payload.id ? "PUT" : "POST";
    const url = payload.id ? "/api/v1/admin/checks/" + encodeURIComponent(payload.id) : "/api/v1/admin/checks";

    try {
      await requestJSON(url, {
        method: method,
        body: payload
      });
      setAdminCheckNotice("检测项已保存。", false);
      activeAdminCheckID = "";
      resetAdminCheckForm();
      renderRoute();
    } catch (error) {
      setAdminCheckNotice(error.message || "保存检测项失败", true);
    }
  }

  async function submitAdminNodeName(form) {
    const nodeID = fieldValue(form, "node_id");
    if (!nodeID) {
      setAdminNodeNotice("请选择节点。", true);
      return;
    }
    setAdminNodeNotice("正在保存节点名称...", false);
    try {
      await requestJSON("/api/v1/admin/nodes/" + encodeURIComponent(nodeID), {
        method: "PUT",
        body: {
          display_name: fieldValue(form, "display_name")
        }
      });
      setAdminNodeNotice("节点名称已保存。", false);
      activeAdminNodeID = "";
      renderRoute();
    } catch (error) {
      setAdminNodeNotice(error.message || "保存节点名称失败", true);
    }
  }

  async function deleteAdminCheck(id) {
    if (!id) {
      return;
    }
    setAdminCheckNotice("正在删除检测项...", false);
    try {
      await requestJSON("/api/v1/admin/checks/" + encodeURIComponent(id), {
        method: "DELETE"
      });
      setAdminCheckNotice("检测项已删除。", false);
      if (activeAdminCheckID === String(id)) {
        activeAdminCheckID = "";
      }
      renderRoute();
    } catch (error) {
      setAdminCheckNotice(error.message || "删除检测项失败", true);
    }
  }

  async function deleteAdminNodeName(nodeID) {
    if (!nodeID) {
      return;
    }
    setAdminNodeNotice("正在恢复默认名称...", false);
    try {
      await requestJSON("/api/v1/admin/nodes/" + encodeURIComponent(nodeID), {
        method: "DELETE"
      });
      setAdminNodeNotice("已恢复默认名称。", false);
      if (activeAdminNodeID === String(nodeID)) {
        activeAdminNodeID = "";
      }
      renderRoute();
    } catch (error) {
      setAdminNodeNotice(error.message || "恢复默认名称失败", true);
    }
  }

  async function updateAdminMemberRole(nodeID, role) {
    if (!nodeID || !role) {
      return;
    }
    const actionLabel = role === "voter" ? "提升为 voter" : "降为 nonvoter";
    setAdminMemberNotice("正在" + actionLabel + "...", false);
    try {
      await requestJSON("/api/v1/admin/members/" + encodeURIComponent(nodeID) + "/role", {
        method: "PUT",
        body: {
          role: role
        }
      });
      setAdminMemberNotice(actionLabel + "完成。", false);
      renderRoute();
    } catch (error) {
      setAdminMemberNotice(error.message || (actionLabel + "失败"), true);
    }
  }

  async function deleteAdminMember(nodeID) {
    if (!nodeID) {
      return;
    }
    if (!window.confirm(localizeText("确认移除节点 " + nodeID + " 吗？这个操作会把它从当前集群成员列表和 Raft 配置里删掉。"))) {
      return;
    }
    setAdminMemberNotice("正在移除节点...", false);
    try {
      await requestJSON("/api/v1/admin/members/" + encodeURIComponent(nodeID), {
        method: "DELETE"
      });
      setAdminMemberNotice("节点已移除。", false);
      renderRoute();
    } catch (error) {
      setAdminMemberNotice(error.message || "移除节点失败", true);
    }
  }

  function fillAdminCheckForm(checkID) {
    const form = document.getElementById("admin-check-form");
    if (!form || !checkID) {
      return;
    }
    const check = currentAdminChecks.find((item) => String(item && item.id || "") === String(checkID));
    if (!check) {
      setAdminCheckNotice("检测项不存在或已刷新。", true);
      return;
    }
    setFieldValue(form, "id", check.id || "");
    setFieldValue(form, "name", check.name || "");
    setFieldValue(form, "type", check.type || "systemd");
    setFieldChecked(form, "enabled", check.enabled !== false);
    setFieldValue(form, "scope_mode", check.scope_mode || "all");
    setFieldValues(form, "node_ids", check.node_ids || []);
    setFieldValue(form, "service_name", check.service_name || "");
    setFieldValue(form, "container_name", check.container_name || "");
    setFieldValue(form, "scheme", check.scheme || "http");
    setFieldValue(form, "host_mode", check.host_mode || "peer");
    setFieldValue(form, "port", check.port || "");
    setFieldValue(form, "path", check.path || "");
    setFieldValue(form, "expect_status", check.expect_status || "");
    setFieldValue(form, "timeout", check.timeout || "");
    setFieldValue(form, "label", check.label || "");
    activeAdminCheckID = String(check.id || "");
    activeAdminNodeID = "";
    syncAdminCheckForm();
    syncAdminRowSelection();
    form.scrollIntoView({ behavior: "smooth", block: "start" });
  }

  function fillAdminNodeNameForm(nodeID) {
    const form = document.getElementById("admin-node-name-form");
    if (!form || !nodeID) {
      return;
    }
    const node = currentAdminNodes.find((item) => String(item && item.node_id || "") === String(nodeID));
    if (!node) {
      setAdminNodeNotice("节点映射不存在或已刷新。", true);
      return;
    }
    setFieldValue(form, "node_id", node.node_id || "");
    setFieldValue(form, "display_name", node.display_name || "");
    activeAdminNodeID = String(node.node_id || "");
    activeAdminCheckID = "";
    syncAdminSelectMenus();
    syncAdminRowSelection();
    form.scrollIntoView({ behavior: "smooth", block: "start" });
  }

  function resetAdminCheckForm() {
    const form = document.getElementById("admin-check-form");
    if (!form) {
      return;
    }
    form.reset();
    setFieldValue(form, "id", "");
    setFieldValue(form, "type", "systemd");
    setFieldValue(form, "scope_mode", "all");
    setFieldChecked(form, "enabled", true);
    setFieldValues(form, "node_ids", []);
    activeAdminCheckID = "";
    syncAdminCheckForm();
    syncAdminRowSelection();
  }

  function resetAdminNodeNameForm() {
    const form = document.getElementById("admin-node-name-form");
    if (!form) {
      return;
    }
    form.reset();
    const nodeField = formField(form, "node_id");
    if (nodeField && nodeField.dataset && nodeField.dataset.defaultValue !== undefined) {
      nodeField.value = nodeField.dataset.defaultValue;
    }
    setFieldValue(form, "display_name", "");
    activeAdminNodeID = "";
    syncAdminSelectMenus();
    syncAdminRowSelection();
  }

  function syncAdminCheckForm() {
    const form = document.getElementById("admin-check-form");
    if (!form) {
      return;
    }
    const type = fieldValue(form, "type") || "systemd";
    const scopeMode = fieldValue(form, "scope_mode") || "all";
    const visibleFields = {
      systemd: ["service_name"],
      docker: ["container_name"],
      http: ["scheme", "host_mode", "port", "path", "expect_status", "timeout"],
      tcp: ["port", "label"]
    }[type] || [];

    document.querySelectorAll(".admin-check-field").forEach((field) => {
      const fieldName = field.dataset.field || "";
      const isVisible = fieldName === "node_ids"
        ? scopeMode !== "all"
        : visibleFields.includes(fieldName);
      field.hidden = !isVisible;
    });
    syncAdminSelectMenus();
  }

  function syncAdminRowSelection() {
    document.querySelectorAll(".admin-check-row[data-check-id]").forEach((row) => {
      row.classList.toggle("is-editing", activeAdminCheckID !== "" && row.dataset.checkId === activeAdminCheckID);
    });
    document.querySelectorAll(".admin-check-row[data-node-id]").forEach((row) => {
      row.classList.toggle("is-editing", activeAdminNodeID !== "" && row.dataset.nodeId === activeAdminNodeID);
    });
  }

  function formField(form, name) {
    return form && form.elements ? form.elements.namedItem(name) : null;
  }

  function fieldValue(form, name) {
    const field = formField(form, name);
    return field && "value" in field ? field.value : "";
  }

  function fieldNumber(form, name) {
    const value = fieldValue(form, name);
    return value ? Number(value) : 0;
  }

  function fieldChecked(form, name) {
    const field = formField(form, name);
    return Boolean(field && field.checked);
  }

  function fieldValues(form, name) {
    if (!form) {
      return [];
    }
    return Array.from(form.querySelectorAll('[name="' + name + '"]'))
      .filter((field) => field.checked)
      .map((field) => field.value || "");
  }

  function setFieldValue(form, name, value) {
    const field = formField(form, name);
    if (field && "value" in field) {
      field.value = value;
    }
  }

  function setFieldChecked(form, name, checked) {
    const field = formField(form, name);
    if (field) {
      field.checked = Boolean(checked);
    }
  }

  function setFieldValues(form, name, values) {
    const selected = new Set(Array.isArray(values) ? values.map((value) => String(value || "")) : []);
    Array.from(form ? form.querySelectorAll('[name="' + name + '"]') : []).forEach((field) => {
      field.checked = selected.has(String(field.value || ""));
    });
  }

  async function submitTestAlert(form) {
    const result = document.getElementById("test-alert-result");
    if (result) {
      result.innerHTML = localizeMarkup("<p>正在发送测试告警...</p>");
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
    result.innerHTML = localizeMarkup(`
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
    `);
  }

  async function fetchJSON(url) {
    return requestJSON(url, { method: "GET" });
  }

  async function requestJSON(url, options) {
    const settings = options || {};
    const headers = {
      Accept: "application/json"
    };
    let body = settings.body;
    if (body !== undefined) {
      headers["Content-Type"] = "application/json";
      body = JSON.stringify(body);
    }
    const response = await fetch(url, {
      method: settings.method || "GET",
      headers: headers,
      body: body
    });
    const data = await response.json().catch(() => ({}));
    if (!response.ok) {
      throw new Error(data.error || ("request failed: " + response.status));
    }
    return data;
  }

  async function fetchHistory(nodeID, metric) {
    const query = new URLSearchParams({
      node_id: nodeID,
      metric: metric
    });
    return fetchJSON("/api/v1/history?" + query.toString());
  }

  function syncThemeSelect() {
    const current = currentControlOption("theme");
    document.querySelectorAll('[data-control-current="theme"]').forEach((node) => {
      node.textContent = current.label;
    });
    document.querySelectorAll('[data-control-option][data-control-kind="theme"]').forEach((button) => {
      const active = (button.dataset.controlValue || "") === current.id;
      button.classList.toggle("is-active", active);
      button.setAttribute("aria-checked", active ? "true" : "false");
      const check = button.querySelector(".obs-theme-switcher__check");
      if (check) {
        check.hidden = !active;
      }
    });
  }

  function syncLanguageSelect() {
    const current = currentControlOption("language");
    document.querySelectorAll('[data-control-current="language"]').forEach((node) => {
      node.textContent = current.label;
    });
    document.querySelectorAll('[data-control-option][data-control-kind="language"]').forEach((button) => {
      const active = (button.dataset.controlValue || "") === current.id;
      button.classList.toggle("is-active", active);
      button.setAttribute("aria-checked", active ? "true" : "false");
      const check = button.querySelector(".obs-theme-switcher__check");
      if (check) {
        check.hidden = !active;
      }
    });
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

  function formatStableNodesSummary(count) {
    return getCurrentLanguage() === "en" ? count + " nodes stable" : count + " 个节点处于稳定状态";
  }

  function formatServiceIssueSummary(count) {
    return getCurrentLanguage() === "en" ? count + " services are degraded or failing." : count + " 项服务处于异常或不稳定状态。";
  }

  function formatProbeObservationSummary(count) {
    return getCurrentLanguage() === "en"
      ? "Using the latest " + count + " observations to see where the path starts to degrade."
      : "最近 " + count + " 条观测，直接用来判断从哪一层开始失真。";
  }

  function formatFleetSummary(nodeCount, healthyCount, activeIncidents) {
    return getCurrentLanguage() === "en"
      ? nodeCount + " nodes · " + healthyCount + " stable · " + activeIncidents + " active incidents"
      : nodeCount + " 个节点 · " + healthyCount + " 稳定 · " + activeIncidents + " 个活跃事件";
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
        return getCurrentLanguage() === "en" ? "Healthy" : "稳定";
      case "degraded":
        return getCurrentLanguage() === "en" ? "Degraded" : "降级";
      case "critical":
        return getCurrentLanguage() === "en" ? "Critical" : "严重";
      default:
        return getCurrentLanguage() === "en" ? "Unknown" : "未知";
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
      return getCurrentLanguage() === "en" ? "Never" : "从未";
    }
    const date = new Date(value);
    if (Number.isNaN(date.getTime())) {
      return getCurrentLanguage() === "en" ? "Never" : "从未";
    }
    return date.toLocaleString(getCurrentLanguage() === "en" ? "en-US" : "zh-CN", { hour12: false });
  }

  function timeAgo(value) {
    if (!value) {
      return getCurrentLanguage() === "en" ? "No signal" : "无信号";
    }
    const date = new Date(value);
    if (Number.isNaN(date.getTime())) {
      return getCurrentLanguage() === "en" ? "No signal" : "无信号";
    }
    let diff = Math.abs(Date.now() - date.getTime());
    if (diff < 60000) {
      return getCurrentLanguage() === "en"
        ? Math.max(1, Math.round(diff / 1000)) + "s ago"
        : Math.max(1, Math.round(diff / 1000)) + " 秒前";
    }
    if (diff < 3600000) {
      return getCurrentLanguage() === "en"
        ? Math.max(1, Math.round(diff / 60000)) + "m ago"
        : Math.max(1, Math.round(diff / 60000)) + " 分钟前";
    }
    return getCurrentLanguage() === "en"
      ? Math.max(1, Math.round(diff / 3600000)) + "h ago"
      : Math.max(1, Math.round(diff / 3600000)) + " 小时前";
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
