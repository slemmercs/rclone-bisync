// Chart.js lifecycle management (isolated from Alpine reactivity proxies)
let performanceChart = null;

function initPerformanceChart(canvasId = "livePerformanceChart") {
    const canvas = document.getElementById(canvasId);
    if (!canvas) return null;
    const ctx = canvas.getContext("2d");
    performanceChart = new Chart(ctx, {
        type: "line",
        data: {
            labels: [],
            datasets: [{
                label: "Duration (s)",
                data: [],
                borderColor: "#6366f1",
                backgroundColor: "rgba(99, 102, 241, 0.03)",
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
                y: { grid: { color: "#1e293b" }, ticks: { color: "#475569", font: { family: "monospace" } } },
                x: { grid: { display: false }, ticks: { color: "#475569", font: { family: "monospace" } } }
            }
        }
    });
    return performanceChart;
}

function updatePerformanceChart(metrics) {
    if (!performanceChart || !metrics || metrics.length === 0) return;
    performanceChart.data.labels = metrics.map(m => new Date(m.datetime).toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" }));
    performanceChart.data.datasets[0].data = metrics.map(m => m.duration_seconds);
    const pointColors = metrics.map(m => m.success ? "#10b981" : "#f43f5e");
    performanceChart.data.datasets[0].pointBackgroundColor = pointColors;
    performanceChart.data.datasets[0].pointBorderColor = pointColors;
    performanceChart.update("none");
}

