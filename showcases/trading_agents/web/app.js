// API Configuration
const API_BASE_URL = 'http://localhost:8080';

// Form elements
const form = document.getElementById('analysisForm');
const symbolInput = document.getElementById('symbol');
const capitalInput = document.getElementById('capital');
const riskToleranceSelect = document.getElementById('riskTolerance');
const timeframeSelect = document.getElementById('timeframe');
const analyzeBtn = document.getElementById('analyzeBtn');
const btnText = analyzeBtn.querySelector('.btn-text');
const spinner = analyzeBtn.querySelector('.spinner');

// Results elements
const resultsSection = document.getElementById('results');
const statsCard = document.getElementById('statsCard');

// Event listeners
form.addEventListener('submit', handleAnalysis);

// Handle form submission
async function handleAnalysis(e) {
    e.preventDefault();

    const symbol = symbolInput.value.trim().toUpperCase();
    if (!symbol) {
        alert('Please enter a stock symbol');
        return;
    }

    // Show loading state
    setLoading(true);

    try {
        const request = {
            symbol: symbol,
            capital: parseFloat(capitalInput.value),
            risk_tolerance: riskToleranceSelect.value,
            timeframe: timeframeSelect.value
        };

        const response = await fetch(`${API_BASE_URL}/api/analyze`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify(request)
        });

        if (!response.ok) {
            throw new Error(`API error: ${response.statusText}`);
        }

        const result = await response.json();
        displayResults(result);

    } catch (error) {
        console.error('Analysis failed:', error);
        alert(`Analysis failed: ${error.message}\n\nMake sure the backend server is running on ${API_BASE_URL}`);
    } finally {
        setLoading(false);
    }
}

// Display analysis results
function displayResults(result) {
    // Show results section
    resultsSection.style.display = 'block';
    statsCard.style.display = 'block';

    // Scroll to results
    resultsSection.scrollIntoView({ behavior: 'smooth' });

    // Update quick stats
    const marketData = result.metadata.market_data;
    document.getElementById('currentPrice').textContent = `$${marketData.price.toFixed(2)}`;
    document.getElementById('priceChange').textContent = formatChange(marketData.change, marketData.change_percent);
    document.getElementById('volume').textContent = formatVolume(marketData.volume);

    // Update recommendation
    const recBadge = document.getElementById('recommendation');
    recBadge.textContent = result.recommendation;
    recBadge.className = `recommendation-badge ${result.recommendation}`;

    // Update confidence
    const confidence = result.confidence;
    document.getElementById('confidenceFill').style.width = `${confidence}%`;
    document.getElementById('confidenceValue').textContent = `${confidence.toFixed(1)}%`;

    // Update risk score
    const riskScore = result.risk_score;
    document.getElementById('riskScore').textContent = `${riskScore.toFixed(1)}/100`;
    document.getElementById('riskFill').style.width = `${riskScore}%`;

    // Update timestamp
    document.getElementById('timestamp').textContent = formatTimestamp(result.timestamp);

    // Update trade details
    if (result.position_size > 0) {
        document.getElementById('tradeDetails').style.display = 'block';
        document.getElementById('positionSize').textContent = `${result.position_size.toFixed(2)} shares`;
        document.getElementById('entryPrice').textContent = `$${result.metadata.current_price.toFixed(2)}`;
        document.getElementById('stopLoss').textContent = `$${result.stop_loss.toFixed(2)}`;
        document.getElementById('takeProfit').textContent = `$${result.take_profit.toFixed(2)}`;
    } else {
        document.getElementById('tradeDetails').style.display = 'none';
    }

    // Update reasoning
    document.getElementById('reasoning').textContent = result.reasoning;

    // Update agent reports
    document.getElementById('fundamentals-tab').textContent = result.reports.fundamentals || 'No report available';
    document.getElementById('sentiment-tab').textContent = result.reports.sentiment || 'No report available';
    document.getElementById('technical-tab').textContent = result.reports.technical || 'No report available';
    document.getElementById('bullish-tab').textContent = result.reports.bullish || 'No report available';
    document.getElementById('bearish-tab').textContent = result.reports.bearish || 'No report available';
    document.getElementById('risk-tab').textContent = result.reports.risk || 'No report available';
}

// Tab switching
function showTab(tabName) {
    // Hide all tabs
    document.querySelectorAll('.tab').forEach(tab => {
        tab.classList.remove('active');
    });
    document.querySelectorAll('.tab-pane').forEach(pane => {
        pane.classList.remove('active');
    });

    // Show selected tab
    event.target.classList.add('active');
    document.getElementById(`${tabName}-tab`).classList.add('active');
}

// Helper functions
function setLoading(loading) {
    if (loading) {
        analyzeBtn.disabled = true;
        btnText.style.display = 'none';
        spinner.style.display = 'inline';
        form.classList.add('loading');
    } else {
        analyzeBtn.disabled = false;
        btnText.style.display = 'inline';
        spinner.style.display = 'none';
        form.classList.remove('loading');
    }
}

function formatChange(change, changePercent) {
    const sign = change >= 0 ? '+' : '';
    const color = change >= 0 ? '#10b981' : '#ef4444';
    return `<span style="color: ${color}">${sign}$${change.toFixed(2)} (${sign}${changePercent.toFixed(2)}%)</span>`;
}

function formatVolume(volume) {
    if (volume >= 1000000) {
        return `${(volume / 1000000).toFixed(2)}M`;
    } else if (volume >= 1000) {
        return `${(volume / 1000).toFixed(2)}K`;
    }
    return volume.toFixed(0);
}

function formatTimestamp(timestamp) {
    const date = new Date(timestamp);
    return date.toLocaleString('en-US', {
        month: 'short',
        day: 'numeric',
        year: 'numeric',
        hour: '2-digit',
        minute: '2-digit'
    });
}

// Make showTab globally accessible
window.showTab = showTab;

// Check backend health on load
window.addEventListener('load', async () => {
    try {
        const response = await fetch(`${API_BASE_URL}/health`);
        if (response.ok) {
            console.log('✅ Backend server is running');
        }
    } catch (error) {
        console.warn('⚠️ Backend server is not running. Please start it with: ./bin/trading-agents');
    }
});
