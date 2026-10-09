let performanceChart = null;
let configLoaded = false;
let currentLogLimit = 40;
const MAX_LOG_LIMIT = 300;

function initializeChart(canvasId) {
    const ctx = document.getElementById(canvasId).getContext('2d');
    performanceChart = new Chart(ctx, {
        type: 'line',
        data: {
            labels: [],
            datasets: [{
                label: 'Run Duration (Seconds)',
                data: [],
                borderColor: '#6366f1',
                backgroundColor: 'rgba(99, 102, 241, 0.02)',
                borderWidth: 2,
                pointBackgroundColor: [],
                pointBorderColor: [],
                pointRadius: 4,
                tension: 0.15,
                fill: true
            }]
        },
        options: {
            responsive: true,
            maintainAspectRatio: false,
            plugins: { legend: { display: false } },
            scales: {
                y: { grid: { color: '#1e293b' }, ticks: { color: '#475569', font: { family: 'monospace' } } },
                x: { grid: { display: false }, ticks: { color: '#475569', font: { family: 'monospace' } } }
            }
        }
    });
}

// HTML escape helper to prevent DOM-based XSS vulnerabilities
function escapeHtml(str) {
    if (str === null || str === undefined) return '';
    return String(str)
        .replace(/&/g, '&amp;')
        .replace(/</g, '&lt;')
        .replace(/>/g, '&gt;')
        .replace(/"/g, '&quot;')
        .replace(/'/g, '&#39;');
}

function formatLogLine(line) {
    if (typeof line !== 'string') return '';
    line = line.trim();
    if (!line) return '';

    try {
        // Parse the entire line directly. Rclone outputs one valid JSON object per line.
        const parsedLog = JSON.parse(line);

        let lvlColor = "text-slate-400";
        const level = (parsedLog.level || 'info').toLowerCase();
        if (level === 'error') lvlColor = "text-rose-400 font-bold";
        else if (level === 'warning') lvlColor = "text-amber-400 font-medium";
        else if (level === 'notice') lvlColor = "text-sky-400";
        else if (level === 'info') lvlColor = "text-emerald-500";

        let timeStr = 'N/A';
        if (parsedLog.time) {
            const dateObj = new Date(parsedLog.time);
            if (!isNaN(dateObj)) {
                timeStr = dateObj.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit', hour12: false });
            }
        }

        const levelStr = level.toUpperCase().padEnd(7);
        let messagePayload = parsedLog.msg || '';

        // Replace tabs with spaces for clean indentation in pre-wrap blocks
        messagePayload = messagePayload.replace(/\t/g, '    ');

        const targetStr = parsedLog.object ? ` <span class="text-indigo-400/90 text-[11px] font-sans bg-indigo-950/40 px-1.5 py-0.5 rounded border border-indigo-900/30 ml-2">→ ${escapeHtml(parsedLog.object)}</span>` : '';

        return `
            <div class="py-1 border-b border-slate-900/40 hover:bg-slate-900/20 transition-colors flex flex-col sm:flex-row sm:items-start gap-x-3 text-[11px] font-mono leading-relaxed">
                <span class="text-slate-500 select-none shrink-0">[${escapeHtml(timeStr)}]</span>
                <span class="${lvlColor} select-none shrink-0 w-14">${escapeHtml(levelStr)}</span>
                <span class="text-slate-200 flex-1 whitespace-pre-wrap break-all">${escapeHtml(messagePayload.trim())}${targetStr}</span>
            </div>
        `;
    } catch (e) {
        // Fallthrough to plaintext if not JSON
    }

    let colorClass = "text-slate-400";
    if (line.includes("[ERROR]")) colorClass = "text-rose-400 font-semibold";
    else if (line.includes("[WARNING]")) colorClass = "text-amber-400";
    else if (line.includes("✅") || line.includes("SUCCESS")) colorClass = "text-emerald-400 font-medium";

    return `<div class="${colorClass} py-0.5 text-[11px] border-b border-slate-900/20 leading-relaxed font-mono whitespace-pre-wrap break-all">${escapeHtml(line)}</div>`;
}

function updateLogContainer(containerId, logLines) {
    const container = document.getElementById(containerId);
    if (!container || !logLines) return;

    const isAtBottom = container.scrollHeight - container.clientHeight <= container.scrollTop + 40;
    container.innerHTML = logLines.map(line => formatLogLine(line)).join('');

    if (isAtBottom) {
        container.scrollTop = container.scrollHeight;
    }
}

async function updateTelemetry() {
    try {
        const response = await fetch(`/api/telemetry?limit=${currentLogLimit}`);
        if (!response.ok) throw new Error(`HTTP network failure: ${response.status}`);

        const data = await response.json();
        
        // 1. Process dynamic configuration drop-downs
        if (data.config && !configLoaded) {
            const banner = document.getElementById('config-banner');
            const grid = document.getElementById('config-grid');
            
            if (document.getElementById('core-local-path')) document.getElementById('core-local-path').textContent = data.config.LOCAL_PATH || 'Not Set';
            if (document.getElementById('core-remote-path')) document.getElementById('core-remote-path').textContent = data.config.REMOTE_PATH || 'Not Set';
            if (document.getElementById('core-base-dir')) document.getElementById('core-base-dir').textContent = data.config.BASE_DIR || 'Not Set';

            if (grid && banner) {
                grid.innerHTML = ''; 
                banner.classList.remove('hidden');

                for (const [key, value] of Object.entries(data.config)) {
                    if (['LOCAL_PATH', 'REMOTE_PATH', 'BASE_DIR'].includes(key)) continue;

                    let formattedValue = '';
                    if (Array.isArray(value)) {
                        // Intelligent pairing for rclone argument structures
                        let argsHtml = '<div class="space-y-1 mt-1.5">';
                        for (let i = 0; i < value.length; i++) {
                            let currentItem = String(value[i]);
                            // If current item is a flag and next item isn't, pair them side-by-side
                            if (currentItem.startsWith('-') && i + 1 < value.length && !String(value[i + 1]).startsWith('-')) {
                                let nextValue = String(value[i + 1]);
                                argsHtml += `
                                    <div class="bg-slate-950 px-2 py-0.5 rounded text-[10px] text-slate-300 font-mono border border-slate-800/40 flex justify-between gap-x-2 truncate" title="${escapeHtml(currentItem)} ${escapeHtml(nextValue)}">
                                        <span class="text-slate-400 truncate">${escapeHtml(currentItem)}</span>
                                        <span class="text-amber-400 font-medium shrink-0">${escapeHtml(nextValue)}</span>
                                    </div>`;
                                i++; // Skip value element in next loop loop iteration
                            } else {
                                let isFlag = currentItem.startsWith('-');
                                let flagColor = isFlag ? 'text-slate-400' : 'text-amber-400 font-medium';
                                argsHtml += `
                                    <div class="bg-slate-950 px-2 py-0.5 rounded text-[10px] ${flagColor} font-mono border border-slate-800/40 truncate text-left" title="${escapeHtml(currentItem)}">
                                        ${escapeHtml(currentItem)}
                                    </div>`;
                            }
                        }
                        argsHtml += '</div>';
                        formattedValue = argsHtml;
                    } else if (typeof value === 'object' && value !== null) {
                        formattedValue = `<div class="space-y-1 mt-1.5 font-mono text-[10px]">` +
                            Object.entries(value).map(([subKey, subVal]) => {
                                const stateBadge = subVal 
                                    ? `<span class="text-emerald-400 font-semibold text-[9px] bg-emerald-950/40 px-1 py-0.2 rounded border border-emerald-900/30">True</span>` 
                                    : `<span class="text-rose-400 font-semibold text-[9px] bg-rose-950/40 px-1 py-0.2 rounded border border-rose-900/30">False</span>`;
                                return `<div class="flex justify-between items-center border-b border-slate-900 pb-1 last:border-0"><span class="text-slate-500 truncate mr-2" title="${escapeHtml(subKey)}">${escapeHtml(subKey)}:</span>${stateBadge}</div>`;
                            }).join('') + `</div>`;
                    } else if (typeof value === 'boolean') {
                        formattedValue = value 
                            ? `<span class="text-emerald-400 font-semibold bg-emerald-950/30 px-1.5 py-0.5 rounded border border-emerald-900/40 inline-block mt-1">True</span>` 
                            : `<span class="text-rose-400 font-semibold bg-rose-950/30 px-1.5 py-0.5 rounded border border-rose-900/40 inline-block mt-1">False</span>`;
                    } else {
                        const strVal = String(value);
                        formattedValue = `<span class="text-slate-300 block mt-1 select-all truncate" title="${escapeHtml(strVal)}">${escapeHtml(strVal)}</span>`;
                    }

                    const gridItem = document.createElement('div');
                    gridItem.className = "bg-slate-950/40 p-2.5 rounded-lg border border-slate-800/40 flex flex-col justify-between overflow-hidden hover:border-slate-700/60 transition-colors";
                    gridItem.innerHTML = `
                        <div class="text-[10px] text-slate-500 uppercase tracking-wider font-semibold truncate" title="${escapeHtml(key)}">${escapeHtml(key.replace(/_/g, ' '))}</div>
                        <div class="w-full">${formattedValue}</div>
                    `;
                    grid.appendChild(gridItem);
                }
                configLoaded = true; 
            }
        }
        
        // 2. Process Daemon health variables
        const daemonStringNode = document.getElementById('daemon-string');
        const daemonLatencyNode = document.getElementById('daemon-latency');
        if (daemonLatencyNode && data.daemon_health) {
            daemonLatencyNode.textContent = data.daemon_health.response_time_ms.toFixed(2);
        }
        if (daemonStringNode && data.daemon_health) {
            daemonStringNode.innerHTML = data.daemon_health.status === 'healthy' 
                ? `<span class="text-emerald-400 tracking-wide">ONLINE</span>` 
                : `<span class="text-rose-500 tracking-wide">OFFLINE</span>`;
        }

        // 3. Dynamic Execution Status Tile (Evaluates active syncing or most up-to-date entry)
        const syncStringNode = document.getElementById('sync-string');
        const syncTimestampNode = document.getElementById('sync-timestamp');
        
        if (data.notification_state) {
            const successTs = data.notification_state.last_success_timestamp || 0;
            const errorTs = data.notification_state.last_error_timestamp || 0;
            const activeTimestamp = Math.max(successTs, errorTs);
            const formattedLastTime = activeTimestamp 
                ? new Date(activeTimestamp * 1000).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit', hour12: false }) 
                : 'N/A';
            
            if (syncTimestampNode) {
                if (data.is_syncing) {
                    syncTimestampNode.innerHTML = `<span class="text-amber-400 font-medium animate-pulse">Sync active</span> (Last: ${formattedLastTime})`;
                } else {
                    syncTimestampNode.textContent = formattedLastTime;
                }
            }
            
            if (syncStringNode) {
                if (data.is_syncing) {
                    syncStringNode.innerHTML = `
                        <div class="flex items-center gap-2.5">
                            <span class="relative flex h-3 w-3">
                                <span class="animate-ping absolute inline-flex h-full w-full rounded-full bg-amber-400 opacity-75"></span>
                                <span class="relative inline-flex rounded-full h-3 w-3 bg-amber-500"></span>
                            </span>
                            <span class="text-amber-400 font-bold tracking-wide">SYNC IN PROGRESS</span>
                        </div>
                    `;
                } else if (errorTs > successTs && data.notification_state.last_error_type) {
                    syncStringNode.innerHTML = `<span class="text-rose-400 text-lg">FAULT (${escapeHtml(data.notification_state.last_error_type)})</span>`;
                } else if (successTs > 0) {
                    syncStringNode.innerHTML = `<span class="text-emerald-400">OPERATIONAL</span>`;
                } else {
                    syncStringNode.innerHTML = `<span class="text-slate-400">UNKNOWN</span>`;
                }
            }

            const progressCard = document.getElementById('sync-progress-card');
            const progressBar = document.getElementById('transfer-progress-bar');
            const progressPercentage = document.getElementById('transfer-percentage');
            const transferSpeed = document.getElementById('transfer-speed');
            const transferEta = document.getElementById('transfer-eta');
            const currentFileSpan = document.getElementById('transfer-current-file');

            if (data.is_syncing && data.transfer_stats && data.transfer_stats.total_bytes > 0) {
                progressCard.classList.remove('hidden');
                const stats = data.transfer_stats;
                progressBar.style.width = `${stats.percentage}%`;
                progressPercentage.textContent = `${stats.percentage}%`;
                transferSpeed.textContent = stats.speed_str;
                
                if (stats.eta !== null && stats.eta !== undefined) {
                    const mins = Math.floor(stats.eta / 60);
                    const secs = stats.eta % 60;
                    transferEta.textContent = `${mins}m ${secs}s`;
                } else {
                    transferEta.textContent = 'Calculating...';
                }
                
                if (stats.transferring && stats.transferring.length > 0) {
                    currentFileSpan.textContent = `Syncing: ${stats.transferring[0].name}`;
                } else {
                    currentFileSpan.textContent = 'Processing files...';
                }
            } else if (data.is_syncing) {
                progressCard.classList.remove('hidden');
                currentFileSpan.textContent = 'Preparing sync and checking index diffs...';
                progressBar.style.width = '100%';
                progressBar.classList.add('animate-pulse');
            } else {
                progressCard.classList.add('hidden');
                progressBar.classList.remove('animate-pulse');
            }
        }

        // 4. Update metrics sample depth counter
        if (document.getElementById('metrics-count') && data.metrics) {
            document.getElementById('metrics-count').textContent = data.metrics.length;
        }

        // 5. Update Line Charts Metrics Vector
        if (performanceChart && data.metrics && data.metrics.length > 0) {
            performanceChart.data.labels = data.metrics.map(m => new Date(m.datetime).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }));
            performanceChart.data.datasets[0].data = data.metrics.map(m => m.duration_seconds);
            const pointColors = data.metrics.map(m => m.success ? '#10b981' : '#f43f5e');
            performanceChart.data.datasets[0].pointBackgroundColor = pointColors;
            performanceChart.data.datasets[0].pointBorderColor = pointColors;
            performanceChart.update('none');
        }

        // Update sample depth counter in header if present
        const tailBadge = document.getElementById('log-tail-count');
        if (tailBadge) tailBadge.textContent = `Tail: ${currentLogLimit} lines`;

        // Push Clean Streams to Terminals
        updateLogContainer('log-bisync', data.bisync_logs);
        updateLogContainer('log-daemon', data.daemon_logs);

    } catch (error) {
        console.error("Dashboard telemetry synchronization error:", error);
    }
}

