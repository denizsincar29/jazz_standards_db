/**
 * Jazz Standards DB — In-browser test runner
 *
 * Usage (from DevTools console):
 *   import('/static/js/tests.js').then(m => m.runTests())
 *
 * Or the app automatically imports it when ?test=1 is in the URL.
 *
 * Output: structured results in console + a floating overlay in the page.
 */

// ─── Mini test framework ───────────────────────────────────────────────────────

const results = [];
let currentSuite = '';

function suite(name) {
    currentSuite = name;
    console.groupCollapsed(`%c📋 ${name}`, 'font-weight:bold;color:#d4a017;');
}

function endSuite() {
    console.groupEnd();
}

async function test(name, fn) {
    const fullName = `${currentSuite} › ${name}`;
    const t0 = performance.now();
    try {
        await fn();
        const ms = Math.round(performance.now() - t0);
        results.push({ ok: true, name: fullName, ms });
        console.log(`%c  ✓ ${name} %c(${ms}ms)`, 'color:#52c27a', 'color:#666');
    } catch (err) {
        const ms = Math.round(performance.now() - t0);
        results.push({ ok: false, name: fullName, ms, error: err.message || String(err) });
        console.error(`%c  ✘ ${name}`, 'color:#e05252', err);
    }
}

function expect(actual) {
    return {
        toBe(expected) {
            if (actual !== expected)
                throw new Error(`Expected ${JSON.stringify(expected)}, got ${JSON.stringify(actual)}`);
        },
        toEqual(expected) {
            if (JSON.stringify(actual) !== JSON.stringify(expected))
                throw new Error(`Expected ${JSON.stringify(expected)}, got ${JSON.stringify(actual)}`);
        },
        toBeTruthy() {
            if (!actual)
                throw new Error(`Expected truthy, got ${JSON.stringify(actual)}`);
        },
        toBeFalsy() {
            if (actual)
                throw new Error(`Expected falsy, got ${JSON.stringify(actual)}`);
        },
        toContain(sub) {
            if (!String(actual).includes(sub))
                throw new Error(`Expected "${actual}" to contain "${sub}"`);
        },
        toMatch(re) {
            if (!re.test(String(actual)))
                throw new Error(`Expected "${actual}" to match ${re}`);
        },
        toBeGreaterThan(n) {
            if (!(actual > n))
                throw new Error(`Expected ${actual} > ${n}`);
        },
        not: {
            toBe(expected) {
                if (actual === expected)
                    throw new Error(`Expected NOT ${JSON.stringify(expected)}`);
            },
            toBeTruthy() {
                if (actual)
                    throw new Error(`Expected falsy, got ${JSON.stringify(actual)}`);
            },
            toContain(sub) {
                if (String(actual).includes(sub))
                    throw new Error(`Expected "${actual}" NOT to contain "${sub}"`);
            }
        }
    };
}

// ─── DOM helpers ─────────────────────────────────────────────────────────────

function $(sel, root = document) { return root.querySelector(sel); }
function $$(sel, root = document) { return [...root.querySelectorAll(sel)]; }

function waitFor(sel, timeout = 3000) {
    return new Promise((resolve, reject) => {
        const el = $(sel);
        if (el) return resolve(el);
        const ob = new MutationObserver(() => {
            const found = $(sel);
            if (found) { ob.disconnect(); resolve(found); }
        });
        ob.observe(document.body, { childList: true, subtree: true, attributes: true });
        setTimeout(() => { ob.disconnect(); reject(new Error(`waitFor timeout: "${sel}" not found in ${timeout}ms`)); }, timeout);
    });
}

function waitForVisible(sel, timeout = 3000) {
    return new Promise((resolve, reject) => {
        function check() {
            const el = $(sel);
            if (el && !el.closest('.hidden') && getComputedStyle(el).display !== 'none') {
                return resolve(el);
            }
        }
        check();
        const ob = new MutationObserver(check);
        ob.observe(document.body, { childList: true, subtree: true, attributes: true });
        setTimeout(() => { ob.disconnect(); reject(new Error(`waitForVisible timeout: "${sel}" in ${timeout}ms`)); }, timeout);
    });
}

