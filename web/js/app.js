// Gestor de la interfaz de terminal web, menú lateral, gestión de VMs, Llaves SSH y conexión WebSocket
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
        this.sidebarBackdrop = document.getElementById('sidebar-backdrop');
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
        this.vmFormFeedback = document.getElementById('vm-form-feedback');
        this.openCreateVmBtn = document.getElementById('open-create-vm');
        this.closeVmFormBtn = document.getElementById('close-vm-form-btn');
        this.cancelVmFormBtn = document.getElementById('cancel-vm-form');
        this.refreshVmsBtn = document.getElementById('refresh-vms-btn');
        this.vmSubmitBtn = document.getElementById('vm-submit');
        this.vmForm = document.getElementById('vm-form');
        this.vmNameInput = document.getElementById('vm-name');
        this.vmOSSelect = document.getElementById('vm-os');
        this.vmISOSelect = document.getElementById('vm-iso');
        this.vmCoresInput = document.getElementById('vm-cores');
        this.vmMemoryInput = document.getElementById('vm-memory');
        this.vmDiskInput = document.getElementById('vm-disk');
        this.rawISOItems = [];

        // VM details modal
        this.vmDetailsModal = document.getElementById('vm-details-modal');
        this.closeVmDetailsModalBtn = document.getElementById('close-vm-details-modal');
        this.btnCloseVmDetails = document.getElementById('btn-close-vm-details');
        this.vmDetailsContent = document.getElementById('vm-details-content');

        // VM delete confirmation modal
        this.deleteVmModal = document.getElementById('delete-vm-modal');
        this.closeDeleteVmModalBtn = document.getElementById('close-delete-vm-modal');
        this.cancelDeleteVmBtn = document.getElementById('cancel-delete-vm');
        this.confirmDeleteVmBtn = document.getElementById('confirm-delete-vm');
        this.deleteVmNameEl = document.getElementById('delete-vm-name');
        this.deleteVmFeedback = document.getElementById('delete-vm-feedback');
        this.pendingDeleteVmName = null;

        // Keys elements
        this.keysListEl = document.getElementById('keys-list');
        this.keysCountPill = document.getElementById('keys-count-pill');
        this.refreshKeysBtn = document.getElementById('refresh-keys-btn');
        this.deleteKeyModal = document.getElementById('delete-key-modal');
        this.closeDeleteKeyModalBtn = document.getElementById('close-delete-key-modal');
        this.cancelDeleteKeyBtn = document.getElementById('cancel-delete-key');
        this.confirmDeleteKeyBtn = document.getElementById('confirm-delete-key');
        this.deleteKeyNameEl = document.getElementById('delete-key-name');
        this.deleteKeyFeedback = document.getElementById('delete-key-feedback');
        this.pendingDeleteKeyId = null;

        // Elementos del modal de configuracion
        this.modalEl = document.getElementById('config-modal');
        this.closeModalBtn = document.getElementById('close-modal-btn');
        this.cancelModalBtn = document.getElementById('btn-cancel-modal');
        this.resetDefaultsBtn = document.getElementById('btn-reset-defaults');
        this.configForm = document.getElementById('config-form');
        this.modalFeedback = document.getElementById('modal-feedback');

        // Inputs del formulario de configuracion
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
        this.currentPath = '~';
        this.isClearingScreen = false;
        this.ignoreEchoCommand = null;

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
        this.loadKeys();
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

    updateSidebarConfig(cfg) {
        if (!cfg) return;
        if (this.sideHost) this.sideHost.textContent = cfg.sshHost || 'Sin host';
        if (this.sidePort) this.sidePort.textContent = String(cfg.sshPort || 22);
        if (this.sideUser) this.sideUser.textContent = cfg.sshUser || 'Sin usuario';
        if (this.sideShell) this.sideShell.textContent = (cfg.sshShell || 'powershell').toLowerCase();
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

        if (this.currentConfig.sshPassword) {
            params.password = this.currentConfig.sshPassword;
        }

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

        // Foco inteligente en terminal
        document.addEventListener('click', (e) => {
            if (this.isModalOpen() || this.isSidebarOpen() || e.target.closest('.modal-dialog') || e.target.closest('#app-sidebar') || e.target.closest('#header')) {
                return;
            }
            const selection = window.getSelection();
            if (!selection || selection.isCollapsed) {
                this.inputEl.focus();
            }
        });

        // Apertura y cierre del menú lateral (3 líneas)
        this.menuBtn.addEventListener('click', () => {
            this.toggleSidebar();
        });

        this.closeSidebarBtn.addEventListener('click', () => {
            this.toggleSidebar(false);
        });

        if (this.sidebarBackdrop) {
            this.sidebarBackdrop.addEventListener('click', () => {
                this.toggleSidebar(false);
            });
        }

        // Pestañas de navegación del sidebar (config, vms, keys)
        this.sideLinks.forEach((link) => {
            link.addEventListener('click', () => {
                const target = link.dataset.menu;
                this.switchSidebarView(target);
            });
        });

        // Abrir modal de configuración desde el sidebar
        this.openConfigModalBtn.addEventListener('click', () => {
            this.toggleSidebar(false);
            this.openModal();
        });

        // Control del formulario de creación de VM
        this.openCreateVmBtn.addEventListener('click', () => {
            this.hideVMFeedback();
            this.vmFormWrap.classList.toggle('hidden');
            if (!this.vmFormWrap.classList.contains('hidden')) {
                this.filterISOsBySelectedOS();
                setTimeout(() => this.vmNameInput.focus(), 50);
            }
        });

        if (this.closeVmFormBtn) {
            this.closeVmFormBtn.addEventListener('click', () => {
                this.vmFormWrap.classList.add('hidden');
                this.vmForm.reset();
                this.hideVMFeedback();
            });
        }

        this.cancelVmFormBtn.addEventListener('click', () => {
            this.vmFormWrap.classList.add('hidden');
            this.vmForm.reset();
            this.hideVMFeedback();
        });

        this.refreshVmsBtn.addEventListener('click', () => {
            this.loadVirtualMachines();
        });

        if (this.refreshKeysBtn) {
            this.refreshKeysBtn.addEventListener('click', () => {
                this.loadKeys();
            });
        }

        // Cambio de SO en el formulario -> filtrar ISOs en tiempo real
        if (this.vmOSSelect) {
            this.vmOSSelect.addEventListener('change', () => {
                this.filterISOsBySelectedOS();
            });
        }

        // Envío de formulario de creación de VM
        this.vmForm.addEventListener('submit', (e) => this.handleVMCreate(e));

        // Modales de eliminación de VM
        this.closeDeleteVmModalBtn.addEventListener('click', () => this.closeDeleteModal());
        this.cancelDeleteVmBtn.addEventListener('click', () => this.closeDeleteModal());
        this.confirmDeleteVmBtn.addEventListener('click', () => {
            if (!this.pendingDeleteVmName) return;
            this.executeDeleteVirtualMachine(this.pendingDeleteVmName);
        });

        // Modales de eliminación de Llave
        if (this.closeDeleteKeyModalBtn) {
            this.closeDeleteKeyModalBtn.addEventListener('click', () => this.closeDeleteKeyModal());
        }
        if (this.cancelDeleteKeyBtn) {
            this.cancelDeleteKeyBtn.addEventListener('click', () => this.closeDeleteKeyModal());
        }
        if (this.confirmDeleteKeyBtn) {
            this.confirmDeleteKeyBtn.addEventListener('click', () => {
                if (!this.pendingDeleteKeyId) return;
                this.executeDeleteKey(this.pendingDeleteKeyId);
            });
        }

        // Modales de detalles de VM
        if (this.closeVmDetailsModalBtn) {
            this.closeVmDetailsModalBtn.addEventListener('click', () => this.closeDetailsModal());
        }
        if (this.btnCloseVmDetails) {
            this.btnCloseVmDetails.addEventListener('click', () => this.closeDetailsModal());
        }

        // Boton de reconexion directa
        this.reconnectBtn.addEventListener('click', () => {
            this.clearOutput();
            this.appendInfoLine('[Reconectando sesión SSH...]');
            this.connectCurrent();
        });

        // Cierre del modal de configuración
        this.closeModalBtn.addEventListener('click', () => this.closeModal());
        this.cancelModalBtn.addEventListener('click', () => this.closeModal());

        // Cierre con tecla Escape
        window.addEventListener('keydown', (e) => {
            if (e.key === 'Escape') {
                if (this.isModalOpen()) {
                    this.closeModal();
                    this.closeDeleteModal();
                    this.closeDeleteKeyModal();
                    this.closeDetailsModal();
                } else if (this.isSidebarOpen()) {
                    this.toggleSidebar(false);
                }
            }
        });

        // Cierre al hacer clic fuera de modales
        this.modalEl.addEventListener('click', (e) => {
            if (e.target === this.modalEl) this.closeModal();
        });
        if (this.deleteVmModal) {
            this.deleteVmModal.addEventListener('click', (e) => {
                if (e.target === this.deleteVmModal) this.closeDeleteModal();
            });
        }
        if (this.deleteKeyModal) {
            this.deleteKeyModal.addEventListener('click', (e) => {
                if (e.target === this.deleteKeyModal) this.closeDeleteKeyModal();
            });
        }
        if (this.vmDetailsModal) {
            this.vmDetailsModal.addEventListener('click', (e) => {
                if (e.target === this.vmDetailsModal) this.closeDetailsModal();
            });
        }

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

    isSidebarOpen() {
        return this.sidebarEl && !this.sidebarEl.classList.contains('hidden');
    }

    toggleSidebar(forceOpen) {
        if (!this.sidebarEl) return;
        const shouldOpen = typeof forceOpen === 'boolean' ? forceOpen : this.sidebarEl.classList.contains('hidden');
        this.sidebarEl.classList.toggle('hidden', !shouldOpen);
        if (this.sidebarBackdrop) {
            this.sidebarBackdrop.classList.toggle('hidden', !shouldOpen);
        }
        if (shouldOpen) {
            this.loadISOs();
            this.loadVirtualMachines();
            this.loadKeys();
        }
    }

    switchSidebarView(view) {
        this.sideLinks.forEach((l) => l.classList.toggle('active', l.dataset.menu === view));
        this.sideViews.forEach((s) => s.classList.toggle('active', s.id === `sidebar-${view}-view`));
        if (view === 'keys') {
            this.loadKeys();
        } else if (view === 'vms') {
            this.loadVirtualMachines();
        }
    }

    // =========================================================================
    // Filtrado de ISOs según el SO seleccionado
    // =========================================================================
    async loadISOs() {
        try {
            const res = await fetch('/api/isos');
            if (!res.ok) return;
            const data = await res.json();
            this.rawISOItems = data.items || [];
            this.filterISOsBySelectedOS();
        } catch (e) {
            console.warn('No se pudieron cargar los ISOs:', e);
        }
    }

    filterISOsBySelectedOS() {
        if (!this.vmISOSelect || !this.vmOSSelect) return;
        const selectedOS = this.vmOSSelect.value || 'Ubuntu 24.04 LTS';
        const osLower = selectedOS.toLowerCase();

        let filtered = this.rawISOItems.filter((item) => {
            const itemOS = (item.os || '').toLowerCase();
            const itemName = (item.isoFilename || item.name || '').toLowerCase();

            if (osLower.includes('ubuntu')) return itemOS.includes('ubuntu') || itemName.includes('ubuntu');
            if (osLower.includes('debian')) return itemOS.includes('debian') || itemName.includes('debian');
            if (osLower.includes('windows 11')) return itemName.includes('win11') || itemOS.includes('11');
            if (osLower.includes('windows 10')) return itemName.includes('win10') || itemOS.includes('10');
            if (osLower.includes('server')) return itemName.includes('server') || itemOS.includes('server');
            if (osLower.includes('centos') || osLower.includes('redhat')) return itemOS.includes('centos') || itemName.includes('centos') || itemName.includes('rhel');
            if (osLower.includes('alpine')) return itemOS.includes('alpine') || itemName.includes('alpine');
            if (osLower.includes('arch')) return itemOS.includes('arch') || itemName.includes('arch');

            return true;
        });

        if (filtered.length === 0) {
            filtered = this.rawISOItems;
        }

        this.vmISOSelect.innerHTML = filtered.map((item) => {
            const filename = item.isoFilename || item.name;
            return `<option value="${filename}">${filename}</option>`;
        }).join('');
    }

    // =========================================================================
    // Gestión de Máquinas Virtuales (Listado, Creación, Detalles, Eliminación)
    // =========================================================================
    async loadVirtualMachines() {
        if (!this.vmListEl) return;
        this.vmListEl.innerHTML = `<div class="empty-vm-list"><span class="btn-spinner"></span> Cargando máquinas virtuales...</div>`;
        try {
            const res = await fetch('/api/vms');
            if (!res.ok) {
                const text = await res.text();
                this.vmListEl.innerHTML = `<div class="empty-vm-list" style="color:var(--accent-red)">Error al consultar VMs: ${this.parseApiError(text)}</div>`;
                return;
            }
            const vms = await res.json();
            this.renderVMList(vms);
        } catch (e) {
            console.warn('No se pudieron cargar las máquinas virtuales:', e);
            this.vmListEl.innerHTML = `<div class="empty-vm-list" style="color:var(--accent-red)">No se pudo conectar al servidor para listar VMs.</div>`;
        }
    }

    renderVMList(vms) {
        if (!this.vmListEl) return;
        if (!Array.isArray(vms) || vms.length === 0) {
            this.vmListEl.innerHTML = `<div class="empty-vm-list">No hay máquinas virtuales creadas o registradas.</div>`;
            return;
        }

        this.vmListEl.innerHTML = vms.map((vm) => `
            <div class="vm-card" id="vm-card-${this.escapeAttr(vm.name)}">
                <div class="vm-card-header">
                    <span class="vm-name">${this.escapeHTML(vm.name)}</span>
                    <span class="vm-os">${this.escapeHTML(vm.os || 'Linux')}</span>
                </div>
                <div class="vm-card-details">
                    <span><strong>CPU:</strong> ${vm.cores || 2} Cores</span>
                    <span><strong>RAM:</strong> ${vm.memoryMB || 2048} MB</span>
                    <span><strong>Disco:</strong> ${vm.diskGB || 40} GB</span>
                    <span><strong>Estado:</strong> ${this.escapeHTML(vm.state || 'Creada')}</span>
                </div>
                <div class="vm-card-delete-loader hidden">
                    <span class="vm-card-spinner"></span>
                    <span>Eliminando máquina virtual...</span>
                </div>
                <div class="vm-card-actions">
                    <button class="btn btn-secondary view-vm" data-id="${this.escapeAttr(vm.name)}">Ver</button>
                    <button class="btn btn-danger delete-vm" data-id="${this.escapeAttr(vm.name)}">Eliminar</button>
                </div>
            </div>
        `).join('');

        this.vmListEl.querySelectorAll('.view-vm').forEach((btn) => {
            btn.addEventListener('click', () => {
                const id = btn.dataset.id;
                this.viewVirtualMachine(id);
            });
        });

        this.vmListEl.querySelectorAll('.delete-vm').forEach((btn) => {
            btn.addEventListener('click', () => {
                const id = btn.dataset.id;
                this.openDeleteVMModal(id);
            });
        });
    }

    async viewVirtualMachine(id) {
        if (!this.vmDetailsModal || !this.vmDetailsContent) return;
        this.vmDetailsContent.innerHTML = `<div style="grid-column:1/-1;text-align:center;padding:16px"><span class="btn-spinner"></span> Cargando detalles...</div>`;
        this.vmDetailsModal.classList.remove('hidden');
        this.vmDetailsModal.setAttribute('aria-hidden', 'false');

        try {
            const res = await fetch(`/api/vms/${encodeURIComponent(id)}`);
            if (!res.ok) {
                const text = await res.text();
                this.vmDetailsContent.innerHTML = `<div style="grid-column:1/-1;color:var(--accent-red)">Error al consultar VM: ${this.parseApiError(text)}</div>`;
                return;
            }
            const vm = await res.json();

            const cores = vm.cores > 0 ? vm.cores : 2;
            const memoryMB = vm.memoryMB > 0 ? vm.memoryMB : 2048;
            const diskGB = vm.diskGB > 0 ? vm.diskGB : 40;
            const os = vm.os && vm.os !== '' ? vm.os : 'Linux';

            this.vmDetailsContent.innerHTML = `
                <div class="vm-detail-item">
                    <span class="vm-detail-key">Nombre de la MV</span>
                    <span class="vm-detail-val">${this.escapeHTML(vm.name)}</span>
                </div>
                <div class="vm-detail-item">
                    <span class="vm-detail-key">Sistema Operativo</span>
                    <span class="vm-detail-val" style="color:var(--accent-green)">${this.escapeHTML(os)}</span>
                </div>
                <div class="vm-detail-item">
                    <span class="vm-detail-key">CPU (Cores)</span>
                    <span class="vm-detail-val">${cores} vCPU</span>
                </div>
                <div class="vm-detail-item">
                    <span class="vm-detail-key">Memoria RAM</span>
                    <span class="vm-detail-val">${memoryMB} MB (${(memoryMB/1024).toFixed(1)} GB)</span>
                </div>
                <div class="vm-detail-item">
                    <span class="vm-detail-key">Almacenamiento (Disco)</span>
                    <span class="vm-detail-val">${diskGB} GB</span>
                </div>
                <div class="vm-detail-item">
                    <span class="vm-detail-key">Estado Actual</span>
                    <span class="vm-detail-val" style="color:var(--accent)">${this.escapeHTML(vm.state || 'Creada')}</span>
                </div>
                ${vm.iso ? `
                <div class="vm-detail-item full-width">
                    <span class="vm-detail-key">Imagen ISO</span>
                    <span class="vm-detail-val">${this.escapeHTML(vm.iso)}</span>
                </div>` : ''}
                ${vm.keyName || vm.keyId ? `
                <div class="vm-detail-item full-width">
                    <span class="vm-detail-key">Llave SSH Asignada</span>
                    <span class="vm-detail-val" style="color:var(--accent-yellow)">🔑 ${this.escapeHTML(vm.keyName || vm.keyId)}</span>
                </div>` : ''}
                ${vm.network ? `
                <div class="vm-detail-item full-width">
                    <span class="vm-detail-key">Adaptador de Red</span>
                    <span class="vm-detail-val">${this.escapeHTML(vm.network)}</span>
                </div>` : ''}
                ${vm.diskPath ? `
                <div class="vm-detail-item full-width">
                    <span class="vm-detail-key">Ruta del Disco Virtual</span>
                    <span class="vm-detail-val">${this.escapeHTML(vm.diskPath)}</span>
                </div>` : ''}
            `;
        } catch (e) {
            console.warn('Error al ver VM:', e);
            this.vmDetailsContent.innerHTML = `<div style="grid-column:1/-1;color:var(--accent-red)">No se pudo cargar la información de la máquina virtual.</div>`;
        }
    }

    closeDetailsModal() {
        if (!this.vmDetailsModal) return;
        this.vmDetailsModal.classList.add('hidden');
        this.vmDetailsModal.setAttribute('aria-hidden', 'true');
    }

    async handleVMCreate(e) {
        e.preventDefault();

        const name = this.vmNameInput.value.trim();
        const os = this.vmOSSelect.value || 'Ubuntu 24.04 LTS';
        const iso = this.vmISOSelect.value || '';
        const cores = parseInt(this.vmCoresInput.value, 10) || 2;
        const memoryMB = parseInt(this.vmMemoryInput.value, 10) || 2048;
        const diskGB = parseInt(this.vmDiskInput.value, 10) || 40;

        if (!name) {
            this.showVMCreateFeedback('El nombre de la máquina es obligatorio.', 'error');
            this.vmNameInput.focus();
            return;
        }

        if (!iso) {
            this.showVMCreateFeedback('Debes seleccionar una imagen ISO.', 'error');
            return;
        }

        this.setCreateVMLoading(true);
        this.hideVMFeedback();

        try {
            const payload = { name, os, iso, cores, memoryMB, diskGB };
            const res = await fetch('/api/vms', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(payload),
            });

            if (!res.ok) {
                const text = await res.text();
                const errMsg = this.parseApiError(text);
                this.showVMCreateFeedback(`Error al crear VM: ${errMsg}`, 'error');
                this.setCreateVMLoading(false);
                return;
            }

            this.showVMCreateFeedback(`¡Máquina virtual '${name}' y su llave SSH creadas exitosamente!`, 'success');
            this.appendInfoLine(`[Crear VM] Máquina virtual '${name}' creada exitosamente con par de llaves SSH.`);

            setTimeout(() => {
                this.vmFormWrap.classList.add('hidden');
                this.vmForm.reset();
                this.hideVMFeedback();
                this.setCreateVMLoading(false);
            }, 1200);

            await this.loadVirtualMachines();
            await this.loadKeys();
        } catch (err) {
            console.warn('Error al crear VM:', err);
            this.showVMCreateFeedback('No se pudo conectar al servidor para crear la MV.', 'error');
            this.setCreateVMLoading(false);
        }
    }

    setCreateVMLoading(loading) {
        if (!this.vmSubmitBtn) return;
        const spinner = this.vmSubmitBtn.querySelector('.btn-spinner');
        const text = this.vmSubmitBtn.querySelector('.btn-text');

        this.vmSubmitBtn.disabled = loading;
        if (this.cancelVmFormBtn) this.cancelVmFormBtn.disabled = loading;

        if (spinner) spinner.classList.toggle('hidden', !loading);
        if (text) text.textContent = loading ? 'Creando Máquina...' : 'Crear Máquina';
    }

    showVMCreateFeedback(text, type = 'error') {
        if (!this.vmFormFeedback) return;
        this.vmFormFeedback.textContent = text;
        this.vmFormFeedback.className = `vm-form-feedback ${type}`;
    }

    hideVMFeedback() {
        if (!this.vmFormFeedback) return;
        this.vmFormFeedback.textContent = '';
        this.vmFormFeedback.className = 'vm-form-feedback hidden';
    }

    openDeleteVMModal(name) {
        this.pendingDeleteVmName = name;
        if (this.deleteVmNameEl) {
            this.deleteVmNameEl.textContent = name;
        }
        if (this.deleteVmFeedback) {
            this.deleteVmFeedback.textContent = '';
            this.deleteVmFeedback.className = 'modal-feedback hidden';
        }
        this.setDeleteModalLoading(false);

        if (!this.deleteVmModal) return;
        this.deleteVmModal.classList.remove('hidden');
        this.deleteVmModal.setAttribute('aria-hidden', 'false');
    }

    closeDeleteModal() {
        if (!this.deleteVmModal) return;
        this.deleteVmModal.classList.add('hidden');
        this.deleteVmModal.setAttribute('aria-hidden', 'true');
        this.pendingDeleteVmName = null;
        this.setDeleteModalLoading(false);
    }

    setDeleteModalLoading(loading) {
        if (!this.confirmDeleteVmBtn) return;
        const spinner = this.confirmDeleteVmBtn.querySelector('.btn-spinner');
        const text = this.confirmDeleteVmBtn.querySelector('.btn-text');

        this.confirmDeleteVmBtn.disabled = loading;
        if (this.cancelDeleteVmBtn) this.cancelDeleteVmBtn.disabled = loading;
        if (this.closeDeleteVmModalBtn) this.closeDeleteVmModalBtn.disabled = loading;

        if (spinner) spinner.classList.toggle('hidden', !loading);
        if (text) text.textContent = loading ? 'Eliminando...' : 'Sí, eliminar';
    }

    async executeDeleteVirtualMachine(name) {
        this.setDeleteModalLoading(true);

        const card = document.getElementById(`vm-card-${name}`);
        if (card) {
            card.classList.add('vm-card-deleting');
            const cardLoader = card.querySelector('.vm-card-delete-loader');
            if (cardLoader) cardLoader.classList.remove('hidden');
            card.querySelectorAll('button').forEach((b) => b.disabled = true);
        }

        try {
            const res = await fetch(`/api/vms/${encodeURIComponent(name)}`, {
                method: 'DELETE'
            });

            if (!res.ok) {
                const text = await res.text();
                const errMsg = this.parseApiError(text);
                if (this.deleteVmFeedback) {
                    this.deleteVmFeedback.textContent = `Error: ${errMsg}`;
                    this.deleteVmFeedback.className = 'modal-feedback error';
                }
                this.appendErrorLine(`[Eliminar VM] Error al eliminar '${name}': ${errMsg}`);
                this.setDeleteModalLoading(false);
                if (card) {
                    card.classList.remove('vm-card-deleting');
                    const cardLoader = card.querySelector('.vm-card-delete-loader');
                    if (cardLoader) cardLoader.classList.add('hidden');
                    card.querySelectorAll('button').forEach((b) => b.disabled = false);
                }
                return;
            }

            this.appendInfoLine(`[Eliminar VM] '${name}' eliminada correctamente vía SSH.`);
            this.closeDeleteModal();
            await this.loadVirtualMachines();
            await this.loadKeys();
        } catch (e) {
            console.warn('Error al eliminar VM:', e);
            if (this.deleteVmFeedback) {
                this.deleteVmFeedback.textContent = 'No se pudo conectar al servidor para eliminar la MV.';
                this.deleteVmFeedback.className = 'modal-feedback error';
            }
            this.appendErrorLine(`[Eliminar VM] No se pudo eliminar la máquina virtual '${name}'.`);
            this.setDeleteModalLoading(false);
            if (card) {
                card.classList.remove('vm-card-deleting');
                const cardLoader = card.querySelector('.vm-card-delete-loader');
                if (cardLoader) cardLoader.classList.add('hidden');
                card.querySelectorAll('button').forEach((b) => b.disabled = false);
            }
        }
    }

    // =========================================================================
    // Módulo de Llaves SSH (Listado, Descarga de Pública/Privada, Eliminación)
    // =========================================================================
    async loadKeys() {
        if (!this.keysListEl) return;
        this.keysListEl.innerHTML = `<div class="empty-vm-list"><span class="btn-spinner"></span> Cargando llaves SSH...</div>`;

        try {
            const res = await fetch('/api/keys');
            if (!res.ok) {
                this.keysListEl.innerHTML = `<div class="empty-vm-list" style="color:var(--accent-red)">Error al consultar llaves.</div>`;
                return;
            }
            const keys = await res.json();
            this.renderKeysList(keys);
        } catch (e) {
            console.warn('Error al cargar llaves:', e);
            this.keysListEl.innerHTML = `<div class="empty-vm-list" style="color:var(--accent-red)">No se pudo conectar para listar las llaves.</div>`;
        }
    }

    renderKeysList(keys) {
        if (!this.keysListEl) return;
        const count = Array.isArray(keys) ? keys.length : 0;
        if (this.keysCountPill) {
            this.keysCountPill.textContent = `${count} ${count === 1 ? 'Llave' : 'Llaves'}`;
        }

        if (!Array.isArray(keys) || keys.length === 0) {
            this.keysListEl.innerHTML = `<div class="empty-vm-list">No hay llaves SSH generadas. Al crear una máquina virtual, se generará su llave automáticamente.</div>`;
            return;
        }

        this.keysListEl.innerHTML = keys.map((key) => `
            <div class="key-card" id="key-card-${this.escapeAttr(key.id)}">
                <div class="key-card-header">
                    <span class="key-name">
                        <svg viewBox="0 0 24 24" width="14" height="14" stroke="currentColor" stroke-width="2" fill="none">
                            <path d="M21 2l-2 2m-7.61 7.61a5.5 5.5 0 1 1-7.78 7.78 5.5 5.5 0 0 1 7.78-7.78zm0 0L15.5 7.5m0 0l3 3L22 7l-3-3m-3.5 3.5L19 4"></path>
                        </svg>
                        ${this.escapeHTML(key.name || key.id)}
                    </span>
                    <span class="key-vm-badge" title="Máquina Virtual asociada">
                        🖥️ ${this.escapeHTML(key.vmName || 'General')}
                    </span>
                </div>
                <div class="key-card-details">
                    <span><strong>Algoritmo:</strong> ${this.escapeHTML(key.algorithm || 'RSA 2048')}</span>
                    <span class="key-fingerprint"><strong>Huella:</strong> ${this.escapeHTML(key.fingerprint || '-')}</span>
                </div>
                <div class="key-card-actions">
                    <a href="/api/keys/${encodeURIComponent(key.id)}/private" class="btn btn-primary" download title="Descargar clave privada .pem">
                        ⬇ Privada (.pem)
                    </a>
                    <a href="/api/keys/${encodeURIComponent(key.id)}/public" class="btn btn-secondary" download title="Descargar clave pública .pub">
                        ⬇ Pública (.pub)
                    </a>
                    <button class="btn btn-danger delete-key-btn" data-id="${this.escapeAttr(key.id)}" data-name="${this.escapeAttr(key.name || key.id)}">
                        Eliminar
                    </button>
                </div>
            </div>
        `).join('');

        this.keysListEl.querySelectorAll('.delete-key-btn').forEach((btn) => {
            btn.addEventListener('click', () => {
                const id = btn.dataset.id;
                const name = btn.dataset.name;
                this.openDeleteKeyModal(id, name);
            });
        });
    }

    openDeleteKeyModal(id, name) {
        this.pendingDeleteKeyId = id;
        if (this.deleteKeyNameEl) {
            this.deleteKeyNameEl.textContent = name;
        }
        if (this.deleteKeyFeedback) {
            this.deleteKeyFeedback.textContent = '';
            this.deleteKeyFeedback.className = 'modal-feedback hidden';
        }
        this.setDeleteKeyModalLoading(false);

        if (!this.deleteKeyModal) return;
        this.deleteKeyModal.classList.remove('hidden');
        this.deleteKeyModal.setAttribute('aria-hidden', 'false');
    }

    closeDeleteKeyModal() {
        if (!this.deleteKeyModal) return;
        this.deleteKeyModal.classList.add('hidden');
        this.deleteKeyModal.setAttribute('aria-hidden', 'true');
        this.pendingDeleteKeyId = null;
        this.setDeleteKeyModalLoading(false);
    }

    setDeleteKeyModalLoading(loading) {
        if (!this.confirmDeleteKeyBtn) return;
        const spinner = this.confirmDeleteKeyBtn.querySelector('.btn-spinner');
        const text = this.confirmDeleteKeyBtn.querySelector('.btn-text');

        this.confirmDeleteKeyBtn.disabled = loading;
        if (this.cancelDeleteKeyBtn) this.cancelDeleteKeyBtn.disabled = loading;
        if (this.closeDeleteKeyModalBtn) this.closeDeleteKeyModalBtn.disabled = loading;

        if (spinner) spinner.classList.toggle('hidden', !loading);
        if (text) text.textContent = loading ? 'Eliminando...' : 'Sí, eliminar';
    }

    async executeDeleteKey(id) {
        this.setDeleteKeyModalLoading(true);

        try {
            const res = await fetch(`/api/keys/${encodeURIComponent(id)}`, {
                method: 'DELETE'
            });

            if (!res.ok) {
                const text = await res.text();
                const errMsg = this.parseApiError(text);
                if (this.deleteKeyFeedback) {
                    this.deleteKeyFeedback.textContent = `Error: ${errMsg}`;
                    this.deleteKeyFeedback.className = 'modal-feedback error';
                }
                this.setDeleteKeyModalLoading(false);
                return;
            }

            this.appendInfoLine(`[Llaves SSH] Llave '${id}' eliminada correctamente.`);
            this.closeDeleteKeyModal();
            await this.loadKeys();
        } catch (e) {
            console.warn('Error al eliminar llave:', e);
            if (this.deleteKeyFeedback) {
                this.deleteKeyFeedback.textContent = 'No se pudo conectar al servidor para eliminar la llave.';
                this.deleteKeyFeedback.className = 'modal-feedback error';
            }
            this.setDeleteKeyModalLoading(false);
        }
    }

    // =========================================================================
    // Modal de Configuración
    // =========================================================================
    isModalOpen() {
        const isCfg = this.modalEl && !this.modalEl.classList.contains('hidden');
        const isDel = this.deleteVmModal && !this.deleteVmModal.classList.contains('hidden');
        const isDelKey = this.deleteKeyModal && !this.deleteKeyModal.classList.contains('hidden');
        const isDet = this.vmDetailsModal && !this.vmDetailsModal.classList.contains('hidden');
        return isCfg || isDel || isDelKey || isDet;
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

        const newConfig = {
            sshHost: host,
            sshPort: port,
            sshUser: user,
            sshShell: shell,
            sshStartPath: startPath,
            sshKeyPath: keyPath,
        };

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
        this.updateCurrentPath(startPath || '~');

        this.closeModal();

        this.clearOutput();
        this.appendInfoLine(`[Iniciando sesión SSH a ${user}@${host}:${port} (${shell})...]`);

        this.connectCurrent();
    }

    // =========================================================================
    // Terminal y Manejo de Comandos
    // =========================================================================
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
            this.isClearingScreen = true;
            this.ws.sendInput(cmdTrimmed + '\r\n');
            this.inputEl.value = '';
            this.autoScroll = true;
            return;
        }

        this.autoScroll = true;

        // Si tenemos una línea activa con el prompt en pantalla, completarla con el comando en amarillo
        if (this.currentLineEl) {
            const promptText = AnsiParser.stripAnsi(this.pendingText || this.currentLineEl.textContent || '').trim();
            if (promptText && (promptText.endsWith('>') || promptText.endsWith('$') || promptText.endsWith('#'))) {
                this.currentLineEl.innerHTML = AnsiParser.renderPromptAndCommand(promptText, cmdTrimmed);
                this.finishLine();
            } else {
                this.finishLine();
            }
        }

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
                // No es JSON, continuar como flujo regular
            }
        }

        if (AnsiParser.hasClearScreen(rawData)) {
            this.clearOutput();
            this.isClearingScreen = false;
            rawData = AnsiParser.removeClearScreen(rawData);
            if (!rawData || !rawData.trim()) return;
        }

        if (this.isClearingScreen) {
            const stripped = AnsiParser.stripAnsi(rawData).trim().toLowerCase();
            if (stripped === 'cls' || stripped === 'clear' || stripped === '') {
                return;
            }
            this.isClearingScreen = false;
        }

        const detectedPath = AnsiParser.extractPath(rawData);
        if (detectedPath) {
            this.updateCurrentPath(detectedPath);
        }

        this.processText(rawData);
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
        // Manejar retornos de carro y secuencias de control de borrado de línea (VT100)
        const tokens = processed.split(/(\r\n|\r|\n|\x1b\[2K|\x1b\[1K|\x1b\[0?K|\x1b\[1G)/);

        for (const token of tokens) {
            if (token === '') continue;

            if (token === '\r\n' || token === '\n') {
                this.finishLine();
            } else if (token === '\r' || token === '\x1b[2K' || token === '\x1b[1K' || token === '\x1b[0K' || token === '\x1b[K' || token === '\x1b[1G') {
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
        }

        this.pendingText += text;
        const result = AnsiParser.renderToHTML(this.pendingText, []);
        this.activeClasses = result.classes;
        this.currentLineEl.innerHTML = result.html;
    }

    finishLine() {
        if (this.currentLineEl) {
            const textContent = this.currentLineEl.textContent.trim();
            if (textContent === '') {
                const prev = this.currentLineEl.previousElementSibling;
                if (!prev || prev.textContent.trim() === '') {
                    this.currentLineEl.remove();
                }
            } else {
                const prev = this.currentLineEl.previousElementSibling;
                if (prev) {
                    const prevText = prev.textContent.trim();
                    // 1. Eliminar duplicados exactos consecutivos (ecos repetidos de ConPTY)
                    if (prevText === textContent) {
                        this.currentLineEl.remove();
                    }
                    // 2. Si la línea previa era solo un prompt y la actual es ese mismo prompt con el comando
                    else if (textContent.startsWith(prevText) && (prevText.endsWith('>') || prevText.endsWith('$') || prevText.endsWith('#'))) {
                        prev.remove();
                    }
                    // 3. Si la línea previa era prompt + comando y la actual es solo el eco del mismo comando sin prompt
                    else if (prevText.endsWith('> ' + textContent) || prevText.endsWith('>' + textContent) ||
                             prevText.endsWith('$ ' + textContent) || prevText.endsWith('$' + textContent) ||
                             prevText.endsWith('# ' + textContent) || prevText.endsWith('#' + textContent)) {
                        this.currentLineEl.remove();
                    }
                }
            }
        }
        this.currentLineEl = null;
        this.pendingText = '';
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
        this.activeClasses = [];
        this.ignoreEchoCommand = null;
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

        if (status === 'disconnected') {
            if (this.promptPathEl) {
                this.promptPathEl.textContent = '~';
            }
        }
    }

    handleResize() {
        const cols = Math.max(40, Math.floor(this.outputEl.clientWidth / 8.5));
        const rows = Math.max(10, Math.floor(this.outputEl.clientHeight / 18));
        this.ws.sendResize(cols, rows);
    }

    parseApiError(raw) {
        try {
            const parsed = JSON.parse(raw);
            if (parsed && parsed.error) return parsed.error;
        } catch (e) {}
        return raw || 'Error desconocido';
    }

    escapeHTML(str) {
        if (!str) return '';
        return String(str)
            .replace(/&/g, '&amp;')
            .replace(/</g, '&lt;')
            .replace(/>/g, '&gt;')
            .replace(/"/g, '&quot;');
    }

    escapeAttr(str) {
        if (!str) return '';
        return String(str).replace(/["'<>]/g, '');
    }
}

document.addEventListener('DOMContentLoaded', () => {
    new TerminalApp();
});
