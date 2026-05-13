import API from './api.js';
import Dom from './dom.js';
import { t, setLocale, applyLocale, currentLocale } from './i18n.js';

// App state
let currentUser = null;
let allStandards = [];
let myStandards = [];
let categories = [];
let currentPage = 1;

function debugPrint(...args) {
    if (window.DEBUG) console.log('[DEBUG]', ...args);
}

function escapeHtml(str) {
    if (!str && str !== 0) return '';
    return String(str).replace(/[&<>"'`]/g, s => ({
        '&': '&amp;', '<': '&lt;', '>': '&gt;',
        '"': '&quot;', "'": '&#39;', '`': '&#96;'
    })[s]);
}

async function init() {
    const token = API.getToken();
    if (token) {
        try {
            currentUser = await API.getMe();
            showMainScreen();
            await loadMyStandards();
        } catch (error) {
            API.clearToken();
            showAuthScreen();
        }
    } else {
        showAuthScreen();
    }
    setupEventListeners();
    checkOnlineStatus();
    applyLocale();
}

function setupEventListeners() {
    document.querySelectorAll('.tab').forEach(tab => {
        tab.addEventListener('click', e => {
            const tabName = e.target.dataset.tab;
            document.querySelectorAll('.tab').forEach(t => { t.classList.remove('active'); t.setAttribute('aria-selected', 'false'); });
            document.querySelectorAll('.auth-form').forEach(f => f.classList.remove('active'));
            e.target.classList.add('active');
            e.target.setAttribute('aria-selected', 'true');
            document.getElementById(`${tabName}-form`).classList.add('active');
        });
    });

    document.getElementById('login-form').addEventListener('submit', handleLogin);
    document.getElementById('register-form').addEventListener('submit', handleRegister);
    const passkeyLoginBtn = document.getElementById('passkey-login-btn');
    if (passkeyLoginBtn) passkeyLoginBtn.addEventListener('click', handlePasskeyLogin);
    document.getElementById('logout-btn').addEventListener('click', handleLogout);

    const localeBtn = document.getElementById('locale-toggle-btn');
    if (localeBtn) localeBtn.addEventListener('click', () => {
        const next = currentLocale === 'en' ? 'ru' : 'en';
        setLocale(next);
        applyLocale();
        renderMyStandards();
        if (!document.getElementById('all-standards-view').classList.contains('hidden')) {
            renderAllStandards({ standards: allStandards, page: currentPage, total: allStandards.length, limit: 100 });
        }
    });

    document.querySelectorAll('.nav-btn').forEach(btn => {
        btn.addEventListener('click', e => switchView(e.target.closest('[data-view]').dataset.view));
    });

    const profileForm = document.getElementById('update-profile-form');
    if (profileForm) profileForm.addEventListener('submit', handleUpdateProfile);
    const passkeyForm = document.getElementById('create-passkey-form');
    if (passkeyForm) passkeyForm.addEventListener('submit', handleCreatePassKey);

    document.getElementById('search-input').addEventListener('input', debounce(loadAllStandards, 300));
    document.getElementById('style-filter').addEventListener('change', loadAllStandards);

    document.getElementById('my-style-filter').addEventListener('change', renderMyStandards);
    document.getElementById('my-composer-filter').addEventListener('input', debounce(renderMyStandards, 200));
    document.getElementById('my-proficiency-filter').addEventListener('change', renderMyStandards);

    document.getElementById('add-standard-btn').addEventListener('click', () => openModal('add-standard-modal'));
    document.getElementById('add-category-btn').addEventListener('click', () => openModal('add-category-modal'));
    document.getElementById('add-standard-form').addEventListener('submit', handleAddStandard);
    document.getElementById('add-category-form').addEventListener('submit', handleAddCategory);

    const colorInput = document.getElementById('new-category-color');
    const colorLabel = document.getElementById('color-preview-label');
    if (colorInput && colorLabel) {
        colorInput.addEventListener('input', () => { colorLabel.textContent = colorInput.value; });
    }

    document.addEventListener('click', e => {
        const closeTarget = e.target.closest('[data-close-modal]');
        if (closeTarget) { closeModal(closeTarget.dataset.closeModal); return; }
        if (e.target.classList.contains('modal')) closeModal(e.target.id);
    });

    document.addEventListener('keydown', e => {
        if (e.key === 'Escape') {
            const om = document.querySelector('.modal:not(.hidden)');
            if (om) closeModal(om.id);
        }
    });

    const friendSearchBtn = document.getElementById('friend-search-btn');
    if (friendSearchBtn) friendSearchBtn.addEventListener('click', handleFriendSearch);
    const friendSearchInput = document.getElementById('friend-search-input');
    if (friendSearchInput) friendSearchInput.addEventListener('keydown', e => { if (e.key === 'Enter') handleFriendSearch(); });

    const markReadBtn = document.getElementById('mark-read-btn');
    if (markReadBtn) markReadBtn.addEventListener('click', handleMarkNotificationsRead);

    Dom.attachDelegation();
}

