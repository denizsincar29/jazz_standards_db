import API from './api.js';
import Dom from './dom.js';

// App state
let currentUser = null;
let allStandards = [];
let myStandards = [];
let categories = [];
let currentPage = 1;

function debugPrint(...args) {
    if (window.DEBUG) {
        console.log('[DEBUG]', ...args);
    }
}

// Utility to escape HTML when rendering strings into templates
function escapeHtml(str) {
    if (!str && str !== 0) return '';
    return String(str).replace(/[&<>"'`]/g, function (s) {
        return ({
            '&': '&amp;',
            '<': '&lt;',
            '>': '&gt;',
            '"': '&quot;',
            "'": '&#39;',
            '`': '&#96;'
        })[s];
    });
}

// Initialize app
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

// Setup event listeners
function setupEventListeners() {
    // Auth tabs
    document.querySelectorAll('.tab').forEach(tab => {
        tab.addEventListener('click', (e) => {
            document.querySelectorAll('.tab').forEach(t => t.classList.remove('active'));
            document.querySelectorAll('.auth-form').forEach(f => f.classList.remove('active'));
            e.target.classList.add('active');
            document.getElementById(`${e.target.dataset.tab}-form`).classList.add('active');
        });
    });
    
    // Auth forms
    document.getElementById('login-form').addEventListener('submit', handleLogin);
    document.getElementById('register-form').addEventListener('submit', handleRegister);
    
    // Logout
    document.getElementById('logout-btn').addEventListener('click', handleLogout);
    
    // Navigation
    document.querySelectorAll('.nav-btn').forEach(btn => {
        btn.addEventListener('click', (e) => {
            const view = e.target.dataset.view;
            switchView(view);
        });
    });

    // Settings forms (may not exist on auth screen)
    const profileForm = document.getElementById('update-profile-form');
    if (profileForm) profileForm.addEventListener('submit', handleUpdateProfile);
    const passkeyForm = document.getElementById('create-passkey-form');
    if (passkeyForm) passkeyForm.addEventListener('submit', handleCreatePassKey);
    
    // Search and filter
    document.getElementById('search-input').addEventListener('input', debounce(loadAllStandards, 300));
    document.getElementById('style-filter').addEventListener('change', loadAllStandards);
    
    // Add standard/category buttons
    document.getElementById('add-standard-btn').addEventListener('click', () => {
        showElement('add-standard-modal');
    });
    document.getElementById('add-category-btn').addEventListener('click', () => {
        showElement('add-category-modal');
    });
    
    // Modal forms
    document.getElementById('add-standard-form').addEventListener('submit', handleAddStandard);
    document.getElementById('add-category-form').addEventListener('submit', handleAddCategory);
    
    // Modal cancel buttons
    document.querySelectorAll('.modal .cancel').forEach(btn => {
        btn.addEventListener('click', (e) => {
            const modal = e.target.closest('.modal');
            if (modal) {
                modal.classList.add('hidden');
                modal.setAttribute('aria-hidden', 'true');
            }
        });
    });

    // Attach delegated handlers for dynamic content
    Dom.attachDelegation();
}

// Auth handlers
async function handleLogin(e) {
    e.preventDefault();
    const username = document.getElementById('login-username').value;
    const password = document.getElementById('login-password').value;
    const errorEl = document.getElementById('login-error');
    
    try {
        debugPrint('handleLogin called for user:', username);
        currentUser = await API.login(username, password);
        debugPrint('Login successful, user:', currentUser);
        showMainScreen();
        await loadMyStandards();
    } catch (error) {
        debugPrint('Login failed with error:', error);
        errorEl.textContent = error.message;
    }
}

async function handleRegister(e) {
    e.preventDefault();
    const name = document.getElementById('register-name').value;
    const username = document.getElementById('register-username').value;
    const password = document.getElementById('register-password').value;
    const errorEl = document.getElementById('register-error');
    
    try {
        currentUser = await API.register(username, name, password);
        showMainScreen();
        await loadMyStandards();
    } catch (error) {
        errorEl.textContent = error.message;
    }
}

async function handleLogout() {
    try {
        await API.logout();
    } catch (error) {
        console.error('Logout error:', error);
    }
    currentUser = null;
    showAuthScreen();
}

// Helper functions for accessibility-aware hiding/showing
function hideElement(elementId) {
    const element = document.getElementById(elementId);
    if (!element) return;
    element.classList.add('hidden');
    element.setAttribute('aria-hidden', 'true');
    try { element.inert = true; } catch (e) { /* inert may be unsupported */ }
}

