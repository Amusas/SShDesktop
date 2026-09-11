// Gestor de la interfaz de terminal web, configuracion dinamica y conexion WebSocket
class TerminalApp {
    constructor() {
        // Elementos principales de la terminal
        this.outputEl = document.getElementById('output');
        this.inputEl = document.getElementById('command-input');
        this.statusEl = document.getElementById('status');
        this.promptPathEl = document.getElementById('prompt-path');

        // Elementos de cabecera y estado
        this.targetUserHostEl = document.getElementById('target-user-host');
        this.targetShellBadgeEl = document.getElementById('target-shell-badge');
        this.menuBtn = document.getElementById('menu-btn');
        this.reconnectBtn = document.getElementById('reconnect-btn');

        // Elementos del modal de configuracion
        this.modalEl = document.getElementById('config-modal');
        this.closeModalBtn = document.getElementById('close-modal-btn');
        this.cancelModalBtn = document.getElementById('btn-cancel-modal');
        this.resetDefaultsBtn = document.getElementById('btn-reset-defaults');
        this.configForm = document.getElementById('config-form');
        this.modalFeedback = document.getElementById('modal-feedback');

        // Inputs del formulario
        this.cfgHost = document.getElementById('cfg-host');
        this.cfgPort = document.getElementById('cfg-port');
        this.cfgUser = document.getElementById('cfg-user');
        this.cfgStartPath = document.getElementById('cfg-start-path');
        this.cfgKeyPath = document.getElementById('cfg-key-path');

        // Estado de la terminal
        this.commandHistory = [];
        this.historyIndex = -1;
        this.autoScroll = true;
        this.activeClasses = [];
        this.currentLineEl = null;
        this.pendingText = '';
        this.hiddenRemotePrompt = false;
        this.remotePromptCandidate = false;
        // Las secuencias VT pueden llegar partidas entre dos mensajes WebSocket.
        this.pendingEscapeSequence = '';
        this.currentPath = '~';

        // Configuracion por defecto y activa
        this.defaultConfig = null;
        this.currentConfig = {};

        this.ws = new TerminalWebSocket('/ws', {
            onStatusChange: (status) => this.setStatus(status),
            onOpen: () => {
                this.handleResize();
            },
            onMessage: (data) => this.handleOutput(data),
            onClose: () => {},
        });

        this.bindEvents();
        this.loadInitialConfig();
    }

    async loadInitialConfig() {
        try {
            const res = await fetch('/api/config');
            if (res.ok) {
                const data = await res.json();
                this.defaultConfig = { ...data };
                this.currentConfig = { ...data };

                this.populateForm(this.currentConfig);
                this.updateTargetPill(this.currentConfig);
                this.updateCurrentPath(this.currentConfig.sshStartPath || '~');

                // Si el host o usuario no están configurados, abrir modal automáticamente
                if (!this.currentConfig.sshHost || !this.currentConfig.sshUser) {
                    this.openModal();
                    this.showModalFeedback('Por favor configura el host y usuario para conectarte.', 'error');
                } else {
                    this.connectCurrent();
                }
            } else {
                this.connectCurrent();
            }
        } catch (e) {
            console.warn('No se pudo obtener configuracion inicial desde la API:', e);
            this.connectCurrent();
        }
    }

    populateForm(cfg) {
        if (!cfg) return;
        this.cfgHost.value = cfg.sshHost || '';
        this.cfgPort.value = cfg.sshPort || 22;
        this.cfgUser.value = cfg.sshUser || '';
        this.cfgStartPath.value = cfg.sshStartPath || '';
        this.cfgKeyPath.value = cfg.sshKeyPath || '';

        const shellVal = (cfg.sshShell || 'powershell').toLowerCase();
        const radio = this.configForm.querySelector(`input[name="shell"][value="${shellVal}"]`);
        if (radio) {
            radio.checked = true;
        }
    }

    getSelectedShell() {
        const checked = this.configForm.querySelector('input[name="shell"]:checked');
        return checked ? checked.value : 'powershell';
    }

    updateTargetPill(cfg) {
        if (!cfg) return;
        const user = cfg.sshUser || 'usuario';
        const host = cfg.sshHost || 'sin-host';
        const port = cfg.sshPort || 22;
        const shell = (cfg.sshShell || 'powershell').toLowerCase();

        if (this.targetUserHostEl) {
            this.targetUserHostEl.textContent = `${user}@${host}:${port}`;
            this.targetUserHostEl.title = `Conectado a ${user}@${host}:${port}`;
        }

        if (this.targetShellBadgeEl) {
            this.targetShellBadgeEl.textContent = shell;
            this.targetShellBadgeEl.className = `badge-shell ${shell}`;
        }
    }