let _prevFocus = null;

function openModal(modalId) {
    const modal = document.getElementById(modalId);
    if (!modal) return;
    _prevFocus = document.activeElement;
    modal.classList.remove('hidden');
    modal.setAttribute('aria-hidden', 'false');
    modal.removeAttribute('inert');
    const focusable = modal.querySelectorAll('button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])');
    const firstReal = Array.from(focusable).find(el => !el.hasAttribute('data-close-modal') || (el.tagName === 'BUTTON' && !el.classList.contains('modal-close')));
    const target = firstReal || focusable[0];
    if (target) requestAnimationFrame(() => target.focus());
    document.body.style.overflow = 'hidden';
}

function closeModal(modalId) {
    const modal = document.getElementById(modalId);
    if (!modal) return;
    modal.classList.add('hidden');
    modal.setAttribute('aria-hidden', 'true');
    document.body.style.overflow = '';
    if (_prevFocus) { requestAnimationFrame(() => { _prevFocus.focus(); _prevFocus = null; }); }
}

function hideElement(elementId) {
    const el = document.getElementById(elementId);
    if (!el) return;
    el.classList.add('hidden');
    el.setAttribute('aria-hidden', 'true');
    el.inert = true;
}

function showElement(elementId) {
    const el = document.getElementById(elementId);
    if (!el) return;
    el.classList.remove('hidden');
    el.setAttribute('aria-hidden', 'false');
    el.inert = false;
}

function showAuthScreen() { showElement('auth-screen'); hideElement('main-screen'); }

function showMainScreen() {
    hideElement('auth-screen');
    showElement('main-screen');
    document.getElementById('user-name').textContent = currentUser.name;
    document.querySelectorAll('.admin-only').forEach(el => {
        if (currentUser.is_admin) { el.classList.remove('hidden'); el.setAttribute('aria-hidden', 'false'); el.inert = false; }
        else { el.classList.add('hidden'); el.setAttribute('aria-hidden', 'true'); el.inert = true; }
    });
    if (currentUser.is_admin) loadPendingStandards();
    loadNotificationBadge();
}

function switchView(viewName) {
    document.querySelectorAll('.nav-btn').forEach(btn => btn.classList.remove('active'));
    document.querySelectorAll('.view').forEach(view => {
        view.classList.remove('active');
        view.classList.add('hidden');
        view.setAttribute('aria-hidden', 'true');
    });
    const navBtn = document.querySelector(`[data-view="${viewName}"]`);
    if (navBtn) navBtn.classList.add('active');
    const activeView = document.getElementById(`${viewName}-view`);
    if (activeView) { activeView.classList.add('active'); activeView.classList.remove('hidden'); activeView.setAttribute('aria-hidden', 'false'); }
    if (viewName === 'all-standards') loadAllStandards();
    else if (viewName === 'categories') loadCategories();
    else if (viewName === 'settings') loadSettings();
    else if (viewName === 'pending') loadPendingStandards(1);
    else if (viewName === 'friends') loadFriends();
    else if (viewName === 'notifications') loadNotifications();
}

async function handleLogin(e) {
    e.preventDefault();
    const username = document.getElementById('login-username').value;
    const password = document.getElementById('login-password').value;
    const errorEl = document.getElementById('login-error');
    errorEl.textContent = '';
    try {
        currentUser = await API.login(username, password);
        showMainScreen();
        await loadMyStandards();
    } catch (error) { errorEl.textContent = error.message; }
}