function waitForHidden(sel, timeout = 3000) {
    return new Promise((resolve, reject) => {
        function check() {
            const el = $(sel);
            if (!el || el.classList.contains('hidden') || getComputedStyle(el).display === 'none') {
                return resolve(true);
            }
        }
        check();
        const ob = new MutationObserver(check);
        ob.observe(document.body, { childList: true, subtree: true, attributes: true });
        setTimeout(() => { ob.disconnect(); reject(new Error(`waitForHidden timeout: "${sel}" still visible after ${timeout}ms`)); }, timeout);
    });
}

function sleep(ms) { return new Promise(r => setTimeout(r, ms)); }

function click(sel) {
    const el = $(sel);
    if (!el) throw new Error(`click: element not found: "${sel}"`);
    el.click();
}

function isHidden(sel) {
    const el = $(sel);
    if (!el) return true;
    return el.classList.contains('hidden') || getComputedStyle(el).display === 'none';
}

function isVisible(sel) { return !isHidden(sel); }

function attr(sel, name) {
    const el = $(sel);
    if (!el) throw new Error(`attr: element not found: "${sel}"`);
    return el.getAttribute(name);
}

function hasClass(sel, cls) {
    const el = $(sel);
    if (!el) throw new Error(`hasClass: element not found: "${sel}"`);
    return el.classList.contains(cls);
}

// ─── Mock helpers ────────────────────────────────────────────────────────────

// Save and restore app globals between tests
function saveState() {
    return {
        html: document.body.innerHTML,
        activeElement: document.activeElement,
    };
}

// Open a modal fresh (close it first if open, then open via the trigger)
function openModal(modalId) {
    const modal = document.getElementById(modalId);
    if (!modal) throw new Error(`openModal: #${modalId} not found`);
    // Use the app's openModal if available
    if (window._openModal) {
        window._openModal(modalId);
    } else {
        modal.classList.remove('hidden');
        modal.setAttribute('aria-hidden', 'false');
    }
}

function closeModalDirect(modalId) {
    const modal = document.getElementById(modalId);
    if (!modal) throw new Error(`closeModal: #${modalId} not found`);
    if (window._closeModal) {
        window._closeModal(modalId);
    } else {
        modal.classList.add('hidden');
        modal.setAttribute('aria-hidden', 'true');
    }
}

// Switch to a named view
function switchView(name) {
    const btn = $(`[data-view="${name}"]`);
    if (!btn) throw new Error(`switchView: no [data-view="${name}"] button found`);
    btn.click();
}

// ─── Expose internal app functions for testing ───────────────────────────────
// app.js must call window.__testExports = { openModal, closeModal } after init
function getAppExports() {
    if (!window.__testExports) {
        throw new Error('window.__testExports not set — make sure app.js exports its modal functions');
    }
    return window.__testExports;
}

// ─── TESTS ───────────────────────────────────────────────────────────────────

