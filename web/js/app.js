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
        this.hostListEl = document.getElementById('host-list');
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
        this.vmTemplateSelect = document.getElementById('vm-template');
        this.vmCoresInput = document.getElementById('vm-cores');
        this.vmMemoryInput = document.getElementById('vm-memory');
        this.templates = [];

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
        this.cfgName = document.getElementById('cfg-name');

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
        this.hosts = [];
        this.activeHostId = '';
        this.editingHostId = null;

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
        this.loadTemplates();
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
                await this.loadHosts();

                if (!this.currentConfig.sshHost || !this.currentConfig.sshUser) {
                    this.openHostModal();
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
        this.cfgName.value = cfg.name || `${cfg.sshUser || 'usuario'}@${cfg.sshHost || 'host'}`;
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

    async loadHosts() {
        try {
            const res = await fetch('/api/hosts');
            if (!res.ok) return;
            const data = await res.json();
            this.hosts = Array.isArray(data.hosts) ? data.hosts : [];
            this.activeHostId = data.activeHostId || '';
            this.renderHostList();
        } catch (e) {
            console.warn('No se pudo cargar la lista de hosts:', e);
        }
    }

    renderHostList() {
        if (!this.hostListEl) return;
        this.hostListEl.replaceChildren();
        if (this.hosts.length === 0) {
            const empty = document.createElement('div');
            empty.className = 'empty-vm-list';
            empty.textContent = 'No hay hosts guardados';
            this.hostListEl.appendChild(empty);
            return;
        }

        this.hosts.forEach((host) => {
            const card = document.createElement('div');
            card.className = `host-card${host.id === this.activeHostId ? ' active' : ''}`;
            const details = document.createElement('div');
            details.className = 'host-card-details';
            const name = document.createElement('strong');
            name.textContent = `${host.sshUser || 'usuario'}@${host.sshHost || 'host'}`;
            const target = document.createElement('span');
            target.textContent = `${host.sshUser}@${host.sshHost}:${host.sshPort} · ${(host.sshShell || 'powershell').toUpperCase()}`;
            details.append(name, target);

            const actions = document.createElement('div');
            actions.className = 'host-card-actions';
            const useButton = document.createElement('button');
            useButton.className = 'btn btn-primary';
            useButton.type = 'button';
            useButton.textContent = host.id === this.activeHostId ? 'Activo' : 'Usar';
            useButton.disabled = host.id === this.activeHostId;
            useButton.addEventListener('click', () => this.activateHost(host.id));
            const editButton = document.createElement('button');
            editButton.className = 'btn btn-secondary';
            editButton.type = 'button';
            editButton.textContent = 'Editar';
            editButton.addEventListener('click', () => this.openHostModal(host));
            const deleteButton = document.createElement('button');
            deleteButton.className = 'btn btn-danger';
            deleteButton.type = 'button';
            deleteButton.textContent = 'Eliminar';
            deleteButton.disabled = this.hosts.length <= 1;
            deleteButton.addEventListener('click', () => this.deleteHost(host));

            actions.append(useButton, editButton, deleteButton);
            card.append(details, actions);
            this.hostListEl.appendChild(card);
        });
    }

    async activateHost(id) {
        try {
            const res = await fetch(`/api/hosts/${encodeURIComponent(id)}/activate`, { method: 'POST' });
            if (!res.ok) {
                this.appendErrorLine(`[Hosts] ${this.parseApiError(await res.text())}`);
                return;
            }
            const host = await res.json();
            this.currentConfig = { ...host };
            this.updateTargetPill(host);
            this.updateSidebarConfig(host);
            this.updateCurrentPath(host.sshStartPath || '~');
            this.activeHostId = id;
            this.renderHostList();
            this.clearOutput();
            this.appendInfoLine(`[Iniciando sesión SSH a ${host.sshUser}@${host.sshHost}:${host.sshPort} (${host.sshShell})...]`);
            this.connectCurrent();
        } catch (e) {
            console.warn('No se pudo activar el host:', e);
        }
    }

    async deleteHost(host) {
        if (!window.confirm(`¿Eliminar el host "${host.name}"?`)) return;
        try {
            const res = await fetch(`/api/hosts/${encodeURIComponent(host.id)}`, { method: 'DELETE' });
            if (!res.ok) {
                this.appendErrorLine(`[Hosts] ${this.parseApiError(await res.text())}`);
                return;
            }
            await this.loadHosts();
        } catch (e) {
            console.warn('No se pudo eliminar el host:', e);
        }
    }

    connectCurrent() {
        this.currentVM = null;
        this.updateTargetPill(this.currentConfig);
        const promptChar = document.querySelector('.prompt-char');
        if (promptChar) {
            promptChar.textContent = '>';
        }
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
            this.openHostModal();
        });

        // Control del formulario de creación de VM
        this.openCreateVmBtn.addEventListener('click', () => {
            this.hideVMFeedback();
            this.vmFormWrap.classList.toggle('hidden');
            if (!this.vmFormWrap.classList.contains('hidden')) {
                this.loadTemplates();
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
    // Carga de Plantillas (Templates) desde C:\VMManager\Templates
    // =========================================================================
    async loadTemplates() {
        if (!this.vmTemplateSelect) return;
        this.vmTemplateSelect.innerHTML = `<option value="" disabled selected>Cargando plantillas disponibles...</option>`;
        try {
            const res = await fetch('/api/templates');
            if (!res.ok) {
                this.vmTemplateSelect.innerHTML = `<option value="" disabled selected>Error al cargar plantillas</option>`;
                return;
            }
            const data = await res.json();
            this.templates = Array.isArray(data) ? data : [];
            this.renderTemplateSelect();
        } catch (e) {
            console.warn('No se pudieron cargar las plantillas:', e);
            this.vmTemplateSelect.innerHTML = `<option value="" disabled selected>No se pudieron cargar las plantillas</option>`;
        }
    }

    renderTemplateSelect() {
        if (!this.vmTemplateSelect) return;
        if (!this.templates || this.templates.length === 0) {
            this.vmTemplateSelect.innerHTML = `<option value="" disabled selected>No hay plantillas disponibles en C:\\VMManager\\Templates</option>`;
            return;
        }

        this.vmTemplateSelect.innerHTML = `
            <option value="" disabled selected>Selecciona una plantilla...</option>
            ${this.templates.map((t) => `
                <option value="${this.escapeAttr(t.id)}">
                    ${this.escapeHTML(t.name || t.id)} (${this.escapeHTML(t.osType || 'Linux')})
                </option>
            `).join('')}
        `;
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

        this.vmListEl.innerHTML = vms.map((vm) => {
            const stateLower = (vm.state || '').toLowerCase();
            const isRunning = stateLower === 'running' || stateLower === 'ejecutando';
            const isStopped = !isRunning;
            const stateColor = isRunning ? 'var(--accent-green)' : stateLower === 'poweroff' ? 'var(--accent-red)' : 'var(--accent-yellow)';
            const templateLabel = vm.templateName || vm.os || 'Plantilla Base';

            return `
            <div class="vm-card" id="vm-card-${this.escapeAttr(vm.name)}">
                <div class="vm-card-header">
                    <span class="vm-name">${this.escapeHTML(vm.name)}</span>
                    <span class="vm-os">${this.escapeHTML(templateLabel)}</span>
                </div>
                <div class="vm-card-details">
                    <span><strong>CPU:</strong> ${vm.cores || 2} Cores</span>
                    <span><strong>RAM:</strong> ${vm.memoryMB || 4096} MB</span>
                    <span><strong>Usuario SSH:</strong> ${this.escapeHTML(vm.sshUser || 'bootstrap')}</span>
                    <span><strong>SSH Port:</strong> ${vm.hostPort ? `${vm.host || ''}:${vm.hostPort}` : 'NAT'}</span>
                    <span><strong>Estado:</strong> <span style="color:${stateColor}">${this.escapeHTML(vm.state || 'Creada')}</span></span>
                </div>
                <div class="vm-card-delete-loader hidden">
                    <span class="vm-card-spinner"></span>
                    <span>Procesando...</span>
                </div>
                <div class="vm-card-actions">
                    <button class="btn btn-secondary view-vm" data-id="${this.escapeAttr(vm.name)}">Ver</button>
                    ${isStopped ? `<button class="btn btn-success start-vm" data-id="${this.escapeAttr(vm.name)}" title="Iniciar máquina virtual">
                        <span class="btn-spinner hidden" aria-hidden="true"></span>
                        <span class="btn-text">▶ Iniciar</span>
                    </button>` : ''}
                    ${isRunning ? `
                    <button class="btn btn-primary terminal-vm" data-id="${this.escapeAttr(vm.name)}" title="Abrir terminal SSH interactivo a esta VM">
                        &gt;_ Terminal
                    </button>
                    <button class="btn btn-warning stop-vm" data-id="${this.escapeAttr(vm.name)}" title="Apagado limpio ACPI">
                        <span class="btn-spinner hidden" aria-hidden="true"></span>
                        <span class="btn-text">⏹ Apagar</span>
                    </button>
                    <button class="btn btn-danger-outline poweroff-vm" data-id="${this.escapeAttr(vm.name)}" title="Apagado forzado inmediato">
                        ⚡ Poweroff
                    </button>` : ''}
                    <button class="btn btn-danger delete-vm" data-id="${this.escapeAttr(vm.name)}">Eliminar</button>
                </div>
            </div>
        `;
        }).join('');

        this.vmListEl.querySelectorAll('.view-vm').forEach((btn) => {
            btn.addEventListener('click', () => {
                const id = btn.dataset.id;
                this.viewVirtualMachine(id);
            });
        });

        this.vmListEl.querySelectorAll('.start-vm').forEach((btn) => {
            btn.addEventListener('click', () => {
                this.startVM(btn.dataset.id, btn);
            });
        });

        this.vmListEl.querySelectorAll('.terminal-vm').forEach((btn) => {
            btn.addEventListener('click', () => {
                this.connectTerminalToVM(btn.dataset.id);
            });
        });

        this.vmListEl.querySelectorAll('.stop-vm').forEach((btn) => {
            btn.addEventListener('click', () => {
                this.stopVM(btn.dataset.id, btn);
            });
        });

        this.vmListEl.querySelectorAll('.poweroff-vm').forEach((btn) => {
            btn.addEventListener('click', () => {
                this.forcePoweroffVM(btn.dataset.id, btn);
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
            const memoryMB = vm.memoryMB > 0 ? vm.memoryMB : 4096;
            const templateName = vm.templateName || vm.os || 'Plantilla Base';

            this.vmDetailsContent.innerHTML = `
                <div class="vm-detail-item">
                    <span class="vm-detail-key">Nombre de la MV</span>
                    <span class="vm-detail-val">${this.escapeHTML(vm.name)}</span>
                </div>
                <div class="vm-detail-item">
                    <span class="vm-detail-key">Plantilla Base</span>
                    <span class="vm-detail-val" style="color:var(--accent-green)">${this.escapeHTML(templateName)}</span>
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
                    <span class="vm-detail-key">Usuario SSH Dedicado</span>
                    <span class="vm-detail-val" style="color:var(--accent-yellow)">${this.escapeHTML(vm.sshUser || 'bootstrap')}</span>
                </div>
                <div class="vm-detail-item">
                    <span class="vm-detail-key">Puerto SSH (NAT)</span>
                    <span class="vm-detail-val">${vm.hostPort ? `${vm.host || 'Host'}:${vm.hostPort}` : 'NAT :22'}</span>
                </div>
                <div class="vm-detail-item full-width">
                    <span class="vm-detail-key">Estado Actual</span>
                    <span class="vm-detail-val" style="color:var(--accent)">${this.escapeHTML(vm.state || 'Creada')}</span>
                </div>
                ${vm.diskPath ? `
                <div class="vm-detail-item full-width">
                    <span class="vm-detail-key">Disco Base (Multiattach)</span>
                    <span class="vm-detail-val" style="font-size:12px">${this.escapeHTML(vm.diskPath)}</span>
                </div>` : ''}
                ${vm.publicKey ? `
                <div class="vm-detail-item full-width">
                    <span class="vm-detail-key">Clave Pública Ed25519</span>
                    <span class="vm-detail-val" style="font-size:11px;word-break:break-all">${this.escapeHTML(vm.publicKey)}</span>
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

        if (this.isCreatingVM) {
            return;
        }

        const name = this.vmNameInput.value.trim();
        const templateId = this.vmTemplateSelect ? this.vmTemplateSelect.value : '';
        const cores = parseInt(this.vmCoresInput.value, 10) || 2;
        const memoryMB = parseInt(this.vmMemoryInput.value, 10) || 4096;

        if (!name) {
            this.showVMCreateFeedback('El nombre de la máquina es obligatorio.', 'error');
            this.vmNameInput.focus();
            return;
        }

        if (!templateId) {
            this.showVMCreateFeedback('Debes seleccionar una plantilla base.', 'error');
            return;
        }

        this.isCreatingVM = true;
        this.setCreateVMLoading(true);
        this.showVMCreateFeedback('Aprovisionando máquina virtual en VirtualBox y configurando acceso SSH... Por favor espera (~1-2 min).', 'info');

        try {
            const payload = { name, templateId, cores, memoryMB };
            const res = await fetch('/api/vms', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(payload),
            });

            if (!res.ok) {
                const text = await res.text();
                const errMsg = this.parseApiError(text);
                this.showVMCreateFeedback(`Error al crear VM: ${errMsg}`, 'error');
                return;
            }

            const created = await res.json();
            this.showVMCreateFeedback(`¡Máquina virtual '${name}' creada y aprovisionada con éxito!`, 'success');
            this.appendInfoLine(`[Crear VM] '${name}' creada exitosamente con plantilla '${created.templateName || templateId}'. Usuario: ${created.sshUser || 'propio'}, Puerto SSH: ${created.hostPort || 'NAT'}`);

            setTimeout(() => {
                this.vmFormWrap.classList.add('hidden');
                this.vmForm.reset();
                this.hideVMFeedback();
            }, 1200);

            await this.loadVirtualMachines();
            await this.loadKeys();
        } catch (err) {
            console.warn('Error al crear VM:', err);
            this.showVMCreateFeedback('No se pudo conectar al servidor para crear la MV.', 'error');
        } finally {
            this.isCreatingVM = false;
            this.setCreateVMLoading(false);
        }
    }

    setCreateVMLoading(loading) {
        if (!this.vmSubmitBtn) return;
        const spinner = this.vmSubmitBtn.querySelector('.btn-spinner');
        const text = this.vmSubmitBtn.querySelector('.btn-text');

        this.vmSubmitBtn.disabled = loading;
        if (this.cancelVmFormBtn) this.cancelVmFormBtn.disabled = loading;
        if (this.vmNameInput) this.vmNameInput.disabled = loading;
        if (this.vmTemplateSelect) this.vmTemplateSelect.disabled = loading;
        if (this.vmCoresInput) this.vmCoresInput.disabled = loading;
        if (this.vmMemoryInput) this.vmMemoryInput.disabled = loading;

        if (spinner) spinner.classList.toggle('hidden', !loading);
        if (text) text.textContent = loading ? 'Aprovisionando (~1-2 min)...' : 'Crear Máquina';
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
    // Acciones de Inicio, Apagado y Terminal de Máquinas Virtuales
    // =========================================================================
    async startVM(name, btn) {
        const spinner = btn ? btn.querySelector('.btn-spinner') : null;
        const text = btn ? btn.querySelector('.btn-text') : null;

        if (btn) btn.disabled = true;
        if (spinner) spinner.classList.remove('hidden');
        if (text) text.textContent = 'Iniciando...';

        try {
            const res = await fetch(`/api/vms/${encodeURIComponent(name)}/start`, { method: 'POST' });
            if (!res.ok) {
                const errText = await res.text();
                this.appendErrorLine(`[Iniciar VM] Error al iniciar '${name}': ${this.parseApiError(errText)}`);
            } else {
                this.appendInfoLine(`[Iniciar VM] '${name}' iniciada correctamente.`);
            }
        } catch (e) {
            this.appendErrorLine(`[Iniciar VM] No se pudo conectar al servidor para iniciar '${name}'.`);
        } finally {
            if (btn) btn.disabled = false;
            if (spinner) spinner.classList.add('hidden');
            if (text) text.textContent = '▶ Iniciar';
            await this.loadVirtualMachines();
        }
    }

    async stopVM(name, btn) {
        const spinner = btn ? btn.querySelector('.btn-spinner') : null;
        const text = btn ? btn.querySelector('.btn-text') : null;

        if (btn) btn.disabled = true;
        if (spinner) spinner.classList.remove('hidden');
        if (text) text.textContent = 'Apagando...';

        try {
            const res = await fetch(`/api/vms/${encodeURIComponent(name)}/stop`, { method: 'POST' });
            if (!res.ok) {
                const errText = await res.text();
                this.appendErrorLine(`[Apagar VM] Error al apagar '${name}': ${this.parseApiError(errText)}`);
            } else {
                this.appendInfoLine(`[Apagar VM] '${name}' apagada limpiamente (ACPI).`);
            }
        } catch (e) {
            this.appendErrorLine(`[Apagar VM] No se pudo conectar al servidor para apagar '${name}'.`);
        } finally {
            if (btn) btn.disabled = false;
            if (spinner) spinner.classList.add('hidden');
            if (text) text.textContent = '⏹ Apagar';
            await this.loadVirtualMachines();
        }
    }

    async forcePoweroffVM(name, btn) {
        if (!confirm(`¿Estás seguro de forzar el apagado inmediato (Poweroff) de la máquina virtual '${name}'?`)) {
            return;
        }

        if (btn) btn.disabled = true;

        try {
            const res = await fetch(`/api/vms/${encodeURIComponent(name)}/poweroff`, { method: 'POST' });
            if (!res.ok) {
                const errText = await res.text();
                this.appendErrorLine(`[Poweroff VM] Error al forzar apagado de '${name}': ${this.parseApiError(errText)}`);
            } else {
                this.appendInfoLine(`[Poweroff VM] '${name}' apagada forzadamente.`);
            }
        } catch (e) {
            this.appendErrorLine(`[Poweroff VM] No se pudo conectar al servidor para forzar apagado de '${name}'.`);
        } finally {
            if (btn) btn.disabled = false;
            await this.loadVirtualMachines();
        }
    }

    connectTerminalToVM(name) {
        this.clearOutput();
        this.appendInfoLine(`[Terminal VM] Conectando terminal SSH interactivo a '${name}'...`);
        this.toggleSidebar(false);
        this.currentVM = name;
        if (this.targetUserHostEl) {
            this.targetUserHostEl.textContent = `[VM] ${name}`;
            this.targetUserHostEl.title = `Conectado a la máquina virtual ${name}`;
        }
        if (this.targetShellBadgeEl) {
            this.targetShellBadgeEl.textContent = 'bash';
            this.targetShellBadgeEl.className = 'badge-shell bash';
        }
        const promptChar = document.querySelector('.prompt-char');
        if (promptChar) {
            promptChar.textContent = '$';
        }
        this.ws.reconnectWith({ vm: name });
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

    openHostModal(host = null) {
        this.editingHostId = host ? host.id : null;
        this.populateForm(host || this.currentConfig);
        this.clearModalFeedback();
        this.modalEl.classList.remove('hidden');
        this.modalEl.setAttribute('aria-hidden', 'false');
        setTimeout(() => this.cfgHost.focus(), 50);
    }

    openModal() {
        this.openHostModal();
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

        const name = this.cfgName.value.trim();
        const host = this.cfgHost.value.trim();
        const port = parseInt(this.cfgPort.value.trim(), 10) || 22;
        const user = this.cfgUser.value.trim();
        const shell = this.getSelectedShell();
        const startPath = this.cfgStartPath.value.trim();
        const keyPath = this.cfgKeyPath.value.trim();

        if (!name) {
            this.showModalFeedback('El nombre del host es obligatorio.', 'error');
            this.cfgName.focus();
            return;
        }

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
            name,
            sshHost: host,
            sshPort: port,
            sshUser: user,
            sshShell: shell,
            sshStartPath: startPath,
            sshKeyPath: keyPath,
        };

        try {
            const endpoint = this.editingHostId
                ? `/api/hosts/${encodeURIComponent(this.editingHostId)}`
                : '/api/hosts';
            const res = await fetch(endpoint, {
                method: this.editingHostId ? 'PUT' : 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(newConfig),
            });
            if (!res.ok) {
                this.showModalFeedback(this.parseApiError(await res.text()), 'error');
                return;
            }
            const savedHost = await res.json();
            if (this.editingHostId && this.editingHostId === this.activeHostId) {
                this.currentConfig = { ...savedHost };
                this.updateTargetPill(savedHost);
                this.updateSidebarConfig(savedHost);
                this.updateCurrentPath(savedHost.sshStartPath || '~');
                this.connectCurrent();
            }
            await this.loadHosts();
            this.closeModal();
            return;
        } catch (err) {
            this.showModalFeedback('No se pudo guardar el host.', 'error');
            console.warn('No se pudo guardar el host en el backend:', err);
            return;
        }
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