async function handlePasskeyLogin() {
    if (!window.PublicKeyCredential) { alert('WebAuthn / passkeys are not supported in this browser.'); return; }
    const username = document.getElementById('login-username').value.trim();
    const errorEl = document.getElementById('login-error');
    errorEl.textContent = '';
    if (!username) { errorEl.textContent = 'Enter your username first, then click "Login with Passkey".'; return; }
    try {
        const options = await API.beginPasskeyAuth(username);
        const requestOptions = prepareRequestOptions(options);
        let assertion;
        try { assertion = await navigator.credentials.get(requestOptions); }
        catch (credErr) {
            errorEl.textContent = credErr.name === 'NotAllowedError' ? 'Passkey login was cancelled or timed out.' : 'Passkey login failed: ' + credErr.message;
            return;
        }
        const assertionJSON = serializeAssertion(assertion);
        const data = await API.finishPasskeyAuth(username, assertionJSON);
        if (data.token) API.setToken(data.token);
        currentUser = data.user;
        showMainScreen();
        await loadMyStandards();
    } catch (error) { errorEl.textContent = error.message; }
}

async function handleRegister(e) {
    e.preventDefault();
    const name = document.getElementById('register-name').value;
    const username = document.getElementById('register-username').value;
    const password = document.getElementById('register-password').value;
    const errorEl = document.getElementById('register-error');
    errorEl.textContent = '';
    try { currentUser = await API.register(username, name, password); showMainScreen(); await loadMyStandards(); }
    catch (error) { errorEl.textContent = error.message; }
}

async function handleLogout() {
    try { await API.logout(); } catch (e) { console.error('Logout error:', e); }
    currentUser = null;
    showAuthScreen();
}

async function loadMyStandards() {
    try {
        const data = await API.getMyStandards();
        myStandards = data.standards || [];
        renderMyStandards();
    } catch (error) {
        if (error.message === 'Unauthorized' || error.message === 'User not found') {
            API.clearToken(); currentUser = null; showAuthScreen(); return;
        }
        console.error('Failed to load my standards:', error);
    }
}

async function loadAllStandards(page = 1) {
    const search = document.getElementById('search-input').value;
    const style = document.getElementById('style-filter').value;
    try {
        const data = await API.getStandards(page, 100, search, style);
        allStandards = data.standards || [];
        currentPage = page;
        renderAllStandards(data);
    } catch (error) { console.error('Failed to load standards:', error); }
}

async function loadCategories() {
    try { categories = await API.getCategories(); renderCategories(); }
    catch (error) { console.error('Failed to load categories:', error); }
}

async function loadPendingStandards(page = 1) {
    if (!currentUser || !currentUser.is_admin) return;
    try { const data = await API.getPendingStandards(page, 100); renderPendingStandards(data); }
    catch (error) { console.error('Failed to load pending standards:', error); }
}

async function loadFriends() {
    try { const following = await API.getFollowing(); renderFollowing(following); }
    catch (error) { console.error('Failed to load friends:', error); }
}

async function loadNotifications() {
    try {
        const notifications = await API.getNotifications();
        renderNotifications(notifications);
        updateNotifBadge(0);
    } catch (error) { console.error('Failed to load notifications:', error); }
}

async function loadNotificationBadge() {
    try {
        const notifications = await API.getNotifications();
        const unread = notifications.filter(n => !n.read).length;
        updateNotifBadge(unread);
    } catch (e) { /* silent */ }
}

function updateNotifBadge(count) {
    const badge = document.getElementById('notif-badge');
    if (!badge) return;
    if (count > 0) { badge.textContent = count; badge.classList.remove('hidden'); }
    else { badge.classList.add('hidden'); }
}