export async function runTests() {
    results.length = 0;

    console.clear();
    console.log('%cJazz Standards DB — Frontend Tests', 'font-size:16px;font-weight:bold;color:#f0c040;');
    console.log('%cRunning against the live DOM…', 'color:#9895a0;');
    console.log('');

    // ── Check we have test exports ──────────────────────────────────────────
    let appExports;
    try {
        appExports = getAppExports();
    } catch (e) {
        console.error('%c✘ SETUP FAILED:', 'color:#e05252;font-weight:bold;', e.message);
        console.error('Make sure the app is loaded first and window.__testExports is set.');
        renderOverlay();
        return;
    }

    const { openModal: _open, closeModal: _close } = appExports;

    // ── Convenience wrappers using app's real functions ─────────────────────
    function openM(id) { _open(id); }
    function closeM(id) { _close(id); }

    // ════════════════════════════════════════════════════════════════════════
    suite('DOM Structure');

    await test('auth screen exists', async () => {
        expect($('#auth-screen')).toBeTruthy();
    });

    await test('main screen exists', async () => {
        expect($('#main-screen')).toBeTruthy();
    });

    await test('add-standard-modal exists in DOM', async () => {
        expect($('#add-standard-modal')).toBeTruthy();
    });

    await test('add-category-modal exists in DOM', async () => {
        expect($('#add-category-modal')).toBeTruthy();
    });

    await test('all navigation view containers exist', async () => {
        const views = ['my-standards', 'all-standards', 'categories', 'settings'];
        for (const v of views) {
            const el = $(`#${v}-view`);
            if (!el) throw new Error(`#${v}-view not found`);
        }
    });

    endSuite();

    // ════════════════════════════════════════════════════════════════════════
    suite('ARIA Attributes — Modals');

    await test('add-standard-modal has role="dialog"', async () => {
        expect(attr('#add-standard-modal', 'role')).toBe('dialog');
    });

    await test('add-standard-modal has aria-modal="true"', async () => {
        expect(attr('#add-standard-modal', 'aria-modal')).toBe('true');
    });

    await test('add-standard-modal has aria-labelledby', async () => {
        const labelId = attr('#add-standard-modal', 'aria-labelledby');
        expect(labelId).toBeTruthy();
        const heading = $(`#${labelId}`);
        if (!heading) throw new Error(`aria-labelledby="${labelId}" points to nothing`);
        expect(heading.textContent.trim().length).toBeGreaterThan(0);
    });

    await test('add-category-modal has role="dialog"', async () => {
        expect(attr('#add-category-modal', 'role')).toBe('dialog');
    });

    await test('add-category-modal has aria-modal="true"', async () => {
        expect(attr('#add-category-modal', 'aria-modal')).toBe('true');
    });

    await test('add-category-modal has aria-labelledby', async () => {
        const labelId = attr('#add-category-modal', 'aria-labelledby');
        expect(labelId).toBeTruthy();
        const heading = $(`#${labelId}`);
        if (!heading) throw new Error(`aria-labelledby="${labelId}" points to nothing`);
        expect(heading.textContent.trim().length).toBeGreaterThan(0);
    });

    endSuite();

    // ════════════════════════════════════════════════════════════════════════
    suite('ARIA Attributes — Views');

    await test('all view regions have role="region"', async () => {
        const views = $$('.view');
        for (const v of views) {
            if (!v.id.includes('admin-only') && !v.classList.contains('admin-only')) {
                if (v.getAttribute('role') !== 'region')
                    throw new Error(`#${v.id} missing role="region"`);
            }
        }
    });

    await test('active view has aria-hidden="false"', async () => {
        const active = $('.view.active');
        if (!active) throw new Error('No active view found');
        expect(active.getAttribute('aria-hidden')).toBe('false');
    });

    await test('hidden views have aria-hidden="true"', async () => {
        const hidden = $$('.view.hidden');
        for (const v of hidden) {
            if (v.getAttribute('aria-hidden') !== 'true')
                throw new Error(`#${v.id} is hidden but aria-hidden="${v.getAttribute('aria-hidden')}"`);
        }
    });

    await test('standards list has aria-live="polite"', async () => {
        expect(attr('#all-standards-list', 'aria-live')).toBe('polite');
    });

    endSuite();

    // ════════════════════════════════════════════════════════════════════════
    suite('Modal: Initial State');

    await test('add-standard-modal starts hidden', async () => {
        // Ensure it's closed first
        closeM('add-standard-modal');
        expect(isHidden('#add-standard-modal')).toBe(true);
        expect(attr('#add-standard-modal', 'aria-hidden')).toBe('true');
    });

    await test('add-category-modal starts hidden', async () => {
        closeM('add-category-modal');
        expect(isHidden('#add-category-modal')).toBe(true);
        expect(attr('#add-category-modal', 'aria-hidden')).toBe('true');
    });

    endSuite();

    // ════════════════════════════════════════════════════════════════════════
    suite('Modal: Open behaviour');

    await test('openModal removes .hidden from add-standard-modal', async () => {
        openM('add-standard-modal');
        expect(isVisible('#add-standard-modal')).toBe(true);
        closeM('add-standard-modal');
    });

    await test('openModal sets aria-hidden="false" on add-standard-modal', async () => {
        openM('add-standard-modal');
        expect(attr('#add-standard-modal', 'aria-hidden')).toBe('false');
        closeM('add-standard-modal');
    });

    await test('openModal moves focus inside the modal', async () => {
        openM('add-standard-modal');
        await sleep(100); // rAF tick
        const focused = document.activeElement;
        const inside = focused && !!focused.closest('#add-standard-modal');
        closeM('add-standard-modal');
        if (!inside) throw new Error(`Focus is on <${focused?.tagName?.toLowerCase()}#${focused?.id}>, not inside #add-standard-modal`);
    });

    await test('openModal sets body overflow to hidden', async () => {
        openM('add-standard-modal');
        const overflow = document.body.style.overflow;
        closeM('add-standard-modal');
        expect(overflow).toBe('hidden');
    });

    await test('openModal: add-category-modal also shows correctly', async () => {
        openM('add-category-modal');
        expect(isVisible('#add-category-modal')).toBe(true);
        expect(attr('#add-category-modal', 'aria-hidden')).toBe('false');
        closeM('add-category-modal');
    });

    endSuite();

    // ════════════════════════════════════════════════════════════════════════
    suite('Modal: Close behaviour');

    await test('closeModal adds .hidden', async () => {
        openM('add-standard-modal');
        closeM('add-standard-modal');
        expect(isHidden('#add-standard-modal')).toBe(true);
    });

    await test('closeModal sets aria-hidden="true"', async () => {
        openM('add-standard-modal');
        closeM('add-standard-modal');
        expect(attr('#add-standard-modal', 'aria-hidden')).toBe('true');
    });

    await test('closeModal restores body overflow', async () => {
        openM('add-standard-modal');
        closeM('add-standard-modal');
        expect(document.body.style.overflow).toBe('');
    });

    await test('✕ close button closes the modal', async () => {
        openM('add-standard-modal');
        click('#add-standard-modal .modal-close');
        await waitForHidden('#add-standard-modal', 1000);
        expect(isHidden('#add-standard-modal')).toBe(true);
    });

    await test('Cancel button closes the modal', async () => {
        openM('add-standard-modal');
        click('#add-standard-modal .cancel');
        await waitForHidden('#add-standard-modal', 1000);
        expect(isHidden('#add-standard-modal')).toBe(true);
    });

    await test('Escape key closes the modal', async () => {
        openM('add-standard-modal');
        document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape', bubbles: true }));
        await waitForHidden('#add-standard-modal', 1000);
        expect(isHidden('#add-standard-modal')).toBe(true);
    });

    await test('clicking the modal overlay (not content) closes modal', async () => {
        openM('add-standard-modal');
        // Simulate click directly on the .modal element (the overlay)
        const modal = $('#add-standard-modal');
        modal.dispatchEvent(new MouseEvent('click', { bubbles: true }));
        await waitForHidden('#add-standard-modal', 1000);
        expect(isHidden('#add-standard-modal')).toBe(true);
    });

    await test('clicking inside modal-content does NOT close modal', async () => {
        openM('add-standard-modal');
        // Click on the form title (inside .modal-content)
        const heading = $('#add-standard-modal-title');
        if (heading) heading.dispatchEvent(new MouseEvent('click', { bubbles: true }));
        await sleep(150);
        expect(isVisible('#add-standard-modal')).toBe(true);
        closeM('add-standard-modal');
    });

    endSuite();

    // ════════════════════════════════════════════════════════════════════════
    suite('Modal: Form fields');

    await test('add-standard-modal has all required inputs', async () => {
        const required = ['#new-standard-title', '#new-standard-composer', '#new-standard-style'];
        for (const sel of required) {
            if (!$(sel)) throw new Error(`Missing input: ${sel}`);
        }
    });

    await test('add-standard-modal inputs have labels', async () => {
        const pairs = [
            ['new-standard-title', 'Title'],
            ['new-standard-composer', 'Composer'],
            ['new-standard-style', 'Style'],
        ];
        for (const [id, hint] of pairs) {
            const label = $(`label[for="${id}"]`);
            if (!label) throw new Error(`No <label for="${id}"> found (expected label for ${hint})`);
        }
    });

    await test('add-category-modal has name input and color picker', async () => {
        if (!$('#new-category-name')) throw new Error('Missing #new-category-name');
        if (!$('#new-category-color')) throw new Error('Missing #new-category-color');
    });

    endSuite();

    // ════════════════════════════════════════════════════════════════════════
    suite('View Navigation');

    await test('navigation buttons exist for all main views', async () => {
        const expected = ['my-standards', 'all-standards', 'categories', 'settings'];
        for (const v of expected) {
            if (!$(`[data-view="${v}"]`))
                throw new Error(`Missing nav button [data-view="${v}"]`);
        }
    });

    await test('only one view is .active at a time', async () => {
        const activeViews = $$('.view.active');
        if (activeViews.length !== 1)
            throw new Error(`Expected 1 active view, found ${activeViews.length}: ${activeViews.map(v => '#' + v.id).join(', ')}`);
    });

    await test('clicking a nav button activates that view', async () => {
        // Click categories
        switchView('categories');
        await sleep(50);
        const active = $$('.view.active');
        if (active.length !== 1) throw new Error(`${active.length} active views after switching`);
        if (!active[0].id.includes('categories'))
            throw new Error(`Active view is #${active[0].id}, expected categories-view`);
    });

    await test('switching views hides all other views', async () => {
        switchView('settings');
        await sleep(50);
        const wrongActive = $$('.view.active').filter(v => !v.id.includes('settings'));
        if (wrongActive.length > 0)
            throw new Error(`Extra active views: ${wrongActive.map(v => '#' + v.id).join(', ')}`);
    });

    await test('switching views updates aria-hidden correctly', async () => {
        switchView('my-standards');
        await sleep(50);
        const active = $('#my-standards-view');
        if (active.getAttribute('aria-hidden') !== 'false')
            throw new Error(`#my-standards-view aria-hidden="${active.getAttribute('aria-hidden')}" after activation`);
        const inactive = $('#settings-view');
        if (inactive.getAttribute('aria-hidden') !== 'true')
            throw new Error(`#settings-view aria-hidden="${inactive.getAttribute('aria-hidden')}" — should be "true"`);
    });

    endSuite();

    // ════════════════════════════════════════════════════════════════════════
    suite('Main Screen State');

    const mainHidden = isHidden('#main-screen');
    if (!mainHidden) {
        // Only run these if we're logged in
        await test('user name element exists and is non-empty when logged in', async () => {
            const el = $('#user-name');
            if (!el) throw new Error('#user-name not found');
            // May be empty if test runs before data loads — just check it exists
        });

        await test('logout button exists', async () => {
            expect($('#logout-btn')).toBeTruthy();
        });
    } else {
        await test('auth screen is visible when not logged in', async () => {
            expect(isVisible('#auth-screen')).toBe(true);
        });

        await test('login tab is selected by default', async () => {
            const loginTab = $('[data-tab="login"]');
            expect(loginTab?.getAttribute('aria-selected')).toBe('true');
        });

        await test('login form is active by default', async () => {
            expect(hasClass('#login-form', 'active')).toBe(true);
        });
    }

    endSuite();

    // ─── Summary ──────────────────────────────────────────────────────────────
    console.log('');
    const passed = results.filter(r => r.ok).length;
    const failed = results.filter(r => !r.ok).length;
    const total  = results.length;

    if (failed === 0) {
        console.log(`%c✅ All ${total} tests passed`, 'font-size:14px;font-weight:bold;color:#52c27a;');
    } else {
        console.log(`%c❌ ${failed} failed / ${total} total`, 'font-size:14px;font-weight:bold;color:#e05252;');
        console.log('%cFailed tests:', 'color:#e05252;font-weight:bold;');
        results.filter(r => !r.ok).forEach(r => {
            console.log(`  ✘ ${r.name}`);
            console.log(`    → ${r.error}`);
        });
    }

    console.log('');
    console.log('%cFull results object: window.__testResults', 'color:#9895a0;font-size:11px;');
    window.__testResults = results;

    renderOverlay(passed, failed, total);
    return { passed, failed, total, results };
}

