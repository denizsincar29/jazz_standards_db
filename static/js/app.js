import API from './api.js';
import Dom from './dom.js';

// App state
let currentUser = null;
let allStandards = [];
let myStandards = [];
let categories = [];
let currentPage = 1;

function debugPrint(...args) {
    if (window.DEBUG) console.log('[DEBUG]', ...args);
}

// ---- HTML escaping ----
function escapeHtml(str) {
    if (!str && str !== 0) return '';
    return String(str).replace(/[&<>"'`]/g, s => ({
        '&': '&amp;', '<': '&lt;', '>': '&gt;',
        '"': '&quot;', "'": '&#39;', '`': '&#96;'
    })[s]);
}

// ---- Init ----
async function init() {
    const token = API.getToken();
    if (token) {
        try {
            currentUser = await API.getMe();
            showMainScreen();
            await loadMyStandards();
        } catch (error) {
            console.error('Failed to load user:', error);
            API.clearToken();
            showAuthScreen();
        }
    } else {
        showAuthScreen();
    }
    setupEventListeners();
    checkOnlineStatus();
}

// ---- Event listeners ----
function setupEventListeners() {
    // Auth tabs
    document.querySelectorAll('.tab').forEach(tab => {
        tab.addEventListener('click', e => {
            const tabName = e.target.dataset.tab;
            document.querySelectorAll('.tab').forEach(t => {
                t.classList.remove('active');
                t.setAttribute('aria-selected', 'false');
            });
            document.querySelectorAll('.auth-form').forEach(f => f.classList.remove('active'));
            e.target.classList.add('active');
            e.target.setAttribute('aria-selected', 'true');
            document.getElementById(`${tabName}-form`).classList.add('active');
        });
    });

    // Auth forms
    document.getElementById('login-form').addEventListener('submit', handleLogin);
    document.getElementById('register-form').addEventListener('submit', handleRegister);

    // Logout
    document.getElementById('logout-btn').addEventListener('click', handleLogout);

    // Navigation
    document.querySelectorAll('.nav-btn').forEach(btn => {
        btn.addEventListener('click', e => switchView(e.target.dataset.view));
    });

    // Settings forms
    const profileForm = document.getElementById('update-profile-form');
    if (profileForm) profileForm.addEventListener('submit', handleUpdateProfile);
    const passkeyForm = document.getElementById('create-passkey-form');
    if (passkeyForm) passkeyForm.addEventListener('submit', handleCreatePassKey);

    // Search and filter
    document.getElementById('search-input').addEventListener('input', debounce(loadAllStandards, 300));
    document.getElementById('style-filter').addEventListener('change', loadAllStandards);

    // Open modals
    document.getElementById('add-standard-btn').addEventListener('click', () => openModal('add-standard-modal'));
    document.getElementById('add-category-btn').addEventListener('click', () => openModal('add-category-modal'));

    // Modal forms
    document.getElementById('add-standard-form').addEventListener('submit', handleAddStandard);
    document.getElementById('add-category-form').addEventListener('submit', handleAddCategory);

    // Color preview label
    const colorInput = document.getElementById('new-category-color');
    const colorLabel = document.getElementById('color-preview-label');
    if (colorInput && colorLabel) {
        colorInput.addEventListener('input', () => {
            colorLabel.textContent = colorInput.value;
        });
    }

    // Close modal via data-close-modal buttons/backdrop, or click outside modal-content
    document.addEventListener('click', e => {
        const closeTarget = e.target.closest('[data-close-modal]');
        if (closeTarget) {
            const modalId = closeTarget.dataset.closeModal;
            closeModal(modalId);
            return;
        }
        // If the click landed directly on the .modal overlay (not inside .modal-content)
        if (e.target.classList.contains('modal')) {
            closeModal(e.target.id);
        }
    });

    // Close modal on Escape key
    document.addEventListener('keydown', e => {
        if (e.key === 'Escape') {
            const openModal = document.querySelector('.modal:not(.hidden)');
            if (openModal) closeModal(openModal.id);
        }
    });

    // Delegated handlers for dynamic content
    Dom.attachDelegation();
}

// ---- Modal management ----
let _prevFocus = null;