function renderMyStandards() {
    const container = document.getElementById('standards-by-category');
    container.innerHTML = '';

    const styleFilter = document.getElementById('my-style-filter').value;
    const composerFilter = document.getElementById('my-composer-filter').value.toLowerCase().trim();
    const profFilter = document.getElementById('my-proficiency-filter').value;

    let filtered = myStandards.filter(us => {
        const std = us.jazz_standard;
        if (!std) return false;
        if (styleFilter && std.style !== styleFilter) return false;
        if (composerFilter && !std.composer.toLowerCase().includes(composerFilter)) return false;
        if (profFilter && us.proficiency !== profFilter) return false;
        return true;
    });

    if (filtered.length === 0) {
        container.innerHTML = `<p class="muted">${t('my_standards_empty')}</p>`;
        return;
    }

    const grouped = {};
    const uncategorized = [];
    for (const us of filtered) {
        if (us.category) { (grouped[us.category.name] = grouped[us.category.name] || []).push(us); }
        else { uncategorized.push(us); }
    }
    if (uncategorized.length > 0) grouped['Uncategorized'] = uncategorized;

    for (const [categoryName, standards] of Object.entries(grouped)) {
        const categoryDiv = document.createElement('div');
        categoryDiv.className = 'category-group';
        const category = standards[0].category;
        const colorHtml = category ? `<span class="category-color" style="background:${category.color}"></span>` : '';
        categoryDiv.innerHTML = `
            <h3>${colorHtml}${escapeHtml(categoryName)}</h3>
            ${standards.map(us => `
                <div class="standard-item" data-id="${us.jazz_standard_id}">
                    <div class="standard-info">
                        <h4>${escapeHtml(us.jazz_standard.title)}</h4>
                        <p>${escapeHtml(us.jazz_standard.composer)} · ${escapeHtml(us.jazz_standard.style)}</p>
                        ${us.notes ? `<p><em>${escapeHtml(us.notes)}</em></p>` : ''}
                    </div>
                    <div class="standard-actions">
                        <span class="proficiency-badge proficiency-${us.proficiency}">${t('prof_' + us.proficiency)}</span>
                        <button class="btn-remove" data-action="remove" data-id="${us.jazz_standard_id}"
                            aria-label="${t('remove_from_list')} ${escapeHtml(us.jazz_standard.title)}">${t('remove_from_list')}</button>
                    </div>
                </div>
            `).join('')}
        `;
        container.appendChild(categoryDiv);
    }
}

function renderAllStandards(data) {
    const container = document.getElementById('all-standards-list');
    const standards = data.standards || [];
    if (standards.length === 0) { container.innerHTML = `<p class="muted">${t('no_standards_found')}</p>`; return; }
    container.innerHTML = standards.map(standard => {
        const isKnown = myStandards.some(us => us.jazz_standard_id === standard.id);
        const isPending = standard.status === 'pending';
        const isRejected = standard.status === 'rejected';
        let statusBadge = '';
        if (isPending) statusBadge = '<span style="color:#f0c040;font-weight:600;font-size:0.8rem;">⏳ Pending</span>';
        else if (isRejected) statusBadge = '<span style="color:#e05252;font-weight:600;font-size:0.8rem;">✗ Rejected</span>';
        const titleId = `standard-title-${standard.id}`;
        const addBtn = `<button class="btn-know" data-action="add" data-id="${standard.id}" aria-label="${t('i_know_this')} ${escapeHtml(standard.title)}">${t('i_know_this')}</button>`;
        const removeBtn = `<button class="btn-remove" data-action="remove" data-id="${standard.id}" aria-label="${t('remove')} ${escapeHtml(standard.title)}">${t('remove')}</button>`;
        const deleteBtn = `<button class="btn-remove" data-action="delete" data-id="${standard.id}" aria-label="${t('delete')} ${escapeHtml(standard.title)}">${t('delete')}</button>`;
        const actionBtn = isKnown ? removeBtn : addBtn;
        return `
            <div class="standard-item" tabindex="0" role="article" aria-labelledby="${titleId}" data-id="${standard.id}">
                <div class="standard-info">
                    <h4 id="${titleId}">${escapeHtml(standard.title)} ${statusBadge}</h4>
                    <p>${escapeHtml(standard.composer)} · ${escapeHtml(standard.style)}</p>
                    ${standard.additional_note ? `<p><em>${escapeHtml(standard.additional_note)}</em></p>` : ''}
                    ${standard.creator ? `<p style="font-size:0.8rem;color:var(--text-dim);">${t('submitted_by')} ${escapeHtml(standard.creator.name)}</p>` : ''}
                </div>
                <div class="standard-actions">
                    ${!isPending && !isRejected ? actionBtn : ''}
                    ${currentUser.is_admin ? deleteBtn : ''}
                </div>
            </div>
        `;
    }).join('');
    renderPagination(data);
}

