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

        // Sidebar y vistas de navegación
        this.sidebarEl = document.getElementById('app-sidebar');
        this.closeSidebarBtn = document.getElementById('close-sidebar-btn');
        this.openConfigModalBtn = document.getElementById('open-config-modal');
        this.sideHost = document.getElementById('side-host');
        this.sidePort = document.getElementById('side-port');
        this.sideUser = document.getElementById('side-user');
        this.sideShell = document.getElementById('side-shell');
        this.sideViews = Array.from(document.querySelectorAll('.sidebar-view'));
        this.sideLinks = Array.from(document.querySelectorAll('.sidebar-link'));

        // VM elements
        this.vmListEl = document.getElementById('vm-list');
        this.vmFormWrap = document.getElementById('vm-form-wrap');
        this.vmFormLoader = document.getElementById('vm-form-loader');
        this.vmFormFeedback = document.getElementById('vm-form-feedback');
        this.openCreateVmBtn = document.getElementById('open-create-vm');
        this.cancelVmFormBtn = document.getElementById('cancel-vm-form');
        this.vmSubmitBtn = document.getElementById('vm-submit');
        this.vmForm = document.getElementById('vm-form');
        this.vmNameInput = document.getElementById('vm-name');
        this.vmISOSelect = document.getElementById('vm-iso');
        this.vmCoresInput = document.getElementById('vm-cores');
        this.vmMemoryInput = document.getElementById('vm-memory');
        this.vmDiskInput = document.getElementById('vm-disk');
        this.vmCreateInProgress = false;

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
        this.loadISOs();
        this.loadVirtualMachines();
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
                this.updateSidebarConfig(this.currentConfig);
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

        // Foco inteligente en terminal (sin interferir con modales, cabecera ni el panel lateral)
        document.addEventListener('click', (e) => {
            if (this.isModalOpen() ||
                e.target.closest('#config-modal') ||
                e.target.closest('#header') ||
                e.target.closest('#app-sidebar')) {
                return;
            }
            const selection = window.getSelection();
            if (!selection || selection.isCollapsed) {
                this.inputEl.focus();
            }
        });

        // Apertura del menu lateral de navegacion
        this.menuBtn.addEventListener('click', () => {
            this.toggleSidebar();
        });

        this.closeSidebarBtn.addEventListener('click', () => {
            this.toggleSidebar(false);
        });

        this.sideLinks.forEach((link) => {
            link.addEventListener('click', () => {
                const target = link.dataset.menu;
                this.switchSidebarView(target);
            });
        });

        this.openConfigModalBtn.addEventListener('click', () => {
            this.toggleSidebar(false);
            this.openModal();
        });

        this.openCreateVmBtn.addEventListener('click', () => {
            this.hideVMFeedback();
            this.vmFormWrap.classList.toggle('hidden');
        });

        this.cancelVmFormBtn.addEventListener('click', () => {
            this.vmFormWrap.classList.add('hidden');
            this.vmForm.reset();
            this.hideVMFeedback();
            this.setCreateLoading(false);
        });

        this.vmForm.addEventListener('submit', (e) => this.handleVMCreate(e));

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

    toggleSidebar(forceOpen) {
        if (!this.sidebarEl) return;
        if (typeof forceOpen === 'boolean') {
            this.sidebarEl.classList.toggle('hidden', !forceOpen);
            return;
        }
        this.sidebarEl.classList.toggle('hidden');
    }

    switchSidebarView(view) {
        this.sideLinks.forEach((l) => l.classList.toggle('active', l.dataset.menu === view));
        this.sideViews.forEach((s) => s.classList.toggle('active', s.id === `sidebar-${view}-view`));
    }

    updateSidebarConfig(cfg) {
        if (!cfg) return;
        if (this.sideHost) this.sideHost.textContent = cfg.sshHost || 'Sin host';
        if (this.sidePort) this.sidePort.textContent = String(cfg.sshPort || 22);
        if (this.sideUser) this.sideUser.textContent = cfg.sshUser || 'Sin usuario';
        if (this.sideShell) this.sideShell.textContent = (cfg.sshShell || 'powershell').toLowerCase();
    }

    async loadISOs() {
        try {
            const res = await fetch('/api/isos');
            if (!res.ok) return;
            const data = await res.json();
            const isos = data.isos || [];
            this.vmISOSelect.innerHTML = isos.map((iso) => `<option value="${iso}">${iso}</option>`).join('');
        } catch (e) {
            console.warn('No se pudieron cargar los ISOs:', e);
        }
    }

    async loadVirtualMachines() {
        try {
            const res = await fetch('/api/vms');
            if (!res.ok) return;
            const vms = await res.json();
            this.renderVMList(vms);
        } catch (e) {
            console.warn('No se pudieron cargar las máquinas virtuales:', e);
        }
    }

    renderVMList(vms) {
        if (!this.vmListEl) return;
        if (!Array.isArray(vms) || vms.length === 0) {
            this.vmListEl.innerHTML = `<div class="empty-vm-list">No hay máquinas virtuales</div>`;
            return;
        }

        this.vmListEl.innerHTML = vms.map((vm) => `
            <div class="vm-card">
                <div class="vm-card-header">
                    <span class="vm-name">${vm.name}</span>
                    <span class="vm-os">${vm.os}</span>
                </div>
                <div class="vm-card-details">
                    <span>ISO: ${vm.iso}</span>
                    <span>CPU: ${vm.cores}</span>
                    <span>RAM: ${vm.memoryMB} MB</span>
                    <span>Disco: ${vm.diskGB} GB</span>
                </div>
                <div class="vm-card-actions">
                    <button class="btn btn-secondary view-vm" data-id="${vm.id}">Ver</button>
                    <button class="btn btn-danger delete-vm" data-id="${vm.id}">Eliminar</button>
                </div>
            </div>
        `).join('');

        this.vmListEl.querySelectorAll('[data-id]').forEach((btn) => {
            btn.addEventListener('click', async (event) => {
                const id = event.target.dataset.id;
                if (event.target.classList.contains('delete-vm')) {
                    await this.deleteVirtualMachine(id);
                    return;
                }
                await this.viewVirtualMachine(id);
            });
        });
    }

    async viewVirtualMachine(id) {
        try {
            const res = await fetch(`/api/vms/${id}`);
            if (!res.ok) return;
            const vm = await res.json();
            this.showModalFeedback(`VM ${vm.name}: SO ${vm.os}, ISO ${vm.iso}, CPU ${vm.cores}, RAM ${vm.memoryMB} MB, Disco ${vm.diskGB} GB`, 'success');
        } catch (e) {
            console.warn('No se pudo ver la VM:', e);
        }
    }

    async deleteVirtualMachine(id) {
        try {
            const res = await fetch(`/api/vms/${id}`, { method: 'DELETE' });
            if (!res.ok) return;
            this.loadVirtualMachines();
        } catch (e) {
            console.warn('No se pudo eliminar la VM:', e);
        }
    }

    setCreateLoading(isBusy) {
        this.vmCreateInProgress = isBusy;
        this.vmFormWrap.classList.toggle('creating', isBusy);
        this.vmFormLoader.classList.toggle('hidden', !isBusy);
        this.vmSubmitBtn.disabled = isBusy;
        this.cancelVmFormBtn.disabled = isBusy;
        Array.from(this.vmForm.elements).forEach((el) => {
            if (el !== this.vmSubmitBtn && el !== this.cancelVmFormBtn) {
                el.disabled = isBusy;
            }
        });
    }

    showVMCreateFeedback(text, type = 'error') {
        if (!this.vmFormFeedback) return;
        this.vmFormFeedback.textContent = text;
        this.vmFormFeedback.className = `vm-form-feedback ${type}`;
        this.vmFormFeedback.classList.remove('hidden');
    }

    hideVMFeedback() {
        if (!this.vmFormFeedback) return;
        this.vmFormFeedback.textContent = '';
        this.vmFormFeedback.className = 'vm-form-feedback hidden';
    }

    parseApiError(text) {
        try {
            const payload = JSON.parse(text);
            if (payload && payload.error) return payload.error;
        } catch (_) {
            // El backend puede responder con una cadena libre.
        }
        return text || 'No se pudo crear la máquina virtual.';
    }

    async handleVMCreate(e) {
        e.preventDefault();
        if (this.vmCreateInProgress) return;

        const payload = {
            name: this.vmNameInput.value.trim(),
            iso: this.vmISOSelect.value,
            cores: parseInt(this.vmCoresInput.value, 10) || 1,
            memoryMB: parseInt(this.vmMemoryInput.value, 10) || 1024,
            diskGB: parseInt(this.vmDiskInput.value, 10) || 10,
        };

        if (!payload.name) {
            this.showVMCreateFeedback('El nombre de la VM es obligatorio.', 'error');
            this.vmNameInput.focus();
            return;
        }

        if (!payload.iso) {
            this.showVMCreateFeedback('Debe elegir una ISO.', 'error');
            this.vmISOSelect.focus();
            return;
        }

        this.hideVMFeedback();
        this.setCreateLoading(true);

        try {
            const res = await fetch('/api/vms', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(payload),
            });
            if (!res.ok) {
                const text = await res.text();
                this.showVMCreateFeedback(this.parseApiError(text), 'error');
                return;
            }

            const vm = await res.json();
            this.vmForm.reset();
            this.hideVMFeedback();
            this.loadVirtualMachines();
            this.showVMCreateFeedback(`VM creada: ${vm.name || payload.name}`, 'success');
            this.vmFormWrap.classList.remove('hidden');
            this.vmFormWrap.classList.remove('creating');
            this.vmFormLoader.classList.add('hidden');
        } catch (err) {
            console.warn('No se pudo crear la máquina virtual:', err);
            this.showVMCreateFeedback(err && err.message ? err.message : 'No se pudo crear la máquina virtual.', 'error');
        } finally {
            this.setCreateLoading(false);
        }
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
        this.updateSidebarConfig(this.currentConfig);
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
        const isBrokenCSI = tail.startsWith('\x1b[') && !/\x1b\[[0-?]*[ -/]*[@-~]$/.test(tail);
        const isBrokenOSC = tail.startsWith('\x1b]') && !/(?:\x07|\x1b\\)$/.test(tail);

        if (isBrokenCSI || isBrokenOSC) {
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

        const processed = this.removeBackspaces(AnsiParser.sanitizeText(text));
        const normalized = processed.replace(/\r\n/g, '\n').replace(/\r/g, '\n');
        const lines = normalized.split('\n');

        for (const rawLine of lines) {
            const line = rawLine.replace(/\s+$/g, '');
            const plain = line.trim();
            if (!plain) {
                continue;
            }

            // Filtra el ruido de la PTY de PowerShell: líneas de prompt, “P”, “>>”
            // y el artefacto “PPS C:\...>” que se empalma con el texto del comando.
            if (plain === 'P' || plain === '>>' || plain === '>' || /^PPS\s+[A-Za-z]:\\/.test(plain)) {
                continue;
            }

            if (/^PS\s+[A-Za-z]:\\[^>]*>\s*$/.test(plain)) {
                const path = AnsiParser.extractPath(line);
                if (path) {
                    this.updateCurrentPath(path);
                }
                continue;
            }

            if (/^PS\s+[A-Za-z]:\\[^>]*>\s+/.test(plain)) {
                continue;
            }

            // Render line-by-line; avoid nested run of the same prompt fragments.
            this.appendToCurrentLine(line);
            this.finishLine();
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

        const result = AnsiParser.renderToHTML(this.pendingText, []);
        this.activeClasses = result.classes;
        this.currentLineEl.innerHTML = result.html;
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