function showElement(elementId) {
    const element = document.getElementById(elementId);
    if (!element) return;
    element.classList.remove('hidden');
    element.setAttribute('aria-hidden', 'false');
    try { element.inert = false; } catch (e) { /* inert may be unsupported */ }
}

// Screen switching
function showAuthScreen() {
    showElement('auth-screen');
    hideElement('main-screen');
}

function showMainScreen() {
    hideElement('auth-screen');
    showElement('main-screen');
    document.getElementById('user-name').textContent = currentUser.name;
    
    // Show/hide admin controls
    const adminElements = document.querySelectorAll('.admin-only');
    adminElements.forEach(el => {
        if (currentUser.is_admin) {
            el.classList.remove('hidden');
            el.setAttribute('aria-hidden', 'false');
        } else {
            el.classList.add('hidden');
            el.setAttribute('aria-hidden', 'true');
        }
    });
    
    // Load pending standards if admin
    if (currentUser.is_admin) {
        loadPendingStandards();
    }
}

function switchView(viewName) {
    document.querySelectorAll('.nav-btn').forEach(btn => btn.classList.remove('active'));
    document.querySelectorAll('.view').forEach(view => {
        view.classList.add('hidden');
        view.setAttribute('aria-hidden', 'true');
    });
    
    document.querySelector(`[data-view="${viewName}"]`).classList.add('active');
    const activeView = document.getElementById(`${viewName}-view`);
    activeView.classList.remove('hidden');
    activeView.setAttribute('aria-hidden', 'false');
    
    // Load data for the view
    if (viewName === 'all-standards') {
        loadAllStandards();
    } else if (viewName === 'categories') {
        loadCategories();
    } else if (viewName === 'settings') {
        loadSettings();
    } else if (viewName === 'pending') {
        loadPendingStandards(1);
    }
}

// Load data
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