function renderPagination(data) {
    const container = document.getElementById('pagination');
    const totalPages = Math.ceil(data.total / data.limit);
    if (totalPages <= 1) { container.innerHTML = ''; return; }
    container.innerHTML = `
        <button ${data.page === 1 ? 'disabled' : ''} data-action="paginate" data-page="${data.page - 1}" aria-label="Previous page">← Previous</button>
        <span>Page ${data.page} of ${totalPages}</span>
        <button ${data.page === totalPages ? 'disabled' : ''} data-action="paginate" data-page="${data.page + 1}" aria-label="Next page">Next →</button>
    `;
}

function renderPendingStandards(data) {
    const container = document.getElementById('pending-standards-list');
    if (!container) return;
    const standards = data.standards || [];
    if (standards.length === 0) { container.innerHTML = '<p class="muted">No pending standards — you\'re all caught up!</p>'; return; }
    container.innerHTML = standards.map(standard => `
        <div class="standard-item pending-item" data-id="${standard.id}">
            <div class="standard-info">
                <h4>${escapeHtml(standard.title)}</h4>
                <p>${escapeHtml(standard.composer)} · ${escapeHtml(standard.style)}</p>
                ${standard.additional_note ? `<p><em>${escapeHtml(standard.additional_note)}</em></p>` : ''}
                <p style="font-size:0.8rem;color:var(--text-dim);">
                    ${t('submitted_by')} ${standard.creator ? escapeHtml(standard.creator.name) : 'Unknown'} ·
                    ${new Date(standard.created_at).toLocaleDateString()}
                </p>
            </div>
            <div class="standard-actions">
                <button class="btn-approve" data-action="approve" data-id="${standard.id}">${t('approve')}</button>
                <button class="btn-reject" data-action="reject" data-id="${standard.id}">${t('reject')}</button>
                <button class="btn-remove" data-action="delete" data-id="${standard.id}">${t('delete')}</button>
            </div>
        </div>
    `).join('');
}

function renderFollowing(users) {
    const container = document.getElementById('following-list');
    if (!container) return;
    if (!users || users.length === 0) { container.innerHTML = `<p class="muted">${t('no_following')}</p>`; return; }
    container.innerHTML = users.map(u => `
        <div class="standard-item">
            <div class="standard-info">
                <h4><a href="#" class="profile-link" data-username="${escapeHtml(u.username)}">@${escapeHtml(u.username)}</a></h4>
                <p>${escapeHtml(u.name)}</p>
            </div>
            <div class="standard-actions">
                <button class="btn-remove" data-action="unfollow" data-username="${escapeHtml(u.username)}">${t('unfollow')}</button>
            </div>
        </div>
    `).join('');
    container.querySelectorAll('.profile-link').forEach(link => {
        link.addEventListener('click', e => { e.preventDefault(); loadPublicProfile(link.dataset.username); });
    });
}

async function handleFriendSearch() {
    const q = document.getElementById('friend-search-input').value.trim();
    if (!q) return;
    const container = document.getElementById('friend-search-results');
    container.innerHTML = '<p class="muted">Searching…</p>';
    try {
        const user = await API.getUserByUsername(q);
        const isFollowingResult = await API.isFollowing(q);
        container.innerHTML = `
            <div class="standard-item">
                <div class="standard-info">
                    <h4><a href="#" class="profile-link" data-username="${escapeHtml(user.username)}">@${escapeHtml(user.username)}</a></h4>
                    <p>${escapeHtml(user.name)}</p>
                </div>
                <div class="standard-actions">
                    ${isFollowingResult.following
                        ? `<button class="btn-remove" data-action="unfollow" data-username="${escapeHtml(user.username)}">${t('unfollow')}</button>`
                        : `<button class="btn-know" data-action="follow" data-username="${escapeHtml(user.username)}">${t('follow')}</button>`
                    }
                    <button class="btn-secondary" data-action="view-profile" data-username="${escapeHtml(user.username)}" style="font-size:0.85rem;">View Profile</button>
                </div>
            </div>
        `;
        const pl = container.querySelector('.profile-link');
        if (pl) pl.addEventListener('click', e => { e.preventDefault(); loadPublicProfile(user.username); });
    } catch (error) {
        container.innerHTML = `<p class="muted">${t('no_results')}</p>`;
    }
}

