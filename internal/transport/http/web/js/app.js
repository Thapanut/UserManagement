/**
 * Mission Control - Web Interface Logic
 * Connects to Go Hexagonal Architecture REST API (/api/v1)
 * Displays Live Telemetry, Auth, and CRUD Operations
 */

(function () {
  'use strict';

  // --- Application State ---
  const state = {
    token: localStorage.getItem('mc_jwt_token') || '',
    userEmail: localStorage.getItem('mc_user_email') || '',
    users: [],
    lastCurl: '# Equivalent curl command will be displayed here',
    workerSeconds: 10,
  };

  // --- DOM Elements ---
  const el = {
    // Top Nav & Telemetry
    authSlot: document.getElementById('auth-slot'),
    statusHttp: document.getElementById('status-http'),
    statusGrpc: document.getElementById('status-grpc'),
    statusMongo: document.getElementById('status-mongo'),
    workerTimer: document.getElementById('worker-timer'),
    workerPulse: document.getElementById('worker-pulse'),

    // Metrics
    metricTotalUsers: document.getElementById('metric-total-users'),
    metricSessionStatus: document.getElementById('metric-session-status'),
    metricSessionDesc: document.getElementById('metric-session-desc'),

    // Directory
    inputSearch: document.getElementById('input-search'),
    btnRefresh: document.getElementById('btn-refresh'),
    btnCreateUser: document.getElementById('btn-create-user'),
    usersTbody: document.getElementById('users-tbody'),

    // Inspector
    inspectorStatus: document.getElementById('inspector-last-status'),
    inspectorUrl: document.getElementById('inspector-url'),
    inspectorTime: document.getElementById('inspector-time'),
    inspectorCurl: document.getElementById('inspector-curl'),
    btnCopyCurl: document.getElementById('btn-copy-curl'),

    // Modals
    modalAuth: document.getElementById('modal-auth'),
    tabLogin: document.getElementById('tab-login'),
    tabRegister: document.getElementById('tab-register'),
    formLogin: document.getElementById('form-login'),
    formRegister: document.getElementById('form-register'),
    loginEmail: document.getElementById('login-email'),
    loginPassword: document.getElementById('login-password'),
    regName: document.getElementById('reg-name'),
    regEmail: document.getElementById('reg-email'),
    regPassword: document.getElementById('reg-password'),

    modalCreateUser: document.getElementById('modal-create-user'),
    formCreateUser: document.getElementById('form-create-user'),
    createName: document.getElementById('create-name'),
    createEmail: document.getElementById('create-email'),
    createPassword: document.getElementById('create-password'),

    modalEditUser: document.getElementById('modal-edit-user'),
    formEditUser: document.getElementById('form-edit-user'),
    editUserId: document.getElementById('edit-user-id'),
    editUserIdDisplay: document.getElementById('edit-user-id-display'),
    editName: document.getElementById('edit-name'),
    editEmail: document.getElementById('edit-email'),

    modalDeleteUser: document.getElementById('modal-delete-user'),
    deleteUserName: document.getElementById('delete-user-name'),
    deleteUserEmail: document.getElementById('delete-user-email'),
    deleteUserId: document.getElementById('delete-user-id'),
    btnConfirmDelete: document.getElementById('btn-confirm-delete'),

    authAlert: document.getElementById('auth-alert'),
    createUserAlert: document.getElementById('create-user-alert'),
    editUserAlert: document.getElementById('edit-user-alert'),

    toastContainer: document.getElementById('toast-container'),
  };

  // --- Alert Helper ---
  function setAlert(elem, message, type = 'error') {
    if (!elem) return;
    if (!message) {
      elem.style.display = 'none';
      elem.textContent = '';
      elem.className = 'modal-alert';
      return;
    }
    elem.className = `modal-alert ${type}`;
    elem.textContent = message;
    elem.style.display = 'flex';
  }

  // --- API Client & Telemetry Inspector Helper ---
  async function apiCall(endpoint, options = {}) {
    const startTime = performance.now();
    const method = options.method || 'GET';
    const headers = {
      'Content-Type': 'application/json',
      ...(options.headers || {}),
    };

    if (state.token && !headers['Authorization']) {
      headers['Authorization'] = `Bearer ${state.token}`;
    }

    // Build equivalent curl snippet
    let curlCmd = `curl -X ${method} "http://localhost:8080${endpoint}"`;
    for (const [key, val] of Object.entries(headers)) {
      curlCmd += ` \\\n  -H "${key}: ${val}"`;
    }
    if (options.body) {
      curlCmd += ` \\\n  -d '${options.body}'`;
    }

    state.lastCurl = curlCmd;
    if (el.inspectorCurl) {
      el.inspectorCurl.textContent = curlCmd;
    }
    if (el.inspectorUrl) {
      el.inspectorUrl.textContent = `${method} ${endpoint}`;
    }

    try {
      const response = await fetch(endpoint, {
        method,
        headers,
        body: options.body,
      });

      const elapsed = Math.round(performance.now() - startTime);
      if (el.inspectorTime) {
        el.inspectorTime.textContent = `${new Date().toLocaleTimeString()} (${elapsed}ms)`;
      }

      const statusText = `${response.status} ${response.statusText || ''}`.trim();
      if (el.inspectorStatus) {
        el.inspectorStatus.textContent = statusText;
        el.inspectorStatus.className = `inspector-badge ${response.ok ? 'success' : 'error'}`;
      }

      let data = null;
      const contentType = response.headers.get('content-type');
      if (contentType && contentType.includes('application/json')) {
        data = await response.json();
      } else {
        data = await response.text();
      }

      if (!response.ok) {
        const errorMsg = (data && data.error) || (typeof data === 'string' && data) || `HTTP error ${response.status}`;
        throw new Error(errorMsg);
      }

      return data;
    } catch (err) {
      const elapsed = Math.round(performance.now() - startTime);
      if (el.inspectorTime) {
        el.inspectorTime.textContent = `${new Date().toLocaleTimeString()} (${elapsed}ms)`;
      }
      if (el.inspectorStatus) {
        el.inspectorStatus.textContent = 'ERROR';
        el.inspectorStatus.className = 'inspector-badge error';
      }
      throw err;
    }
  }

  // --- Toast Notifications ---
  function showToast(message, type = 'info') {
    const toast = document.createElement('div');
    toast.className = `toast ${type}`;

    let iconSvg = '';
    if (type === 'success') {
      iconSvg = '<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="#4ade80" stroke-width="2"><polyline points="20 6 9 17 4 12"></polyline></svg>';
    } else if (type === 'error') {
      iconSvg = '<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="#f87171" stroke-width="2"><circle cx="12" cy="12" r="10"></circle><line x1="12" y1="8" x2="12" y2="12"></line><line x1="12" y1="16" x2="12.01" y2="16"></line></svg>';
    } else {
      iconSvg = '<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="#3b82f6" stroke-width="2"><circle cx="12" cy="12" r="10"></circle><line x1="12" y1="16" x2="12" y2="12"></line><line x1="12" y1="8" x2="12.01" y2="8"></line></svg>';
    }

    toast.innerHTML = `
      ${iconSvg}
      <span style="flex: 1;">${escapeHtml(message)}</span>
    `;

    el.toastContainer.appendChild(toast);

    setTimeout(() => {
      toast.style.opacity = '0';
      toast.style.transform = 'translateY(10px)';
      toast.style.transition = 'all 0.3s ease';
      setTimeout(() => toast.remove(), 300);
    }, 3500);
  }

  // --- Modal Helpers ---
  function openModal(modal) {
    if (modal && typeof modal.showModal === 'function') {
      modal.showModal();
    }
  }

  function closeModal(modal) {
    if (modal && typeof modal.close === 'function') {
      modal.close();
    }
  }

  // Global close listener for data-close attribute
  document.addEventListener('click', (e) => {
    const target = e.target.closest('[data-close]');
    if (target) {
      const modalId = target.getAttribute('data-close');
      const targetModal = document.getElementById(modalId);
      if (targetModal) closeModal(targetModal);
    }
  });

  // --- Session Management & UI Sync ---
  function updateSessionUI() {
    if (state.token) {
      // Authenticated state
      el.authSlot.innerHTML = `
        <div class="user-profile-pill">
          <span class="pulse-dot active"></span>
          <span class="user-profile-email" title="${escapeHtml(state.userEmail)}">${escapeHtml(state.userEmail || 'Active Session')}</span>
          <button class="btn-sign-out" id="btn-sign-out" title="Sign out and clear JWT token">Sign Out</button>
        </div>
      `;

      document.getElementById('btn-sign-out').addEventListener('click', signOut);

      el.metricSessionStatus.textContent = 'Authenticated';
      el.metricSessionStatus.style.color = 'var(--accent-mint)';
      el.metricSessionDesc.innerHTML = '<span class="footer-tag mint">JWT Token Valid (HS256)</span>';
    } else {
      // Guest state
      el.authSlot.innerHTML = `
        <button class="nav-btn primary" id="btn-open-auth">Sign In</button>
      `;
      document.getElementById('btn-open-auth').addEventListener('click', () => {
        openModal(el.modalAuth);
      });

      el.metricSessionStatus.textContent = 'Guest Mode';
      el.metricSessionStatus.style.color = 'var(--text-primary)';
      el.metricSessionDesc.innerHTML = '<span class="footer-tag muted">JWT Token Required for CRUD</span>';
    }
  }

  function signOut() {
    state.token = '';
    state.userEmail = '';
    localStorage.removeItem('mc_jwt_token');
    localStorage.removeItem('mc_user_email');
    updateSessionUI();
    showToast('Signed out. Switched to Guest Mode.', 'info');
    loadUsers();
  }

  // --- Telemetry Background Ticker (Goroutine Simulation) ---
  function initGoroutineTicker() {
    setInterval(() => {
      state.workerSeconds -= 1;
      if (state.workerSeconds <= 0) {
        state.workerSeconds = 10;
        // Trigger pulse effect
        if (el.workerPulse) {
          el.workerPulse.classList.remove('active');
          void el.workerPulse.offsetWidth; // force reflow
          el.workerPulse.classList.add('active');
        }
      }
      if (el.workerTimer) {
        el.workerTimer.textContent = `${state.workerSeconds}s`;
      }
    }, 1000);
  }

  // --- User Directory Operations ---
  async function loadUsers() {
    el.usersTbody.innerHTML = `
      <tr>
        <td colspan="5" class="empty-state">
          <div class="loading-spinner"></div>
          <span>Querying REST API & MongoDB...</span>
        </td>
      </tr>
    `;

    try {
      const data = await apiCall('/api/v1/users');
      // Backend returns StandardResponse: { success: true, data: [...], message: "..." }
      const userList = (data && Array.isArray(data.data)) ? data.data : (Array.isArray(data) ? data : []);
      state.users = userList;
      el.metricTotalUsers.textContent = state.users.length.toString();
      renderUsersTable(state.users);
    } catch (err) {
      el.metricTotalUsers.textContent = '-';
      if (err.message && (err.message.toLowerCase().includes('token') || err.message.toLowerCase().includes('unauthorized') || err.message.includes('401'))) {
        el.usersTbody.innerHTML = `
          <tr>
            <td colspan="5" class="empty-state">
              <div style="margin-bottom: 8px; color: var(--accent-amber);">Authentication Required (401 Unauthorized)</div>
              <p style="font-size: 0.8rem; margin-bottom: 12px; color: var(--text-secondary);">The User Directory is protected by JWTMiddleware. Please Sign In with your credentials or register a new user.</p>
              <button class="nav-btn primary" onclick="document.getElementById('modal-auth').showModal()">Sign In Now</button>
            </td>
          </tr>
        `;
      } else {
        el.usersTbody.innerHTML = `
          <tr>
            <td colspan="5" class="empty-state" style="color: var(--accent-coral);">
              <div>Error fetching users: ${escapeHtml(err.message)}</div>
              <button class="nav-btn ghost" style="margin-top: 10px;" id="btn-retry-load">Retry</button>
            </td>
          </tr>
        `;
        const retryBtn = document.getElementById('btn-retry-load');
        if (retryBtn) retryBtn.addEventListener('click', loadUsers);
      }
    }
  }

  function renderUsersTable(users) {
    if (!users || users.length === 0) {
      el.usersTbody.innerHTML = `
        <tr>
          <td colspan="5" class="empty-state">
            <div style="font-weight: 500; margin-bottom: 4px;">No users found in database</div>
            <div style="font-size: 0.8rem;">Click "New User" above to create your first user record.</div>
          </td>
        </tr>
      `;
      return;
    }

    el.usersTbody.innerHTML = users
      .map((u) => {
        const initials = getInitials(u.name || u.email || 'U');
        const createdDate = u.created_at ? formatDate(u.created_at) : '-';
        const userId = u.id || u._id || '';

        return `
          <tr data-user-id="${escapeHtml(userId)}">
            <td>
              <div class="user-cell">
                <div class="user-avatar">${escapeHtml(initials)}</div>
                <div class="user-info">
                  <span class="user-name">${escapeHtml(u.name || 'Unnamed')}</span>
                </div>
              </div>
            </td>
            <td>
              <span style="color: var(--text-secondary); font-size: 0.84rem;">${escapeHtml(u.email || '-')}</span>
            </td>
            <td>
              <span class="code-badge" title="${escapeHtml(userId)}">${escapeHtml(truncate(userId, 16))}</span>
            </td>
            <td>
              <span class="date-text">${escapeHtml(createdDate)}</span>
            </td>
            <td style="text-align: right;">
              <div class="action-btn-group">
                <button class="action-btn btn-edit-user" data-id="${escapeHtml(userId)}" data-name="${escapeHtml(u.name || '')}" data-email="${escapeHtml(u.email || '')}" title="Edit User">
                  <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M12 20h9"></path><path d="M16.5 3.5a2.121 2.121 0 0 1 3 3L7 19l-4 1 1-4L16.5 3.5z"></path></svg>
                  <span>Edit</span>
                </button>
                <button class="action-btn delete btn-delete-user" data-id="${escapeHtml(userId)}" data-name="${escapeHtml(u.name || '')}" data-email="${escapeHtml(u.email || '')}" title="Delete User">
                  <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="3 6 5 6 21 6"></polyline><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"></path></svg>
                  <span>Delete</span>
                </button>
              </div>
            </td>
          </tr>
        `;
      })
      .join('');

    // Attach Edit button events
    el.usersTbody.querySelectorAll('.btn-edit-user').forEach((btn) => {
      btn.addEventListener('click', () => {
        const id = btn.getAttribute('data-id');
        const name = btn.getAttribute('data-name');
        const email = btn.getAttribute('data-email');

        el.editUserId.value = id;
        el.editUserIdDisplay.value = id;
        el.editName.value = name;
        el.editEmail.value = email;

        setAlert(el.editUserAlert, '');
        openModal(el.modalEditUser);
      });
    });

    // Attach Delete button events
    el.usersTbody.querySelectorAll('.btn-delete-user').forEach((btn) => {
      btn.addEventListener('click', () => {
        const id = btn.getAttribute('data-id');
        const name = btn.getAttribute('data-name');
        const email = btn.getAttribute('data-email');

        el.deleteUserId.textContent = id;
        el.deleteUserName.textContent = name || 'Unnamed';
        el.deleteUserEmail.textContent = email || '-';
        el.btnConfirmDelete.setAttribute('data-target-id', id);

        openModal(el.modalDeleteUser);
      });
    });
  }

  // --- Filter / Search ---
  el.inputSearch.addEventListener('input', (e) => {
    const q = e.target.value.toLowerCase().trim();
    if (!q) {
      renderUsersTable(state.users);
      return;
    }
    const filtered = state.users.filter((u) => {
      const name = (u.name || '').toLowerCase();
      const email = (u.email || '').toLowerCase();
      const id = (u.id || u._id || '').toLowerCase();
      return name.includes(q) || email.includes(q) || id.includes(q);
    });
    renderUsersTable(filtered);
  });

  // --- Refresh Button ---
  el.btnRefresh.addEventListener('click', () => {
    loadUsers();
    showToast('Refreshed user directory.', 'info');
  });

  // --- Auth Modal Tab Switching ---
  el.tabLogin.addEventListener('click', () => {
    el.tabLogin.classList.add('active');
    el.tabRegister.classList.remove('active');
    el.formLogin.classList.add('active');
    el.formRegister.classList.remove('active');
    setAlert(el.authAlert, '');
  });

  el.tabRegister.addEventListener('click', () => {
    el.tabRegister.classList.add('active');
    el.tabLogin.classList.remove('active');
    el.formRegister.classList.add('active');
    el.formLogin.classList.remove('active');
    setAlert(el.authAlert, '');
  });

  // --- Login Form Submission ---
  el.formLogin.addEventListener('submit', async (e) => {
    e.preventDefault();
    const email = el.loginEmail.value.trim();
    const password = el.loginPassword.value;

    const submitBtn = el.formLogin.querySelector('button[type="submit"]');
    submitBtn.disabled = true;
    submitBtn.innerHTML = '<span>Signing In...</span>';
    setAlert(el.authAlert, '');

    try {
      const data = await apiCall('/api/v1/auth/login', {
        method: 'POST',
        body: JSON.stringify({ email, password }),
      });

      // Extract token from either data.data.token or data.token
      const token = (data && data.data && data.data.token) || (data && data.token);

      if (token) {
        state.token = token;
        state.userEmail = email;
        localStorage.setItem('mc_jwt_token', token);
        localStorage.setItem('mc_user_email', email);

        setAlert(el.authAlert, 'Sign in successful! Entering dashboard...', 'success');
        updateSessionUI();
        showToast(`Signed in successfully as ${email}`, 'success');
        el.loginPassword.value = '';

        setTimeout(() => {
          closeModal(el.modalAuth);
          setAlert(el.authAlert, '');
          loadUsers();
        }, 350);
      } else {
        throw new Error((data && data.error) || (data && data.message) || 'No token returned from server');
      }
    } catch (err) {
      setAlert(el.authAlert, err.message, 'error');
      showToast(err.message, 'error');
    } finally {
      submitBtn.disabled = false;
      submitBtn.innerHTML = '<span>Sign In</span>';
    }
  });

  // --- Register Form Submission ---
  el.formRegister.addEventListener('submit', async (e) => {
    e.preventDefault();
    const name = el.regName.value.trim();
    const email = el.regEmail.value.trim();
    const password = el.regPassword.value;

    const submitBtn = el.formRegister.querySelector('button[type="submit"]');
    submitBtn.disabled = true;
    submitBtn.innerHTML = '<span>Creating Account...</span>';
    setAlert(el.authAlert, '');

    try {
      await apiCall('/api/v1/auth/register', {
        method: 'POST',
        body: JSON.stringify({ name, email, password }),
      });

      setAlert(el.authAlert, 'Account registered! Signing in automatically...', 'success');
      showToast('Account registered! Automatically signing in...', 'success');

      // Auto login with new credentials
      const loginData = await apiCall('/api/v1/auth/login', {
        method: 'POST',
        body: JSON.stringify({ email, password }),
      });

      const token = (loginData && loginData.data && loginData.data.token) || (loginData && loginData.token);

      if (token) {
        state.token = token;
        state.userEmail = email;
        localStorage.setItem('mc_jwt_token', token);
        localStorage.setItem('mc_user_email', email);

        updateSessionUI();
        showToast(`Welcome, ${name}! Signed in successfully.`, 'success');
        el.regPassword.value = '';

        setTimeout(() => {
          closeModal(el.modalAuth);
          setAlert(el.authAlert, '');
          loadUsers();
        }, 350);
      } else {
        setAlert(el.authAlert, 'Registered successfully! Please Sign In.', 'success');
        setTimeout(() => {
          el.tabLogin.click();
          el.loginEmail.value = email;
        }, 800);
      }
    } catch (err) {
      setAlert(el.authAlert, err.message, 'error');
      showToast(err.message, 'error');
    } finally {
      submitBtn.disabled = false;
      submitBtn.innerHTML = '<span>Create Account</span>';
    }
  });

  // --- Create User (Protected POST /users) ---
  el.btnCreateUser.addEventListener('click', () => {
    if (!state.token) {
      showToast('Please sign in first to create users.', 'error');
      setAlert(el.authAlert, 'Please sign in first to manage users.', 'error');
      openModal(el.modalAuth);
      return;
    }
    el.formCreateUser.reset();
    setAlert(el.createUserAlert, '');
    openModal(el.modalCreateUser);
  });

  el.formCreateUser.addEventListener('submit', async (e) => {
    e.preventDefault();
    const name = el.createName.value.trim();
    const email = el.createEmail.value.trim();
    const password = el.createPassword.value;

    const submitBtn = el.formCreateUser.querySelector('button[type="submit"]');
    submitBtn.disabled = true;
    submitBtn.textContent = 'Saving...';
    setAlert(el.createUserAlert, '');

    try {
      await apiCall('/api/v1/users', {
        method: 'POST',
        body: JSON.stringify({ name, email, password }),
      });

      setAlert(el.createUserAlert, `User ${name} created successfully!`, 'success');
      showToast(`User ${name} created successfully!`, 'success');

      setTimeout(() => {
        closeModal(el.modalCreateUser);
        setAlert(el.createUserAlert, '');
        loadUsers();
      }, 350);
    } catch (err) {
      setAlert(el.createUserAlert, err.message, 'error');
      showToast(err.message, 'error');
    } finally {
      submitBtn.disabled = false;
      submitBtn.textContent = 'Create User';
    }
  });

  // --- Edit User (Protected PUT /users/{id}) ---
  el.formEditUser.addEventListener('submit', async (e) => {
    e.preventDefault();
    const id = el.editUserId.value;
    const name = el.editName.value.trim();
    const email = el.editEmail.value.trim();

    const payload = {};
    if (name) payload.name = name;
    if (email) payload.email = email;

    if (Object.keys(payload).length === 0) {
      showToast('No changes specified.', 'info');
      closeModal(el.modalEditUser);
      return;
    }

    const submitBtn = el.formEditUser.querySelector('button[type="submit"]');
    submitBtn.disabled = true;
    submitBtn.textContent = 'Saving...';
    setAlert(el.editUserAlert, '');

    try {
      await apiCall(`/api/v1/users/${id}`, {
        method: 'PUT',
        body: JSON.stringify(payload),
      });

      setAlert(el.editUserAlert, 'User record updated successfully!', 'success');
      showToast('User record updated.', 'success');

      setTimeout(() => {
        closeModal(el.modalEditUser);
        setAlert(el.editUserAlert, '');
        loadUsers();
      }, 350);
    } catch (err) {
      setAlert(el.editUserAlert, err.message, 'error');
      showToast(err.message, 'error');
    } finally {
      submitBtn.disabled = false;
      submitBtn.textContent = 'Save Changes';
    }
  });

  // --- Delete User (Protected DELETE /users/{id}) ---
  el.btnConfirmDelete.addEventListener('click', async () => {
    const id = el.btnConfirmDelete.getAttribute('data-target-id');
    if (!id) return;

    el.btnConfirmDelete.disabled = true;
    el.btnConfirmDelete.textContent = 'Deleting...';

    try {
      await apiCall(`/api/v1/users/${id}`, {
        method: 'DELETE',
      });

      closeModal(el.modalDeleteUser);
      showToast('User deleted from MongoDB.', 'success');
      loadUsers();
    } catch (err) {
      showToast(err.message, 'error');
    } finally {
      el.btnConfirmDelete.disabled = false;
      el.btnConfirmDelete.textContent = 'Delete User';
    }
  });

  // --- Copy cURL Button ---
  el.btnCopyCurl.addEventListener('click', async () => {
    try {
      await navigator.clipboard.writeText(state.lastCurl);
      const originalText = el.btnCopyCurl.textContent;
      el.btnCopyCurl.textContent = 'Copied!';
      setTimeout(() => {
        el.btnCopyCurl.textContent = originalText;
      }, 1500);
    } catch (err) {
      showToast('Failed to copy to clipboard', 'error');
    }
  });

  // --- Utility Functions ---
  function escapeHtml(str) {
    if (!str) return '';
    return String(str)
      .replace(/&/g, '&amp;')
      .replace(/</g, '&lt;')
      .replace(/>/g, '&gt;')
      .replace(/"/g, '&quot;')
      .replace(/'/g, '&#039;');
  }

  function getInitials(name) {
    if (!name) return 'U';
    const parts = name.trim().split(/\s+/);
    if (parts.length >= 2) {
      return (parts[0][0] + parts[1][0]).toUpperCase();
    }
    return name.slice(0, 2).toUpperCase();
  }

  function formatDate(isoStr) {
    try {
      const d = new Date(isoStr);
      return d.toLocaleDateString(undefined, {
        year: 'numeric',
        month: 'short',
        day: 'numeric',
      });
    } catch (e) {
      return isoStr;
    }
  }

  function truncate(str, maxLen) {
    if (!str) return '';
    return str.length > maxLen ? str.slice(0, maxLen) + '...' : str;
  }

  // --- Initialization ---
  function init() {
    updateSessionUI();
    initGoroutineTicker();
    loadUsers();
  }

  // Run when DOM is ready
  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', init);
  } else {
    init();
  }
})();
