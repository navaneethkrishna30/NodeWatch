function fetchData() {
  fetch('/status')
    .then(res => res.json())
    .then(data => {
      const statusTextEl = document.getElementById('statusText');
      const statusIconEl = document.getElementById('statusIcon');
      const logEl = document.getElementById('logbox');
      const updatedAtEl = document.getElementById('updatedAt');
      const subscriptionIdEl = document.getElementById('subscriptionId');
      const nodeTypeEl = document.getElementById('nodeType');
      
      if (data.status) {
        statusTextEl.textContent = '🟢 Running';
        statusIconEl.className = 'status-icon ok';
      } else {
        statusTextEl.textContent = '🔴 Not running';
        statusIconEl.className = 'status-icon fail';
      }

      subscriptionIdEl.textContent = data.subscription_id;
      nodeTypeEl.textContent = data.node_type;

      const shouldAutoScroll = (logEl.scrollHeight - logEl.scrollTop - logEl.clientHeight) < 10;

      // Show logs or a message if none are available
      let logsText = '';
      if (Array.isArray(data.logs) && data.logs.length > 0) {
        logsText = data.logs.join('\n');
      } else {
        logsText = 'No logs available.';
      }

      // If node is not running, prepend a message
      if (!data.status) {
        logsText = '[Node is not running]\n' + logsText;
      }

      logEl.textContent = logsText;

      if (shouldAutoScroll) {
        logEl.scrollTop = logEl.scrollHeight;
      }

      const dateObj = new Date(data.last_updated_at);
      const pad = n => n.toString().padStart(2, '0');
      const timestamp = `${pad(dateObj.getUTCDate())}-${pad(dateObj.getUTCMonth() + 1)}-${pad(dateObj.getUTCFullYear())} ${pad(dateObj.getUTCHours())}:${pad(dateObj.getUTCMinutes())}:${pad(dateObj.getUTCSeconds())} UTC`;
      updatedAtEl.textContent = 'Last updated at: ' + timestamp;
    })
    .catch(error => {
      console.error('Error fetching metrics:', error);
      document.getElementById('statusText').textContent = '❌ Error';
      document.getElementById('logbox').textContent = 'Error fetching data: ' + error.message;
    });
}

// Poll for new data every 30 seconds
setInterval(fetchData, 30000);
fetchData(); 