async function followUser(username) {
    try { await API.followUser(username); await loadFriends(); }
    catch (error) { alert(error.message); }
}

async function unfollowUser(username) {
    try {
        await API.unfollowUser(username);
        await loadFriends();
        const results = document.getElementById('friend-search-results');
        if (results) results.innerHTML = '';
    } catch (error) { alert(error.message); }
}

async function loadPublicProfile(username) {
    const container = document.getElementById('friend-search-results');
    if (container) container.innerHTML = '<p class="muted">Loading profile…</p>';
    try {
        const data = await API.getPublicStandards(username);
        renderProfileView(username, data);
    } catch (error) {
        if (container) container.innerHTML = `<p class="muted">${error.message === "This user's list is private" ? t('private_profile') : t('no_results')}</p>`;
    }
}

function renderProfileView(username, data) {
    const container = document.getElementById('friend-search-results');
    if (!container) return;
    const standards = data.standards || [];
    const user = data.user || {};
    container.innerHTML = `
        <div style="margin-bottom:1rem;padding-bottom:0.75rem;border-bottom:1px solid var(--border);">
            <h3 style="margin:0;">@${escapeHtml(username)} <span style="font-weight:400;color:var(--text-dim);">— ${escapeHtml(user.name || '')}</span></h3>
        </div>
        ${standards.length === 0 ? `<p class="muted">${t('my_standards_empty')}</p>` : ''}
        ${standards.map(us => `
            <div class="standard-item">
                <div class="standard-info">
                    <h4>${escapeHtml(us.jazz_standard ? us.jazz_standard.title : '')}</h4>
                    <p>${escapeHtml(us.jazz_standard ? us.jazz_standard.composer : '')} · ${escapeHtml(us.jazz_standard ? us.jazz_standard.style : '')}</p>
                </div>
                <div class="standard-actions">
                    <span class="proficiency-badge proficiency-${us.proficiency}">${t('prof_' + us.proficiency)}</span>
                </div>
            </div>
        `).join('')}
    `;
}

function renderNotifications(notifications) {
    const container = document.getElementById('notifications-list');
    if (!container) return;
    if (!notifications || notifications.length === 0) { container.innerHTML = `<p class="muted">${t('no_notifications')}</p>`; return; }
    container.innerHTML = notifications.map(n => {
        const actor = n.actor ? `@${escapeHtml(n.actor.username)}` : 'Someone';
        const standard = n.jazz_standard ? escapeHtml(n.jazz_standard.title) : '';
        const action = n.action_type === 'mastered' ? t('notif_mastered') : t('notif_learned');
        const time = new Date(n.created_at).toLocaleString();
        return `
            <div class="standard-item ${n.read ? '' : 'notif-unread'}">
                <div class="standard-info">
                    <p><strong>${actor}</strong> ${action} <em>${standard}</em></p>
                    <p style="font-size:0.82rem;color:var(--text-dim);">${time}</p>
                </div>
            </div>
        `;
    }).join('');
}

async function handleMarkNotificationsRead() {
    try { await API.markNotificationsRead(); await loadNotifications(); }
    catch (error) { alert(error.message); }
}

async function loadSettings() {
    try {
        const user = await API.getMe();
        const nameEl = document.getElementById('profile-name');
        const usernameEl = document.getElementById('profile-username');
        const publicEl = document.getElementById('profile-public');
        if (nameEl) nameEl.value = user.name || '';
        if (usernameEl) usernameEl.value = user.username || '';
        if (publicEl) publicEl.checked = !!user.public_profile;
        const keys = await API.listPassKeys();
        renderPassKeys(keys || []);
    } catch (error) { console.error('Failed to load settings:', error); }
}

function renderPassKeys(keys) {
    const container = document.getElementById('passkeys-list');
    if (!container) return;
    if (!keys || keys.length === 0) { container.innerHTML = `<p class="muted">${t('no_passkeys')}</p>`; return; }
    container.innerHTML = keys.map(k => `
        <div class="passkey-item" data-id="${k.id}">
            <strong>${escapeHtml(k.name)}</strong>
            <span style="color:var(--text-dim);font-size:0.85rem;"> (${escapeHtml(k.token_hint || '')})</span>
            <div style="font-size:0.82rem;color:var(--text-dim);margin-top:2px;">Created ${new Date(k.created_at).toLocaleString()}</div>
            <button data-action="delete-passkey" data-id="${k.id}" aria-label="${t('revoke')} ${escapeHtml(k.name)}">${t('revoke')}</button>
        </div>
    `).join('');
}