function setupLogScrolling(containerId) {
    const container = document.getElementById(containerId);
    if (!container) return;

    container.addEventListener('scroll', () => {
        // If user scrolls to the top (within 20px) and we haven't reached MAX_LOG_LIMIT
        if (container.scrollTop <= 20 && currentLogLimit < MAX_LOG_LIMIT) {
            const oldHeight = container.scrollHeight;
            currentLogLimit += 60; // Load 60 more lines
            updateTelemetry().then(() => {
                // Adjust scroll position so the user doesn't jump back to the top
                container.scrollTop = container.scrollHeight - oldHeight;
            });
        }
    });
}

async function triggerAction(actionStr, buttonId) {
    const btn = document.getElementById(buttonId);
    const originalText = btn.innerText;
    btn.innerText = "WAIT...";
    btn.disabled = true;
    btn.classList.add("opacity-50", "cursor-not-allowed");

    try {
        await fetch('/api/action', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ action: actionStr })
        });
        // Force an immediate UI refresh
        updateTelemetry();
    } catch (error) {
        console.error("Action execution failed:", error);
    } finally {
        setTimeout(() => {
            btn.innerText = originalText;
            btn.disabled = false;
            btn.classList.remove("opacity-50", "cursor-not-allowed");
        }, 1500);
    }
}

