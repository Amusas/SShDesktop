// Cliente WebSocket con reconexion automatica y soporte para parametros de sesion
class TerminalWebSocket {
    constructor(url, callbacks) {
        this.url = url || '/ws';
        this.callbacks = callbacks || {};
        this.ws = null;
        this.reconnectDelay = 2000;
        this.maxReconnectDelay = 30000;
        this.shouldReconnect = true;
        this.reconnectTimer = null;
        this.connectionParams = {};
    }

    connect(params) {
        // Limpiar cualquier intento previo de reconexión programada
        if (this.reconnectTimer) {
            clearTimeout(this.reconnectTimer);
            this.reconnectTimer = null;
        }

        if (params) {
            this.connectionParams = { ...params };
        }

        // Si ya existía un socket previo, desvincular todos sus manejadores y cerrarlo
        // para evitar que sus eventos tardíos ('onclose', 'onmessage') dupliquen tráfico
        if (this.ws) {
            const oldSocket = this.ws;
            oldSocket.onopen = null;
            oldSocket.onmessage = null;
            oldSocket.onerror = null;
            oldSocket.onclose = null;
            try {
                oldSocket.close();
            } catch (e) {}
            this.ws = null;
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

        const currentSocket = this.ws;

        currentSocket.onopen = () => {
            if (this.ws !== currentSocket) return;

            this.reconnectDelay = 2000;
            if (this.callbacks.onStatusChange) {
                this.callbacks.onStatusChange('connected');
            }
            if (this.callbacks.onOpen) {
                this.callbacks.onOpen();
            }
        };

        currentSocket.onmessage = (event) => {
            if (this.ws !== currentSocket) return;

            if (this.callbacks.onMessage) {
                this.callbacks.onMessage(event.data);
            }
        };

        currentSocket.onclose = () => {
            if (this.ws !== currentSocket) return;
            this.ws = null;

            if (this.callbacks.onStatusChange) {
                this.callbacks.onStatusChange('disconnected');
            }
            if (this.callbacks.onClose) {
                this.callbacks.onClose();
            }
            if (this.shouldReconnect) {
                if (this.reconnectTimer) clearTimeout(this.reconnectTimer);
                this.reconnectTimer = setTimeout(() => this.connect(), this.reconnectDelay);
                this.reconnectDelay = Math.min(this.reconnectDelay * 2, this.maxReconnectDelay);
            }
        };

        currentSocket.onerror = () => {
            if (this.ws !== currentSocket) return;

            if (this.callbacks.onStatusChange) {
                this.callbacks.onStatusChange('disconnected');
            }
        };
    }

    reconnectWith(params) {
        if (this.reconnectTimer) {
            clearTimeout(this.reconnectTimer);
            this.reconnectTimer = null;
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
        if (this.reconnectTimer) {
            clearTimeout(this.reconnectTimer);
            this.reconnectTimer = null;
        }
        if (this.ws) {
            this.ws.onopen = null;
            this.ws.onmessage = null;
            this.ws.onerror = null;
            this.ws.onclose = null;
            try {
                this.ws.close();
            } catch (e) {}
            this.ws = null;
        }
    }
}
