import { useEffect, useState, useCallback } from 'react';

export default function NotificationListener({ onNotificationReceived, merchantId }) {
  const [ws, setWs] = useState(null);
  const [isConnected, setIsConnected] = useState(false);
  const [retryCount, setRetryCount] = useState(0);
  const maxRetries = 5;

  useEffect(() => {
    if (!merchantId) {
      console.warn('⚠ Merchant ID not provided');
      return;
    }

    // ✨ Setup WebSocket connection
    const connectWebSocket = () => {
      try {
        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        const wsURL = `${protocol}//${window.location.hostname}:8080/ws?merchant_id=${merchantId}`;

        console.log(`🔌 Connecting to WebSocket: ${wsURL}`);

        const socket = new WebSocket(wsURL);

        // On connection open
        socket.onopen = () => {
          console.log('✓ WebSocket connected');
          setIsConnected(true);
          setRetryCount(0); // Reset retry counter

          // Optional: Send heartbeat/ping
          const heartbeat = setInterval(() => {
            if (socket.readyState === WebSocket.OPEN) {
              socket.send(JSON.stringify({ type: 'ping' }));
            }
          }, 30000); // Ping setiap 30 detik

          socket.onclose = () => clearInterval(heartbeat);
        };

        // On message received
        socket.onmessage = (event) => {
          try {
            const message = JSON.parse(event.data);
            console.log('📨 Notification received:', message);

            // Filter: hanya transaction notifications
            if (message.type === 'transaction_notification') {
              onNotificationReceived(message);
            }
          } catch (err) {
            console.error('❌ Failed to parse message:', err);
          }
        };

        // On error
        socket.onerror = (error) => {
          console.error('❌ WebSocket error:', error);
          setIsConnected(false);
        };

        // On connection close
        socket.onclose = () => {
          console.log('⚠ WebSocket disconnected');
          setIsConnected(false);

          // Auto-reconnect dengan exponential backoff
          if (retryCount < maxRetries) {
            const delayMs = Math.min(1000 * Math.pow(2, retryCount), 30000);
            console.log(`🔄 Reconnecting in ${delayMs}ms...`);
            setTimeout(() => {
              setRetryCount(prev => prev + 1);
              connectWebSocket();
            }, delayMs);
          } else {
            console.error('❌ Max retries reached, giving up');
          }
        };

        setWs(socket);
      } catch (err) {
        console.error('❌ WebSocket connection error:', err);
      }
    };

    connectWebSocket();

    // Cleanup on unmount
    return () => {
      if (ws) {
        ws.close();
      }
    };
  }, [merchantId, onNotificationReceived, retryCount]);

  return (
    <div style={{ fontSize: '12px', color: '#666', position: 'fixed', bottom: '10px', left: '10px' }}>
      WebSocket: <span style={{ color: isConnected ? '#10b981' : '#ef4444' }}>
        {isConnected ? '🟢 Connected' : '🔴 Disconnected'}
      </span>
    </div>
  );
}