// Render functions
function renderMyStandards(grouped) {
    const container = document.getElementById('standards-by-category');
    container.innerHTML = '';
    
    if (Object.keys(grouped).length === 0) {
        container.innerHTML = '<p style="color: white;">No standards yet. Add some from the "All Standards" tab!</p>';
        return;
    }
    
    for (const [categoryName, standards] of Object.entries(grouped)) {
        const categoryDiv = document.createElement('div');
        categoryDiv.className = 'category-group';
        
        const category = standards[0].category;
        const colorHtml = category ? `<span class="category-color" style="background: ${category.color}"></span>` : '';
        
        categoryDiv.innerHTML = `
            <h3>${colorHtml}${categoryName}</h3>
            ${standards.map(us => `
                <div class="standard-item" data-id="${us.jazz_standard_id}">
                    <div class="standard-info">
                        <h4>${escapeHtml(us.jazz_standard.title)}</h4>
                        <p>${escapeHtml(us.jazz_standard.composer)} - ${escapeHtml(us.jazz_standard.style)}</p>
                        ${us.notes ? `<p><em>${escapeHtml(us.notes)}</em></p>` : ''}
                    </div>
                    <div class="standard-actions">
                        <button class="btn-remove" data-action="remove" data-id="${us.jazz_standard_id}">Remove</button>
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
        container.innerHTML = '<p style="color: white;">No standards found.</p>';
        return;
    }
    
    container.innerHTML = standards.map(standard => {
        const isKnown = myStandards.some(us => us.jazz_standard_id === standard.id);
        const isPending = standard.status === 'pending';
        const isRejected = standard.status === 'rejected';
        
        let statusBadge = '';
        if (isPending) {
            statusBadge = '<span style="color: #ffc107; font-weight: bold;">⏳ Pending Approval</span>';
        } else if (isRejected) {
            statusBadge = '<span style="color: #dc3545; font-weight: bold;">❌ Rejected</span>';
        }
        
        const addBtn = `<button class="btn-know" data-action="add" data-id="${standard.id}" aria-label="Add ${escapeHtml(standard.title)}">I Know This</button>`;
        const removeBtn = `<button class="btn-remove" data-action="remove" data-id="${standard.id}" aria-label="Remove ${escapeHtml(standard.title)}">Remove</button>`;
        const deleteBtn = `<button class="btn-remove" data-action="delete" data-id="${standard.id}" aria-label="Delete ${escapeHtml(standard.title)}">Delete</button>`;
        const actionBtn = isKnown ? removeBtn : addBtn;

        // Ensure the title has an id for aria-labelledby and the item is focusable
        const titleId = `standard-title-${standard.id}`;

        return `
            <div class="standard-item" tabindex="0" role="article" aria-labelledby="${titleId}" data-id="${standard.id}">
                <div class="standard-info">
                    <h4 id="${titleId}">${escapeHtml(standard.title)} ${statusBadge}</h4>
                    <p>${escapeHtml(standard.composer)} - ${escapeHtml(standard.style)}</p>
                    ${standard.additional_note ? `<p><em>${escapeHtml(standard.additional_note)}</em></p>` : ''}
                    ${standard.creator ? `<p style="font-size: 0.85em; color: #aaa;">Submitted by: ${escapeHtml(standard.creator.name)}</p>` : ''}
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
    
    if (totalPages <= 1) {
        container.innerHTML = '';
        return;
    }
    
    container.innerHTML = `
        <button ${data.page === 1 ? 'disabled' : ''} data-action="paginate" data-page="${data.page - 1}">Previous</button>
        <span>Page ${data.page} of ${totalPages}</span>
        <button ${data.page === totalPages ? 'disabled' : ''} data-action="paginate" data-page="${data.page + 1}">Next</button>
    `;
}

function renderPendingStandards(data) {
    const container = document.getElementById('pending-standards-list');
    if (!container) return;
    
    const standards = data.standards || [];
    
    if (standards.length === 0) {
        container.innerHTML = '<p style="color: white;">No pending standards.</p>';
        return;
    }
    
    container.innerHTML = standards.map(standard => `
        <div class="standard-item pending-item" data-id="${standard.id}">
            <div class="standard-info">
                <h4>${escapeHtml(standard.title)}</h4>
                <p>${escapeHtml(standard.composer)} - ${escapeHtml(standard.style)}</p>
                ${standard.additional_note ? `<p><em>${escapeHtml(standard.additional_note)}</em></p>` : ''}
                <p style="font-size: 0.85em; color: #aaa;">
                    Submitted by: ${standard.creator ? escapeHtml(standard.creator.name) : 'Unknown'} 
                    on ${new Date(standard.created_at).toLocaleDateString()}
                </p>
            </div>
            <div class="standard-actions">
                <button class="btn-approve" data-action="approve" data-id="${standard.id}">✓ Approve</button>
                <button class="btn-reject" data-action="reject" data-id="${standard.id}">✗ Reject</button>
                <button class="btn-remove" data-action="delete" data-id="${standard.id}">Delete</button>
            </div>
        </div>
    `).join('');
}

// Settings / Passkeys
async function loadSettings() {
    try {
        const user = await API.getMe();
        // prefill profile form
        const nameEl = document.getElementById('profile-name');
        const usernameEl = document.getElementById('profile-username');
        const publicEl = document.getElementById('profile-public');
        if (nameEl) nameEl.value = user.name || '';
        if (usernameEl) usernameEl.value = user.username || '';
        if (publicEl) publicEl.checked = !!user.public_profile;

        // load passkeys
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
        container.innerHTML = '<p style="color: white;">No pass keys yet.</p>';
        return;
    }
    container.innerHTML = keys.map(k => `
        <div class="passkey-item" data-id="${k.id}">
            <strong>${escapeHtml(k.name)}</strong> <span style="color:#aaa">(${escapeHtml(k.token_hint || '')})</span>
            <div style="font-size:0.85em;color:#aaa">Created: ${new Date(k.created_at).toLocaleString()}</div>
            <div><button data-action="delete-passkey" data-id="${k.id}">Revoke</button></div>
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
        alert('Profile updated');
    } catch (error) {
        alert(error.message);
    }
}

async function handleCreatePassKey(e) {
    e.preventDefault();
    const name = document.getElementById('new-passkey-name').value;
    if (!name) return alert('Name is required');
    try {
        const data = await API.createPassKey(name);
        // Show token once
        if (data.token) {
            alert(`Pass key created. Save this token now:\n\n${data.token}`);
        } else if (data.message) {
            alert(data.message);
        }
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
        container.innerHTML = '<p style="color: white;">No categories yet. Create one!</p>';
        return;
    }
    
    container.innerHTML = categories.map(cat => `
        <div class="category-item" data-id="${cat.id}">
            <h3>
                <span class="category-color" style="background: ${cat.color}"></span>
                ${escapeHtml(cat.name)}
            </h3>
            <button class="btn-remove" data-action="delete-category" data-id="${cat.id}">Delete</button>
        </div>
    `).join('');
}

// Action handlers
async function addStandard(standardId) {
    try {
        await API.addToMyStandards(standardId);
        await loadMyStandards();
        if (document.getElementById('all-standards-view').classList.contains('active')) {
            await loadAllStandards(currentPage);
        }
    } catch (error) {
        alert(error.message);
    }
}

async function removeStandard(standardId) {
    if (!confirm('Remove this standard from your list?')) return;
    
    try {
        await API.removeFromMyStandards(standardId);
        await loadMyStandards();
        if (document.getElementById('all-standards-view').classList.contains('active')) {
            await loadAllStandards(currentPage);
        }
    } catch (error) {
        alert(error.message);
    }
}

async function deleteStandard(standardId) {
    if (!confirm('Delete this standard permanently?')) return;
    
    try {
        await API.deleteStandard(standardId);
        await loadAllStandards(currentPage);
    } catch (error) {
        alert(error.message);
    }
}

async function handleAddStandard(e) {
    e.preventDefault();
    
    const title = document.getElementById('new-standard-title').value;
    const composer = document.getElementById('new-standard-composer').value;
    const style = document.getElementById('new-standard-style').value;
    const note = document.getElementById('new-standard-note').value;
    
    try {
        const response = await API.createStandard(title, composer, style, note);
        hideElement('add-standard-modal');
        e.target.reset();
        
        // Show success message
        if (response.message) {
            alert(response.message);
        }
        
        await loadAllStandards(currentPage);
        
        // If admin, also reload pending standards
        if (currentUser.is_admin) {
            loadPendingStandards();
        }
    } catch (error) {
        alert(error.message);
    }
}

async function handleAddCategory(e) {
    e.preventDefault();
    
    const name = document.getElementById('new-category-name').value;
    const color = document.getElementById('new-category-color').value;
    
    try {
        await API.createCategory(name, color);
        hideElement('add-category-modal');
        e.target.reset();
        await loadCategories();
    } catch (error) {
        alert(error.message);
    }
}

async function deleteCategory(categoryId) {
    if (!confirm('Delete this category? Standards will be uncategorized.')) return;
    
    try {
        await API.deleteCategory(categoryId);
        await loadCategories();
        await loadMyStandards();
    } catch (error) {
        alert(error.message);
    }
}

// Utility functions
function debounce(func, wait) {
    let timeout;
    return function executedFunction(...args) {
        const later = () => {
            clearTimeout(timeout);
            func(...args);
        };
        clearTimeout(timeout);
        timeout = setTimeout(later, wait);
    };
}

function checkOnlineStatus() {
    const indicator = document.getElementById('offline-indicator');
    
    function updateStatus() {
        if (navigator.onLine) {
            indicator.classList.add('hidden');
            indicator.setAttribute('aria-hidden', 'true');
        } else {
            indicator.classList.remove('hidden');
            indicator.setAttribute('aria-hidden', 'false');
        }
    }
    
    window.addEventListener('online', updateStatus);
    window.addEventListener('offline', updateStatus);
    updateStatus();
}

// Admin functions for approving/rejecting standards
async function approveStandard(id) {
    if (!confirm('Approve this standard?')) return;
    
    try {
        await API.approveStandard(id);
        await loadPendingStandards();
        await loadAllStandards(currentPage);
        alert('Standard approved successfully!');
    } catch (error) {
        alert(error.message);
    }
}

async function rejectStandard(id) {
    if (!confirm('Reject this standard? It will be marked as rejected.')) return;
    
    try {
        await API.rejectStandard(id);
        await loadPendingStandards();
        await loadAllStandards(currentPage);
        alert('Standard rejected.');
    } catch (error) {
        alert(error.message);
    }
}

// Initialize on load
document.addEventListener('DOMContentLoaded', init);

// Export selected functions to window for inline handlers and external scripts
window.addStandard = addStandard;
window.removeStandard = removeStandard;
window.deleteStandard = deleteStandard;
window.approveStandard = approveStandard;
window.rejectStandard = rejectStandard;
window.deleteCategory = deleteCategory;
window.deletePassKey = deletePassKey;
window.loadAllStandards = loadAllStandards;
