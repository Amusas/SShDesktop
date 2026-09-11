// Cliente WebSocket con reconexion automatica y soporte para parametros de sesion
class TerminalWebSocket {
    constructor(url, callbacks) {
        this.url = url || '/ws';
        this.callbacks = callbacks || {};
        this.ws = null;
        this.reconnectDelay = 2000;
        this.maxReconnectDelay = 30000;
        this.shouldReconnect = true;
        this.connectionParams = {};
    }

    connect(params) {
        if (params) {
            this.connectionParams = { ...params };
        }

        const protocol = location.protocol === 'https:' ? 'wss:' : 'ws:';
        const searchParams = new URLSearchParams();

        for (const [key, val] of Object.entries(this.connectionParams)) {
            if (val !== undefined && val !== null && val !== '') {
                searchParams.append(key, val);
            }
        }

        const qs = searchParams.toString();
        const wsUrl = `${protocol}//${location.host}${this.url}${qs ? '?' + qs : ''}`;

        if (this.callbacks.onStatusChange) {
            this.callbacks.onStatusChange('connecting');
        }

        try {
            this.ws = new WebSocket(wsUrl);
        } catch (e) {
            if (this.callbacks.onStatusChange) {
                this.callbacks.onStatusChange('disconnected');
            }
            return;
        }

        this.ws.onopen = () => {
            this.reconnectDelay = 2000;
            if (this.callbacks.onStatusChange) {
                this.callbacks.onStatusChange('connected');
            }
            if (this.callbacks.onOpen) {
                this.callbacks.onOpen();
            }
        };

        this.ws.onmessage = (event) => {
            if (this.callbacks.onMessage) {
                this.callbacks.onMessage(event.data);
            }
        };

        this.ws.onclose = () => {
            if (this.callbacks.onStatusChange) {
                this.callbacks.onStatusChange('disconnected');
            }
            if (this.callbacks.onClose) {
                this.callbacks.onClose();
            }
            if (this.shouldReconnect) {
                setTimeout(() => this.connect(), this.reconnectDelay);
                this.reconnectDelay = Math.min(this.reconnectDelay * 2, this.maxReconnectDelay);
            }
        };

        this.ws.onerror = () => {
            if (this.callbacks.onStatusChange) {
                this.callbacks.onStatusChange('disconnected');
            }
        };
    }

    reconnectWith(params) {
        this.shouldReconnect = false;
        if (this.ws) {
            this.ws.close();
            this.ws = null;
        }
        this.shouldReconnect = true;
        this.reconnectDelay = 2000;
        this.connect(params);
    }

    sendInput(data) {
        if (this.ws && this.ws.readyState === WebSocket.OPEN) {
            this.ws.send(JSON.stringify({ type: 'input', data }));
        }
    }

    sendResize(cols, rows) {
        if (this.ws && this.ws.readyState === WebSocket.OPEN) {
            this.ws.send(JSON.stringify({ type: 'resize', cols, rows }));
        }
    }

    disconnect() {
        this.shouldReconnect = false;
        if (this.ws) {
            this.ws.close();
            this.ws = null;
        }
    }
}