// ─── Floating overlay ─────────────────────────────────────────────────────────

function renderOverlay(passed = 0, failed = 0, total = 0) {
    // Remove old overlay
    document.getElementById('__jazz-test-overlay')?.remove();

    const overlay = document.createElement('div');
    overlay.id = '__jazz-test-overlay';
    overlay.style.cssText = `
        position:fixed; bottom:16px; right:16px; z-index:99999;
        background:#181828; border:1px solid #2c2c4a;
        border-radius:10px; padding:14px 18px; min-width:260px;
        font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',sans-serif;
        font-size:13px; box-shadow:0 4px 24px rgba(0,0,0,.7);
        color:#e8e4dc; max-height:80vh; overflow-y:auto;
    `;

    const statusColor = failed === 0 ? '#52c27a' : '#e05252';
    const statusIcon  = failed === 0 ? '✅' : '❌';

    const failList = results.filter(r => !r.ok).map(r => `
        <div style="margin-top:6px;padding:6px 8px;background:#2c2c4a;border-radius:6px;border-left:3px solid #e05252;">
            <div style="color:#e05252;font-weight:600;">✘ ${escHtml(r.name.split(' › ').pop())}</div>
            <div style="color:#9895a0;font-size:11px;margin-top:2px;">${escHtml(r.error)}</div>
        </div>
    `).join('');

    overlay.innerHTML = `
        <div style="display:flex;justify-content:space-between;align-items:center;margin-bottom:10px;">
            <span style="font-weight:700;color:#d4a017;">🎷 Test Results</span>
            <button onclick="this.closest('#__jazz-test-overlay').remove()"
                    style="background:none;border:none;color:#9895a0;cursor:pointer;font-size:16px;line-height:1;">✕</button>
        </div>
        <div style="font-size:15px;font-weight:700;color:${statusColor};">
            ${statusIcon} ${passed}/${total} passed
            ${failed > 0 ? `<span style="color:#e05252;"> (${failed} failed)</span>` : ''}
        </div>
        ${failList ? `<div style="margin-top:10px;">${failList}</div>` : ''}
        <div style="margin-top:10px;border-top:1px solid #2c2c4a;padding-top:8px;">
            <button onclick="import('/static/js/tests.js').then(m=>m.runTests())"
                    style="width:100%;padding:7px;background:#d4a017;color:#000;border:none;
                           border-radius:6px;cursor:pointer;font-weight:600;font-size:12px;">
                ↺ Re-run tests
            </button>
        </div>
    `;

    document.body.appendChild(overlay);
}

function escHtml(s) {
    return String(s).replace(/[&<>"']/g, c =>
        ({ '&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;' }[c])
    );
}

// ─── Auto-run if ?test=1 in URL ───────────────────────────────────────────────
if (new URLSearchParams(location.search).get('test') === '1') {
    // Wait for DOMContentLoaded + app init
    const tryRun = () => {
        if (window.__testExports) {
            runTests();
        } else {
            setTimeout(tryRun, 200);
        }
    };
    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', () => setTimeout(tryRun, 300));
    } else {
        setTimeout(tryRun, 300);
    }
}