function openModal(modalId) {
    const modal = document.getElementById(modalId);
    if (!modal) return;

    _prevFocus = document.activeElement;

    modal.classList.remove('hidden');
    modal.setAttribute('aria-hidden', 'false');
    modal.removeAttribute('inert');

    // Trap focus: move focus to first focusable element inside modal
    const focusable = modal.querySelectorAll(
        'button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])'
    );
    // Skip backdrop (first), find first real focusable
    const firstReal = Array.from(focusable).find(el => !el.hasAttribute('data-close-modal') || el.tagName === 'BUTTON' && !el.classList.contains('modal-close'));
    const target = firstReal || focusable[0];
    if (target) requestAnimationFrame(() => target.focus());

    // Prevent body scroll
    document.body.style.overflow = 'hidden';

    // Announce to screen reader
    modal.setAttribute('aria-hidden', 'false');
}

function closeModal(modalId) {
    const modal = document.getElementById(modalId);
    if (!modal) return;

    modal.classList.add('hidden');
    modal.setAttribute('aria-hidden', 'true');

    document.body.style.overflow = '';

    // Restore focus
    if (_prevFocus) {
        requestAnimationFrame(() => {
            _prevFocus.focus();
            _prevFocus = null;
        });
    }
}

// ---- Accessibility helpers ----
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

// ---- Screen switching ----
function showAuthScreen() {
    showElement('auth-screen');
    hideElement('main-screen');
}

function showMainScreen() {
    hideElement('auth-screen');
    showElement('main-screen');
    document.getElementById('user-name').textContent = currentUser.name;

    document.querySelectorAll('.admin-only').forEach(el => {
        if (currentUser.is_admin) {
            el.classList.remove('hidden');
            el.setAttribute('aria-hidden', 'false');
            el.inert = false;
        } else {
            el.classList.add('hidden');
            el.setAttribute('aria-hidden', 'true');
            el.inert = true;
        }
    });

    if (currentUser.is_admin) loadPendingStandards();
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
    if (activeView) {
        activeView.classList.add('active');
        activeView.classList.remove('hidden');
        activeView.setAttribute('aria-hidden', 'false');
    }

    if (viewName === 'all-standards') loadAllStandards();
    else if (viewName === 'categories') loadCategories();
    else if (viewName === 'settings') loadSettings();
    else if (viewName === 'pending') loadPendingStandards(1);
}

// ---- Auth handlers ----
async function handleLogin(e) {
    e.preventDefault();
    const username = document.getElementById('login-username').value;
    const password = document.getElementById('login-password').value;
    const errorEl = document.getElementById('login-error');
    errorEl.textContent = '';
    try {
        debugPrint('handleLogin:', username);
        currentUser = await API.login(username, password);
        showMainScreen();
        await loadMyStandards();
    } catch (error) {
        debugPrint('Login error:', error);
        errorEl.textContent = error.message;
    }
}

async function handleRegister(e) {
    e.preventDefault();
    const name = document.getElementById('register-name').value;
    const username = document.getElementById('register-username').value;
    const password = document.getElementById('register-password').value;
    const errorEl = document.getElementById('register-error');
    errorEl.textContent = '';
    try {
        currentUser = await API.register(username, name, password);
        showMainScreen();
        await loadMyStandards();
    } catch (error) {
        errorEl.textContent = error.message;
    }
}

async function handleLogout() {
    try { await API.logout(); } catch (e) { console.error('Logout error:', e); }
    currentUser = null;
    showAuthScreen();
}