async function handleUpdateProfile(e) {
    e.preventDefault();
    const name = document.getElementById('profile-name').value;
    const username = document.getElementById('profile-username').value;
    const publicProfile = document.getElementById('profile-public').checked;
    try {
        const updated = await API.updateMe({ name: name || null, username: username || null, public_profile: publicProfile });
        currentUser = updated;
        showMainScreen();
        alert(t('save_changes') + ' ✓');
    } catch (error) { alert(error.message); }
}

function base64urlToBuffer(base64url) {
    const base64 = base64url.replace(/-/g, '+').replace(/_/g, '/');
    const binary = atob(base64);
    const buffer = new Uint8Array(binary.length);
    for (let i = 0; i < binary.length; i++) buffer[i] = binary.charCodeAt(i);
    return buffer.buffer;
}

function bufferToBase64url(buffer) {
    const bytes = new Uint8Array(buffer);
    let str = '';
    for (const b of bytes) str += String.fromCharCode(b);
    return btoa(str).replace(/\+/g, '-').replace(/\//g, '_').replace(/=/g, '');
}

function prepareCreationOptions(opts) {
    const pk = opts.publicKey || opts;
    pk.challenge = base64urlToBuffer(pk.challenge);
    pk.user.id = base64urlToBuffer(pk.user.id);
    if (pk.excludeCredentials) pk.excludeCredentials = pk.excludeCredentials.map(c => ({ ...c, id: base64urlToBuffer(c.id) }));
    return { publicKey: pk };
}

function prepareRequestOptions(opts) {
    const pk = opts.publicKey || opts;
    pk.challenge = base64urlToBuffer(pk.challenge);
    if (pk.allowCredentials) pk.allowCredentials = pk.allowCredentials.map(c => ({ ...c, id: base64urlToBuffer(c.id) }));
    return { publicKey: pk };
}

function serializeCredential(cred) {
    return { id: cred.id, rawId: bufferToBase64url(cred.rawId), type: cred.type, response: { attestationObject: bufferToBase64url(cred.response.attestationObject), clientDataJSON: bufferToBase64url(cred.response.clientDataJSON) } };
}

function serializeAssertion(cred) {
    return { id: cred.id, rawId: bufferToBase64url(cred.rawId), type: cred.type, response: { authenticatorData: bufferToBase64url(cred.response.authenticatorData), clientDataJSON: bufferToBase64url(cred.response.clientDataJSON), signature: bufferToBase64url(cred.response.signature), userHandle: cred.response.userHandle ? bufferToBase64url(cred.response.userHandle) : null } };
}

async function handleCreatePassKey(e) {
    e.preventDefault();
    if (!window.PublicKeyCredential) { alert('WebAuthn / passkeys are not supported in this browser.\nUse Chrome, Safari, Firefox, or Edge on a modern OS.'); return; }
    const name = document.getElementById('new-passkey-name').value.trim();
    if (!name) { alert('Please enter a name for this passkey.'); return; }
    try {
        const options = await API.beginPasskeyRegistration();
        const creationOptions = prepareCreationOptions(options);
        let credential;
        try { credential = await navigator.credentials.create(creationOptions); }
        catch (credErr) {
            alert(credErr.name === 'NotAllowedError' ? 'Passkey creation was cancelled or timed out.' : 'Passkey creation failed: ' + credErr.message);
            return;
        }
        const credJSON = serializeCredential(credential);
        await API.finishPasskeyRegistration(name, credJSON);
        alert(`Passkey "${name}" registered! You can now log in with biometrics.`);
        document.getElementById('create-passkey-form').reset();
        await loadSettings();
    } catch (error) { alert('Error registering passkey: ' + error.message); }
}

async function deletePassKey(id) {
    if (!confirm(t('revoke') + '?')) return;
    try { await API.deletePassKey(id); await loadSettings(); }
    catch (error) { alert(error.message); }
}

function renderCategories() {
    const container = document.getElementById('categories-list');
    if (categories.length === 0) { container.innerHTML = `<p class="muted">${t('no_categories')}</p>`; return; }
    container.innerHTML = categories.map(cat => `
        <div class="category-item" data-id="${cat.id}">
            <h3><span class="category-color" style="background:${cat.color}"></span>${escapeHtml(cat.name)}</h3>
            <button class="btn-remove" data-action="delete-category" data-id="${cat.id}" aria-label="${t('delete_category')} ${escapeHtml(cat.name)}">${t('delete_category')}</button>
        </div>
    `).join('');
}

async function addStandard(standardId) {
    try { await API.addToMyStandards(standardId); await loadMyStandards(); if (!document.getElementById('all-standards-view').classList.contains('hidden')) await loadAllStandards(currentPage); }
    catch (error) { alert(error.message); }
}

async function removeStandard(standardId) {
    if (!confirm('Remove this standard from your list?')) return;
    try { await API.removeFromMyStandards(standardId); await loadMyStandards(); if (!document.getElementById('all-standards-view').classList.contains('hidden')) await loadAllStandards(currentPage); }
    catch (error) { alert(error.message); }
}

async function deleteStandard(standardId) {
    if (!confirm('Delete this standard permanently?')) return;
    try { await API.deleteStandard(standardId); await loadAllStandards(currentPage); }
    catch (error) { alert(error.message); }
}

async function handleAddStandard(e) {
    e.preventDefault();
    const title = document.getElementById('new-standard-title').value;
    const composer = document.getElementById('new-standard-composer').value;
    const style = document.getElementById('new-standard-style').value;
    const note = document.getElementById('new-standard-note').value;
    try {
        const response = await API.createStandard(title, composer, style, note);
        closeModal('add-standard-modal');
        e.target.reset();
        if (response.message) alert(response.message);
        await loadAllStandards(currentPage);
        if (currentUser.is_admin) loadPendingStandards();
    } catch (error) { alert(error.message); }
}

async function handleAddCategory(e) {
    e.preventDefault();
    const name = document.getElementById('new-category-name').value;
    const color = document.getElementById('new-category-color').value;
    try { await API.createCategory(name, color); closeModal('add-category-modal'); e.target.reset(); await loadCategories(); }
    catch (error) { alert(error.message); }
}

async function deleteCategory(categoryId) {
    if (!confirm('Delete this category? Standards will be uncategorized.')) return;
    try { await API.deleteCategory(categoryId); await loadCategories(); await loadMyStandards(); }
    catch (error) { alert(error.message); }
}

async function approveStandard(id) {
    if (!confirm('Approve this standard?')) return;
    try { await API.approveStandard(id); await loadPendingStandards(); await loadAllStandards(currentPage); alert('Standard approved!'); }
    catch (error) { alert(error.message); }
}

async function rejectStandard(id) {
    if (!confirm('Reject this standard?')) return;
    try { await API.rejectStandard(id); await loadPendingStandards(); await loadAllStandards(currentPage); alert('Standard rejected.'); }
    catch (error) { alert(error.message); }
}

function debounce(func, wait) {
    let timeout;
    return function (...args) { clearTimeout(timeout); timeout = setTimeout(() => func(...args), wait); };
}

function checkOnlineStatus() {
    const indicator = document.getElementById('offline-indicator');
    function update() {
        if (navigator.onLine) { indicator.classList.add('hidden'); indicator.setAttribute('aria-hidden', 'true'); }
        else { indicator.classList.remove('hidden'); indicator.setAttribute('aria-hidden', 'false'); }
    }
    window.addEventListener('online', update);
    window.addEventListener('offline', update);
    update();
}

document.addEventListener('DOMContentLoaded', async () => {
    await init();
    window.__testExports = { openModal, closeModal };
});

window.addStandard = addStandard;
window.removeStandard = removeStandard;
window.deleteStandard = deleteStandard;
window.approveStandard = approveStandard;
window.rejectStandard = rejectStandard;
window.deleteCategory = deleteCategory;
window.deletePassKey = deletePassKey;
window.loadAllStandards = loadAllStandards;
window.followUser = followUser;
window.unfollowUser = unfollowUser;
window.loadPublicProfile = loadPublicProfile;