    connectCurrent() {
        const shell = this.currentConfig.sshShell || 'powershell';
        const params = {
            host: this.currentConfig.sshHost || '',
            port: this.currentConfig.sshPort || 22,
            user: this.currentConfig.sshUser || '',
            shell: shell,
            startPath: this.currentConfig.sshStartPath || '',
            keyPath: this.currentConfig.sshKeyPath || '',
        };

        this.ws.reconnectWith(params);
    }

    bindEvents() {
        // Eventos de teclado en la linea de comandos
        this.inputEl.addEventListener('keydown', (e) => this.handleKeyDown(e));

        // Control de autoscroll en la salida
        this.outputEl.addEventListener('scroll', () => {
            const { scrollTop, scrollHeight, clientHeight } = this.outputEl;
            this.autoScroll = (scrollHeight - scrollTop - clientHeight) < 60;
        });

        // Redimensionamiento de ventana
        window.addEventListener('resize', () => this.handleResize());

        // Foco inteligente en terminal (sin interferir con modales ni botones de cabecera)
        document.addEventListener('click', (e) => {
            if (this.isModalOpen() || e.target.closest('#config-modal') || e.target.closest('#header')) {
                return;
            }
            const selection = window.getSelection();
            if (!selection || selection.isCollapsed) {
                this.inputEl.focus();
            }
        });

        // Apertura del modal de configuracion
        this.menuBtn.addEventListener('click', () => {
            this.openModal();
        });

        // Boton de reconexion directa
        this.reconnectBtn.addEventListener('click', () => {
            this.clearOutput();
            this.appendInfoLine('[Reconectando sesión SSH...]');
            this.connectCurrent();
        });

        // Cierre del modal
        this.closeModalBtn.addEventListener('click', () => this.closeModal());
        this.cancelModalBtn.addEventListener('click', () => this.closeModal());

        // Cierre con tecla Escape
        window.addEventListener('keydown', (e) => {
            if (e.key === 'Escape' && this.isModalOpen()) {
                this.closeModal();
            }
        });

        // Cierre al hacer clic fuera del dialogo modal
        this.modalEl.addEventListener('click', (e) => {
            if (e.target === this.modalEl) {
                this.closeModal();
            }
        });

        // Boton de valores por defecto
        this.resetDefaultsBtn.addEventListener('click', () => {
            if (this.defaultConfig) {
                this.populateForm(this.defaultConfig);
                this.showModalFeedback('Valores por defecto cargados en el formulario.', 'success');
            }
        });

        // Envio del formulario de configuracion (Guardar y Conectar)
        this.configForm.addEventListener('submit', (e) => this.handleConfigSubmit(e));
    }

    isModalOpen() {
        return this.modalEl && !this.modalEl.classList.contains('hidden');
    }

    openModal() {
        this.populateForm(this.currentConfig);
        this.clearModalFeedback();
        this.modalEl.classList.remove('hidden');
        this.modalEl.setAttribute('aria-hidden', 'false');
        setTimeout(() => this.cfgHost.focus(), 50);
    }

    closeModal() {
        this.modalEl.classList.add('hidden');
        this.modalEl.setAttribute('aria-hidden', 'true');
        this.clearModalFeedback();
        this.inputEl.focus();
    }

    showModalFeedback(text, type = 'error') {
        if (!this.modalFeedback) return;
        this.modalFeedback.textContent = text;
        this.modalFeedback.className = `modal-feedback ${type}`;
    }

    clearModalFeedback() {
        if (!this.modalFeedback) return;
        this.modalFeedback.textContent = '';
        this.modalFeedback.className = 'modal-feedback hidden';
    }

    async handleConfigSubmit(e) {
        e.preventDefault();

        const host = this.cfgHost.value.trim();
        const port = parseInt(this.cfgPort.value.trim(), 10) || 22;
        const user = this.cfgUser.value.trim();
        const shell = this.getSelectedShell();
        const startPath = this.cfgStartPath.value.trim();
        const keyPath = this.cfgKeyPath.value.trim();

        if (!host) {
            this.showModalFeedback('El host o IP del servidor es obligatorio.', 'error');
            this.cfgHost.focus();
            return;
        }

        if (!user) {
            this.showModalFeedback('El usuario SSH es obligatorio.', 'error');
            this.cfgUser.focus();
            return;
        }

        if (!keyPath) {
            this.showModalFeedback('La ubicación de la llave privada es obligatoria.', 'error');
            this.cfgKeyPath.focus();
            return;
        }

        const newConfig = {
            sshHost: host,
            sshPort: port,
            sshUser: user,
            sshShell: shell,
            sshStartPath: startPath,
            sshKeyPath: keyPath,
        };

        // Guardar configuracion en backend (persistencia en config.properties)
        try {
            await fetch('/api/config', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(newConfig),
            });
        } catch (err) {
            console.warn('No se pudo guardar la configuracion en el backend:', err);
        }