// ---- Data loading ----
async function loadMyStandards() {
    try {
        const data = await API.getMyStandards();
        myStandards = data.standards || [];
        renderMyStandards(data.grouped || {});
    } catch (error) {
        if (error.message === 'Unauthorized' || error.message === 'User not found') {
            API.clearToken();
            currentUser = null;
            showAuthScreen();
            return;
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
    } catch (error) {
        console.error('Failed to load standards:', error);
    }
}

async function loadCategories() {
    try {
        categories = await API.getCategories();
        renderCategories();
    } catch (error) {
        console.error('Failed to load categories:', error);
    }
}

async function loadPendingStandards(page = 1) {
    if (!currentUser || !currentUser.is_admin) return;
    try {
        const data = await API.getPendingStandards(page, 100);
        renderPendingStandards(data);
    } catch (error) {
        console.error('Failed to load pending standards:', error);
    }
}

// ---- Render functions ----
function renderMyStandards(grouped) {
    const container = document.getElementById('standards-by-category');
    container.innerHTML = '';

    if (Object.keys(grouped).length === 0) {
        container.innerHTML = '<p class="muted">No standards yet. Add some from the "All Standards" tab!</p>';
        return;
    }

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
                        <button class="btn-remove" data-action="remove" data-id="${us.jazz_standard_id}"
                            aria-label="Remove ${escapeHtml(us.jazz_standard.title)}">Remove</button>
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

    if (standards.length === 0) {
        container.innerHTML = '<p class="muted">No standards found.</p>';
        return;
    }

    container.innerHTML = standards.map(standard => {
        const isKnown = myStandards.some(us => us.jazz_standard_id === standard.id);
        const isPending = standard.status === 'pending';
        const isRejected = standard.status === 'rejected';

        let statusBadge = '';
        if (isPending) statusBadge = '<span style="color:#f0c040;font-weight:600;font-size:0.8rem;">⏳ Pending</span>';
        else if (isRejected) statusBadge = '<span style="color:#e05252;font-weight:600;font-size:0.8rem;">✗ Rejected</span>';

        const titleId = `standard-title-${standard.id}`;
        const addBtn = `<button class="btn-know" data-action="add" data-id="${standard.id}" aria-label="I know ${escapeHtml(standard.title)}">I Know This</button>`;
        const removeBtn = `<button class="btn-remove" data-action="remove" data-id="${standard.id}" aria-label="Remove ${escapeHtml(standard.title)}">Remove</button>`;
        const deleteBtn = `<button class="btn-remove" data-action="delete" data-id="${standard.id}" aria-label="Delete ${escapeHtml(standard.title)}">Delete</button>`;
        const actionBtn = isKnown ? removeBtn : addBtn;

        return `
            <div class="standard-item" tabindex="0" role="article" aria-labelledby="${titleId}" data-id="${standard.id}">
                <div class="standard-info">
                    <h4 id="${titleId}">${escapeHtml(standard.title)} ${statusBadge}</h4>
                    <p>${escapeHtml(standard.composer)} · ${escapeHtml(standard.style)}</p>
                    ${standard.additional_note ? `<p><em>${escapeHtml(standard.additional_note)}</em></p>` : ''}
                    ${standard.creator ? `<p style="font-size:0.8rem;color:var(--text-dim);">Submitted by ${escapeHtml(standard.creator.name)}</p>` : ''}
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
    if (standards.length === 0) {
        container.innerHTML = '<p class="muted">No pending standards — you\'re all caught up!</p>';
        return;
    }

    container.innerHTML = standards.map(standard => `
        <div class="standard-item pending-item" data-id="${standard.id}">
            <div class="standard-info">
                <h4>${escapeHtml(standard.title)}</h4>
                <p>${escapeHtml(standard.composer)} · ${escapeHtml(standard.style)}</p>
                ${standard.additional_note ? `<p><em>${escapeHtml(standard.additional_note)}</em></p>` : ''}
                <p style="font-size:0.8rem;color:var(--text-dim);">
                    By ${standard.creator ? escapeHtml(standard.creator.name) : 'Unknown'} ·
                    ${new Date(standard.created_at).toLocaleDateString()}
                </p>
            </div>
            <div class="standard-actions">
                <button class="btn-approve" data-action="approve" data-id="${standard.id}" aria-label="Approve ${escapeHtml(standard.title)}">✓ Approve</button>
                <button class="btn-reject" data-action="reject" data-id="${standard.id}" aria-label="Reject ${escapeHtml(standard.title)}">✗ Reject</button>
                <button class="btn-remove" data-action="delete" data-id="${standard.id}" aria-label="Delete ${escapeHtml(standard.title)}">Delete</button>
            </div>
        </div>
    `).join('');
}

// ---- Settings / Passkeys ----
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
    } catch (error) {
        console.error('Failed to load settings:', error);
    }
}

function renderPassKeys(keys) {
    const container = document.getElementById('passkeys-list');
    if (!container) return;
    if (!keys || keys.length === 0) {
        container.innerHTML = '<p class="muted">No pass keys yet.</p>';
        return;
    }
    container.innerHTML = keys.map(k => `
        <div class="passkey-item" data-id="${k.id}">
            <strong>${escapeHtml(k.name)}</strong>
            <span style="color:var(--text-dim);font-size:0.85rem;"> (${escapeHtml(k.token_hint || '')})</span>
            <div style="font-size:0.82rem;color:var(--text-dim);margin-top:2px;">Created ${new Date(k.created_at).toLocaleString()}</div>
            <button data-action="delete-passkey" data-id="${k.id}" aria-label="Revoke key ${escapeHtml(k.name)}">Revoke</button>
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
        alert('Profile updated successfully!');
    } catch (error) {
        alert(error.message);
    }
}

async function handleCreatePassKey(e) {
    e.preventDefault();
    const name = document.getElementById('new-passkey-name').value;
    if (!name) { alert('Name is required'); return; }
    try {
        const data = await API.createPassKey(name);
        if (data.token) alert(`Pass key created. Save this token now:\n\n${data.token}`);
        else if (data.message) alert(data.message);
        document.getElementById('create-passkey-form').reset();
        await loadSettings();
    } catch (error) {
        alert(error.message);
    }
}

async function deletePassKey(id) {
    if (!confirm('Revoke this pass key?')) return;
    try {
        await API.deletePassKey(id);
        await loadSettings();
    } catch (error) {
        alert(error.message);
    }
}

function renderCategories() {
    const container = document.getElementById('categories-list');
    if (categories.length === 0) {
        container.innerHTML = '<p class="muted">No categories yet. Create one to organize your standards!</p>';
        return;
    }

    container.innerHTML = categories.map(cat => `
        <div class="category-item" data-id="${cat.id}">
            <h3>
                <span class="category-color" style="background:${cat.color}"></span>
                ${escapeHtml(cat.name)}
            </h3>
            <button class="btn-remove" data-action="delete-category" data-id="${cat.id}"
                aria-label="Delete category ${escapeHtml(cat.name)}">Delete</button>
        </div>
    `).join('');
}

// ---- Action handlers ----
async function addStandard(standardId) {
    try {
        await API.addToMyStandards(standardId);
        await loadMyStandards();
        if (!document.getElementById('all-standards-view').classList.contains('hidden')) {
            await loadAllStandards(currentPage);
        }
    } catch (error) { alert(error.message); }
}

async function removeStandard(standardId) {
    if (!confirm('Remove this standard from your list?')) return;
    try {
        await API.removeFromMyStandards(standardId);
        await loadMyStandards();
        if (!document.getElementById('all-standards-view').classList.contains('hidden')) {
            await loadAllStandards(currentPage);
        }
    } catch (error) { alert(error.message); }
}

async function deleteStandard(standardId) {
    if (!confirm('Delete this standard permanently?')) return;
    try {
        await API.deleteStandard(standardId);
        await loadAllStandards(currentPage);
    } catch (error) { alert(error.message); }
}

async function handleAddStandard(e) {
    e.preventDefault();
    const title    = document.getElementById('new-standard-title').value;
    const composer = document.getElementById('new-standard-composer').value;
    const style    = document.getElementById('new-standard-style').value;
    const note     = document.getElementById('new-standard-note').value;

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
    const name  = document.getElementById('new-category-name').value;
    const color = document.getElementById('new-category-color').value;
    try {
        await API.createCategory(name, color);
        closeModal('add-category-modal');
        e.target.reset();
        await loadCategories();
    } catch (error) { alert(error.message); }
}

async function deleteCategory(categoryId) {
    if (!confirm('Delete this category? Standards will be uncategorized.')) return;
    try {
        await API.deleteCategory(categoryId);
        await loadCategories();
        await loadMyStandards();
    } catch (error) { alert(error.message); }
}

async function approveStandard(id) {
    if (!confirm('Approve this standard?')) return;
    try {
        await API.approveStandard(id);
        await loadPendingStandards();
        await loadAllStandards(currentPage);
        alert('Standard approved!');
    } catch (error) { alert(error.message); }
}

async function rejectStandard(id) {
    if (!confirm('Reject this standard?')) return;
    try {
        await API.rejectStandard(id);
        await loadPendingStandards();
        await loadAllStandards(currentPage);
        alert('Standard rejected.');
    } catch (error) { alert(error.message); }
}

// ---- Utilities ----
function debounce(func, wait) {
    let timeout;
    return function (...args) {
        clearTimeout(timeout);
        timeout = setTimeout(() => func(...args), wait);
    };
}

function checkOnlineStatus() {
    const indicator = document.getElementById('offline-indicator');
    function update() {
        if (navigator.onLine) {
            indicator.classList.add('hidden');
            indicator.setAttribute('aria-hidden', 'true');
        } else {
            indicator.classList.remove('hidden');
            indicator.setAttribute('aria-hidden', 'false');
        }
    }
    window.addEventListener('online', update);
    window.addEventListener('offline', update);
    update();
}

// ---- Boot ----
document.addEventListener('DOMContentLoaded', async () => {
    await init();
    // Expose internals for the in-browser test runner (tests.js)
    window.__testExports = { openModal, closeModal };
});

// Expose to global for delegated handlers
window.addStandard = addStandard;
window.removeStandard = removeStandard;
window.deleteStandard = deleteStandard;
window.approveStandard = approveStandard;
window.rejectStandard = rejectStandard;
window.deleteCategory = deleteCategory;
window.deletePassKey = deletePassKey;
window.loadAllStandards = loadAllStandards;