window.addEventListener('DOMContentLoaded', () => {
    initializeChart('livePerformanceChart');
    updateTelemetry();
    setupLogScrolling('log-daemon');
    setupLogScrolling('log-bisync');
    
    const toggleBtn = document.getElementById('toggle-extended-config');
    const extendedPanel = document.getElementById('extended-config-panel');
    const toggleIcon = document.getElementById('toggle-icon');

    // Hook up control buttons
    document.getElementById('btn-start')?.addEventListener('click', () => triggerAction('start_sync', 'btn-start'));
    document.getElementById('btn-stop')?.addEventListener('click', () => triggerAction('stop_sync', 'btn-stop'));
    document.getElementById('btn-unlock')?.addEventListener('click', () => triggerAction('remove_lock', 'btn-unlock'));
    document.getElementById('btn-force')?.addEventListener('click', () => {
        if(confirm("⚠ AUTHORIZE MASS DELETIONS ⚠\n\nAre you sure you want to approve deletions with --force?\nThis will permanently delete local files that were removed on Google Drive.")) {
            triggerAction('force_sync', 'btn-force');
        }
    });
    document.getElementById('btn-resync')?.addEventListener('click', () => {
        if(confirm("⚠ WARNING ⚠\n\nAre you sure you want to force a full resync?\nThis will rebuild the cache baseline and may take significantly longer.")) {
            triggerAction('force_resync', 'btn-resync');
        }
    });
    
    if (toggleBtn && extendedPanel && toggleIcon) {
        toggleBtn.addEventListener('click', () => {
            const isHidden = extendedPanel.classList.contains('hidden');
            if (isHidden) {
                extendedPanel.classList.remove('hidden');
                toggleIcon.classList.add('rotate-180');
                toggleBtn.querySelector('span').innerText = 'Hide Parameters';
            } else {
                extendedPanel.classList.add('hidden');
                toggleIcon.classList.remove('rotate-180');
                toggleBtn.querySelector('span').innerText = 'Show Full Parameters';
            }
        });
    }
    setInterval(updateTelemetry, 3000);
});