        this.currentConfig = { ...newConfig };
        this.updateTargetPill(this.currentConfig);
        this.updateCurrentPath(this.currentConfig.sshStartPath || '~');

        this.closeModal();

        this.clearOutput();
        this.appendInfoLine(`[Iniciando sesión SSH a ${user}@${host}:${port} (${shell})...]`);

        this.connectCurrent();
    }

    handleKeyDown(e) {
        switch (e.key) {
            case 'Enter':
                e.preventDefault();
                this.submitCommand();
                break;
            case 'ArrowUp':
                e.preventDefault();
                this.navigateHistory(-1);
                break;
            case 'ArrowDown':
                e.preventDefault();
                this.navigateHistory(1);
                break;
            case 'c':
                if (e.ctrlKey) {
                    this.ws.sendInput('\x03');
                    this.inputEl.value = '';
                }
                break;
            case 'l':
                if (e.ctrlKey) {
                    e.preventDefault();
                    this.clearOutput();
                }
                break;
        }
    }

    submitCommand() {
        const command = this.inputEl.value;
        if (command.trim() === '') return;

        this.commandHistory.push(command);
        this.historyIndex = this.commandHistory.length;

        const cmdTrimmed = command.trim();
        const cmdLower = cmdTrimmed.toLowerCase();

        if (cmdLower === 'cls' || cmdLower === 'clear') {
            this.clearOutput();
            this.ws.sendInput(cmdTrimmed + '\r\n');
            this.inputEl.value = '';
            this.autoScroll = true;
            return;
        }

        this.autoScroll = true;
        this.ws.sendInput(command + '\r\n');
        this.inputEl.value = '';
        this.scrollToBottom();
    }

    navigateHistory(direction) {
        if (this.commandHistory.length === 0) return;

        this.historyIndex += direction;

        if (this.historyIndex < 0) {
            this.historyIndex = 0;
        } else if (this.historyIndex >= this.commandHistory.length) {
            this.historyIndex = this.commandHistory.length;
            this.inputEl.value = '';
            return;
        }

        this.inputEl.value = this.commandHistory[this.historyIndex];
    }

    handleOutput(rawData) {
        if (!rawData) return;

        if (typeof rawData === 'string' && rawData.startsWith('{')) {
            try {
                const parsed = JSON.parse(rawData);
                if (parsed.type === 'error') {
                    this.appendErrorLine(`[Error SSH]: ${parsed.data}`);
                    return;
                }
            } catch (e) {
                // No es JSON, continuar como flujo ANSI regular
            }
        }

        rawData = this.takeCompleteEscapeSequences(rawData);
        if (!rawData) return;

        if (AnsiParser.hasClearScreen(rawData)) {
            this.clearOutput();
            rawData = AnsiParser.removeClearScreen(rawData);
            if (!rawData) return;
        }

        const detectedPath = AnsiParser.extractPath(rawData);
        if (detectedPath) {
            this.updateCurrentPath(detectedPath);
        }

        this.processText(rawData);
    }

    takeCompleteEscapeSequences(text) {
        const combined = this.pendingEscapeSequence + text;
        this.pendingEscapeSequence = '';

        const lastEscape = combined.lastIndexOf('\x1b');
        if (lastEscape === -1) return combined;

        const tail = combined.slice(lastEscape);
        // CSI incompleta (por ejemplo ESC[2, a la espera de la J) u OSC sin terminar.
        if ((tail.startsWith('\x1b[') && !/\x1b\[[0-?]*[ -/]*[@-~]$/.test(tail)) ||
            (tail.startsWith('\x1b]') && !/(?:\x07|\x1b\\)$/.test(tail))) {
            this.pendingEscapeSequence = tail;
            return combined.slice(0, lastEscape);
        }

        return combined;
    }

    removeBackspaces(str) {
        if (!str || !str.includes('\b')) return str;
        const result = [];
        for (let i = 0; i < str.length; i++) {
            const ch = str[i];
            if (ch === '\b') {
                if (result.length > 0 && result[result.length - 1] !== '\n' && result[result.length - 1] !== '\r') {
                    result.pop();
                }
            } else {
                result.push(ch);
            }
        }
        return result.join('');
    }

    processText(text) {
        if (!text) return;

        const processed = this.removeBackspaces(text);
        const tokens = processed.split(/(\r\n|\r|\n)/);

        for (const token of tokens) {
            if (token === '') continue;

            if (token === '\r\n' || token === '\n') {
                this.finishLine();
            } else if (token === '\r') {
                this.pendingText = '';
                if (this.currentLineEl) {
                    this.currentLineEl.innerHTML = '';
                }
            } else {
                this.appendToCurrentLine(token);
            }
        }

        this.scrollToBottom();
    }

    appendToCurrentLine(text) {
        if (!this.currentLineEl) {
            this.currentLineEl = this.createLineElement();
            this.outputEl.appendChild(this.currentLineEl);
            this.hiddenRemotePrompt = false;
            this.remotePromptCandidate = false;
        }

        this.pendingText += text;
        const detectedPath = AnsiParser.extractPath(this.pendingText);
        if (detectedPath) {
            this.updateCurrentPath(detectedPath);
        }
        const promptState = this.getRemotePromptState(this.pendingText);
        if (this.hiddenRemotePrompt || promptState === 'prompt') {
            // El prompt se muestra exclusivamente en la barra de entrada propia.
            this.hiddenRemotePrompt = true;
            this.remotePromptCandidate = false;
            this.currentLineEl.innerHTML = '';
            return;
        }

        if (promptState === 'candidate') {
            // Esperar solo el siguiente fragmento para distinguir "P" de "PS ".
            // No se marca como prompt todavía: puede ser salida real (p. ej. ping).
            this.remotePromptCandidate = true;
            this.currentLineEl.innerHTML = '';
            return;
        }

        this.remotePromptCandidate = false;
        const result = AnsiParser.renderToHTML(this.pendingText, []);
        this.activeClasses = result.classes;
        this.currentLineEl.innerHTML = result.html;
    }

    getRemotePromptState(text) {
        const plain = AnsiParser.stripAnsi(text).replace(/\r/g, '');
        const shell = (this.currentConfig.sshShell || '').toLowerCase();

        if (shell === 'powershell' || shell === 'powershell.exe' || shell === 'pwsh') {
            if (/^P(?:S\s*)?$/.test(plain)) return 'candidate';
            if (/^PS\s+[a-zA-Z]:[\\/][^>\r\n]*>.*$/.test(plain)) return 'prompt';
            return null;
        }

        if (/^[a-zA-Z]:[\\/][^>\r\n]*$/.test(plain)) return 'candidate';
        if (/^[a-zA-Z]:[\\/][^>\r\n]*>.*$/.test(plain)) return 'prompt';
        return null;
    }

    finishLine() {
        if (this.currentLineEl) {
            if (this.remotePromptCandidate && !this.hiddenRemotePrompt) {
                const result = AnsiParser.renderToHTML(this.pendingText, []);
                this.currentLineEl.innerHTML = result.html;
            }
            const textContent = this.currentLineEl.textContent.trim();
            if (this.hiddenRemotePrompt) {
                this.currentLineEl.remove();
            } else if (textContent === '') {
                const prev = this.currentLineEl.previousElementSibling;
                if (!prev || prev.textContent.trim() === '') {
                    this.currentLineEl.remove();
                }
            }
        }
        this.currentLineEl = null;
        this.pendingText = '';
        this.hiddenRemotePrompt = false;
        this.remotePromptCandidate = false;
    }

    appendErrorLine(text) {
        const line = this.createLineElement();
        line.style.color = 'var(--accent-red)';
        line.textContent = text;
        this.outputEl.appendChild(line);
        this.finishLine();
        this.scrollToBottom();
    }

    appendInfoLine(text) {
        const line = this.createLineElement();
        line.style.color = 'var(--accent)';
        line.style.fontStyle = 'italic';
        line.textContent = text;
        this.outputEl.appendChild(line);
        this.finishLine();
        this.scrollToBottom();
    }

    createLineElement() {
        const line = document.createElement('div');
        line.className = 'line';
        return line;
    }

    clearOutput() {
        this.outputEl.innerHTML = '';
        this.currentLineEl = null;
        this.pendingText = '';
        this.hiddenRemotePrompt = false;
        this.remotePromptCandidate = false;
        this.activeClasses = [];
        this.pendingEscapeSequence = '';
    }

    scrollToBottom() {
        if (this.autoScroll) {
            this.outputEl.scrollTop = this.outputEl.scrollHeight;
        }
    }

    updateCurrentPath(path) {
        if (!path || path === this.currentPath) return;
        this.currentPath = path;

        if (this.promptPathEl) {
            this.promptPathEl.textContent = path;
            this.promptPathEl.title = path;
        }
    }

    setStatus(status) {
        this.statusEl.className = `status ${status}`;
        const labels = {
            connected: 'Conectado',
            disconnected: 'Desconectado',
            connecting: 'Conectando...',
        };
        this.statusEl.textContent = labels[status] || status;

        // La ruta es parte del estado de la sesión, no de la salida visual. Se
        // conserva mientras el WebSocket se reconecta o se limpia la terminal.
    }

    handleResize() {
        const cols = Math.max(40, Math.floor(this.outputEl.clientWidth / 8.5));
        const rows = Math.max(10, Math.floor(this.outputEl.clientHeight / 18));
        this.ws.sendResize(cols, rows);
    }
}

document.addEventListener('DOMContentLoaded', () => {
    new TerminalApp();
});
