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

        // Elementos de Contenedores Docker
        this.containersListEl = document.getElementById('containers-list');
        this.containersVmFilter = document.getElementById('containers-vm-filter');
        this.dockerStatusPill = document.getElementById('docker-status-pill');
        this.openCreateContainerBtn = document.getElementById('open-create-container-btn');
        this.refreshContainersBtn = document.getElementById('refresh-containers-btn');
        this.containers = [];
        this.dockerCatalog = [];

        // Asistente Wizard de creación
        this.wizardModal = document.getElementById('container-wizard-modal');
        this.closeWizardBtn = document.getElementById('close-container-wizard-btn');
        this.wizCancelBtn = document.getElementById('wiz-cancel-btn');
        this.wizPrevBtn = document.getElementById('wiz-prev-btn');
        this.wizNextBtn = document.getElementById('wiz-next-btn');
        this.wizSubmitBtn = document.getElementById('wiz-submit-btn');
        this.wizStepPills = Array.from(document.querySelectorAll('.wizard-step-pill'));
        this.wizStepContents = Array.from(document.querySelectorAll('.wizard-step-content'));
        this.wizVmSelect = document.getElementById('wiz-vm-select');
        this.wizCategoriesGrid = document.getElementById('wiz-categories-grid');
        this.wizCategorySelectedTitle = document.getElementById('wiz-category-selected-title');
        this.wizAppsGrid = document.getElementById('wiz-apps-grid');
        this.wizNameInput = document.getElementById('wiz-name');
        this.wizRestartSelect = document.getElementById('wiz-restart');
        this.wizImageInput = document.getElementById('wiz-image');
        this.wizPortsContainer = document.getElementById('wiz-ports-container');
        this.wizEnvContainer = document.getElementById('wiz-env-container');
        this.wizVolumesContainer = document.getElementById('wiz-volumes-container');
        this.wizAddPortBtn = document.getElementById('wiz-add-port-btn');
        this.wizAddEnvBtn = document.getElementById('wiz-add-env-btn');
        this.wizAddVolumeBtn = document.getElementById('wiz-add-volume-btn');
        this.wizNetworkSelect = document.getElementById('wiz-network');
        this.wizAddNetworkBtn = document.getElementById('wiz-add-network-btn');
        this.wizSummaryCard = document.getElementById('wiz-summary-card');
        this.wizFeedback = document.getElementById('wiz-feedback');
        this.currentWizardStep = 1;
        this.selectedCategory = null;
        this.selectedTemplate = null;

        // Subnavegación de Docker (Contenedores / Imágenes)
        this.dockerSubnav = document.getElementById('docker-subnav');
        this.dockerSubnavBtns = Array.from(document.querySelectorAll('.docker-subnav-btn'));
        this.activeDockerSubnav = 'containers';

        // Elementos de Gestión de Imágenes Docker
        this.imagesListEl = document.getElementById('images-list');
        this.imagesVmFilter = document.getElementById('images-vm-filter');
        this.openCreateImageBtn = document.getElementById('open-create-image-btn');
        this.refreshImagesBtn = document.getElementById('refresh-images-btn');
        this.images = [];

        // Modal de Creación / Compilación de Imagen
        this.imageBuildModal = document.getElementById('image-build-modal');
        this.closeImageBuildBtn = document.getElementById('close-image-build-btn');
        this.cancelImageBuildBtn = document.getElementById('cancel-image-build-btn');
        this.buildVmSelect = document.getElementById('build-vm-select');
        this.buildImageName = document.getElementById('build-image-name');
        this.buildImageTag = document.getElementById('build-image-tag');
        this.dfTabs = Array.from(document.querySelectorAll('.df-tab'));
        this.dfUploadPanel = document.getElementById('df-upload-panel');
        this.dfPastePanel = document.getElementById('df-paste-panel');
        this.dfFileInput = document.getElementById('df-file-input');
        this.dfFileDropzone = document.getElementById('df-file-dropzone');
        this.dfSelectedCard = document.getElementById('df-selected-card');
        this.dfChangeFileBtn = document.getElementById('df-change-file-btn');
        this.dfFileName = document.getElementById('df-file-name');
        this.dfFileSize = document.getElementById('df-file-size');
        this.dfAttachTriggerBtn = document.getElementById('df-attach-trigger-btn');
        this.dfExtraFilesInput = document.getElementById('df-extra-files-input');
        this.dfContextDropzone = document.getElementById('df-context-dropzone');
        this.dfAttachedCount = document.getElementById('df-attached-count');
        this.dfAttachmentsContainer = document.getElementById('df-attachments-container');
        this.dfAttachmentsList = document.getElementById('df-attachments-list');
        this.dfClearAllBtn = document.getElementById('df-clear-all-btn');
        this.attachedBuildFiles = [];
        this.buildDockerfileContent = document.getElementById('build-dockerfile-content');
        this.buildImageFeedback = document.getElementById('build-image-feedback');
        this.buildLogContainer = document.getElementById('build-log-container');
        this.buildLogOutput = document.getElementById('build-log-output');
        this.submitImageBuildBtn = document.getElementById('submit-image-build-btn');
        this.submitImageBuildRunBtn = document.getElementById('submit-image-build-run-btn');
        this.activeDfTab = 'paste';

        // Modal de Eliminación de Imagen
        this.deleteImageModal = document.getElementById('delete-image-modal');
        this.closeDeleteImageModalBtn = document.getElementById('close-delete-image-modal');
        this.cancelDeleteImageBtn = document.getElementById('cancel-delete-image');
        this.confirmDeleteImageBtn = document.getElementById('confirm-delete-image');
        this.deleteImageNameEl = document.getElementById('delete-image-name');
        this.deleteImageVmTag = document.getElementById('delete-image-vm-tag');
        this.deleteImageForceCheckbox = document.getElementById('delete-image-force');
        this.deleteImageFeedback = document.getElementById('delete-image-feedback');
        this.pendingDeleteImage = null;

        // Opciones de Origen en Wizard
        this.wizSourceImages = document.getElementById('wiz-source-images');
        this.wizSourceCatalog = document.getElementById('wiz-source-catalog');
        this.wizImagesPanel = document.getElementById('wiz-images-panel');
        this.wizCatalogPanel = document.getElementById('wiz-catalog-panel');
        this.wizImagesGrid = document.getElementById('wiz-images-grid');
        this.wizCreateNewImageBtn = document.getElementById('wiz-create-new-image-btn');
        this.wizardSource = 'images';
        this.selectedWizardImage = null;

        // Modal de Logs de Contenedor
        this.containerLogsModal = document.getElementById('container-logs-modal');
        this.closeContainerLogsBtn = document.getElementById('close-container-logs-btn');
        this.btnCloseContainerLogs = document.getElementById('btn-close-container-logs');
        this.refreshContainerLogsBtn = document.getElementById('refresh-container-logs-btn');
        this.containerLogsTail = document.getElementById('container-logs-tail');
        this.containerLogsTitle = document.getElementById('container-logs-title');
        this.containerLogsSubtitle = document.getElementById('container-logs-subtitle');
        this.containerLogsOutput = document.getElementById('container-logs-output');
        this.activeLogsTarget = null;

        // Modal de Eliminación de Contenedor
        this.deleteContainerModal = document.getElementById('delete-container-modal');
        this.closeDeleteContainerModalBtn = document.getElementById('close-delete-container-modal');
        this.cancelDeleteContainerBtn = document.getElementById('cancel-delete-container');
        this.confirmDeleteContainerBtn = document.getElementById('confirm-delete-container');
        this.deleteContainerNameEl = document.getElementById('delete-container-name');
        this.deleteContainerVmTag = document.getElementById('delete-container-vm-tag');
        this.deleteContainerForceCheckbox = document.getElementById('delete-container-force');
        this.deleteContainerFeedback = document.getElementById('delete-container-feedback');
        this.pendingDeleteContainer = null;

        // Modal de Inspección / Detalles de Contenedor
        this.containerDetailsModal = document.getElementById('container-details-modal');
        this.closeContainerDetailsBtn = document.getElementById('close-container-details-btn');
        this.btnCloseContainerDetails = document.getElementById('btn-close-container-details');
        this.containerDetailsContent = document.getElementById('container-details-content');

        // Modal de Creación de Imagen desde Contenedor (Commit)
        this.commitContainerModal = document.getElementById('commit-container-modal');
        this.closeCommitContainerBtn = document.getElementById('close-commit-container-modal');
        this.cancelCommitContainerBtn = document.getElementById('cancel-commit-container');
        this.commitContainerForm = document.getElementById('commit-container-form');
        this.commitContainerTargetName = document.getElementById('commit-container-target-name');
        this.commitContainerVmTag = document.getElementById('commit-container-vm-tag');
        this.commitImageNameInput = document.getElementById('commit-image-name');
        this.commitImageTagInput = document.getElementById('commit-image-tag');
        this.commitImageCommentInput = document.getElementById('commit-image-comment');
        this.commitContainerFeedback = document.getElementById('commit-container-feedback');
        this.submitCommitContainerBtn = document.getElementById('submit-commit-container');
        this.currentCommitVm = '';
        this.currentCommitContainerId = '';

        // Elementos de Gestión de Volúmenes Docker
        this.volumesListEl = document.getElementById('volumes-list');
        this.volumesVmFilter = document.getElementById('volumes-vm-filter');
        this.openCreateVolumeBtn = document.getElementById('open-create-volume-btn');
        this.refreshVolumesBtn = document.getElementById('refresh-volumes-btn');
        this.volumes = [];

        // Modal de Creación de Volumen
        this.createVolumeModal = document.getElementById('create-volume-modal');
        this.closeCreateVolumeModalBtn = document.getElementById('close-create-volume-modal');
        this.cancelCreateVolumeBtn = document.getElementById('cancel-create-volume-btn');
        this.createVolumeForm = document.getElementById('create-volume-form');
        this.createVolumeVm = document.getElementById('create-volume-vm');
        this.createVolumeName = document.getElementById('create-volume-name');
        this.createVolumeFeedback = document.getElementById('create-volume-feedback');
        this.submitCreateVolumeBtn = document.getElementById('submit-create-volume-btn');

        // Modal de Eliminación de Volumen
        this.deleteVolumeModal = document.getElementById('delete-volume-modal');
        this.closeDeleteVolumeModalBtn = document.getElementById('close-delete-volume-modal');
        this.cancelDeleteVolumeBtn = document.getElementById('cancel-delete-volume');
        this.confirmDeleteVolumeBtn = document.getElementById('confirm-delete-volume');
        this.deleteVolumeNameEl = document.getElementById('delete-volume-name');
        this.deleteVolumeVmTag = document.getElementById('delete-volume-vm-tag');
        this.deleteVolumeFeedback = document.getElementById('delete-volume-feedback');
        this.pendingDeleteVolume = null;

        // Elementos de Gestión de Redes Docker
        this.networksListEl = document.getElementById('networks-list');
        this.networksVmFilter = document.getElementById('networks-vm-filter');
        this.openCreateNetworkBtn = document.getElementById('open-create-network-btn');
        this.refreshNetworksBtn = document.getElementById('refresh-networks-btn');
        this.networks = [];

        // Modal de Creación de Red
        this.createNetworkModal = document.getElementById('create-network-modal');
        this.closeCreateNetworkModalBtn = document.getElementById('close-create-network-modal');
        this.cancelCreateNetworkBtn = document.getElementById('cancel-create-network-btn');
        this.createNetworkForm = document.getElementById('create-network-form');
        this.createNetworkVm = document.getElementById('create-network-vm');
        this.createNetworkName = document.getElementById('create-network-name');
        this.createNetworkDriver = document.getElementById('create-network-driver');
        this.createNetworkFeedback = document.getElementById('create-network-feedback');
        this.submitCreateNetworkBtn = document.getElementById('submit-create-network-btn');

        // Modal de Detalles de Red
        this.networkDetailsModal = document.getElementById('network-details-modal');
        this.closeNetworkDetailsModalBtn = document.getElementById('close-network-details-modal');
        this.closeNetworkDetailsBtn = document.getElementById('close-network-details-btn');
        this.networkDetailsBody = document.getElementById('network-details-body');

        // Modal de Eliminación de Red
        this.deleteNetworkModal = document.getElementById('delete-network-modal');
        this.closeDeleteNetworkModalBtn = document.getElementById('close-delete-network-modal');
        this.cancelDeleteNetworkBtn = document.getElementById('cancel-delete-network');
        this.confirmDeleteNetworkBtn = document.getElementById('confirm-delete-network');
        this.deleteNetworkNameEl = document.getElementById('delete-network-name');
        this.deleteNetworkVmTag = document.getElementById('delete-network-vm-tag');
        this.deleteNetworkFeedback = document.getElementById('delete-network-feedback');
        this.pendingDeleteNetwork = null;

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

        window.terminalApp = this;
        this.bindEvents();
        this.loadInitialConfig();
        this.loadTemplates();
        this.loadVirtualMachines();
        this.loadKeys();
        this.loadDockerCatalog();
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

        // Eventos de Contenedores y Asistente
        if (this.openCreateContainerBtn) {
            this.openCreateContainerBtn.addEventListener('click', () => {
                this.openContainerWizard(this.containersVmFilter ? this.containersVmFilter.value : '');
            });
        }
        if (this.refreshContainersBtn) {
            this.refreshContainersBtn.addEventListener('click', () => {
                this.loadContainers(this.containersVmFilter ? this.containersVmFilter.value : '');
            });
        }
        if (this.containersVmFilter) {
            this.containersVmFilter.addEventListener('change', () => {
                this.loadContainers(this.containersVmFilter.value);
            });
        }

        // Eventos del Asistente Wizard
        if (this.closeWizardBtn) {
            this.closeWizardBtn.addEventListener('click', () => this.closeContainerWizard());
        }
        if (this.wizCancelBtn) {
            this.wizCancelBtn.addEventListener('click', () => this.closeContainerWizard());
        }
        if (this.wizPrevBtn) {
            this.wizPrevBtn.addEventListener('click', () => this.goToWizardStep(this.currentWizardStep - 1));
        }
        if (this.wizNextBtn) {
            this.wizNextBtn.addEventListener('click', () => this.goToWizardStep(this.currentWizardStep + 1));
        }
        if (this.wizSubmitBtn) {
            this.wizSubmitBtn.addEventListener('click', () => this.submitCreateContainer());
        }
        if (this.wizAddPortBtn) {
            this.wizAddPortBtn.addEventListener('click', () => this.addWizPortRow());
        }
        if (this.wizAddEnvBtn) {
            this.wizAddEnvBtn.addEventListener('click', () => this.addWizEnvRow());
        }
        if (this.wizAddVolumeBtn) {
            this.wizAddVolumeBtn.addEventListener('click', () => this.addWizVolumeRow());
        }

        // Eventos de Logs de Contenedor
        if (this.closeContainerLogsBtn) {
            this.closeContainerLogsBtn.addEventListener('click', () => this.closeContainerLogs());
        }
        if (this.btnCloseContainerLogs) {
            this.btnCloseContainerLogs.addEventListener('click', () => this.closeContainerLogs());
        }
        if (this.refreshContainerLogsBtn) {
            this.refreshContainerLogsBtn.addEventListener('click', () => this.refreshContainerLogs());
        }
        if (this.containerLogsTail) {
            this.containerLogsTail.addEventListener('change', () => this.refreshContainerLogs());
        }

        // Eventos de Eliminación de Contenedor
        if (this.closeDeleteContainerModalBtn) {
            this.closeDeleteContainerModalBtn.addEventListener('click', () => this.closeDeleteContainerModal());
        }
        if (this.cancelDeleteContainerBtn) {
            this.cancelDeleteContainerBtn.addEventListener('click', () => this.closeDeleteContainerModal());
        }
        if (this.confirmDeleteContainerBtn) {
            this.confirmDeleteContainerBtn.addEventListener('click', () => this.executeDeleteContainer());
        }

        // Eventos de Gestión de Volúmenes
        if (this.openCreateVolumeBtn) {
            this.openCreateVolumeBtn.addEventListener('click', () => {
                this.createVolumeForm.reset();
                this.createVolumeFeedback.classList.add('hidden');
                
                // Limpiar select de VM y rellenar con la lista actual
                this.createVolumeVm.innerHTML = '<option value="" disabled selected>Selecciona una máquina...</option>';
                if (this.vms) {
                    this.vms.forEach(vm => {
                        const opt = document.createElement('option');
                        opt.value = vm.name || vm.Name;
                        opt.textContent = vm.name || vm.Name;
                        this.createVolumeVm.appendChild(opt);
                    });
                }

                if (this.volumesVmFilter && this.volumesVmFilter.value) {
                    this.createVolumeVm.value = this.volumesVmFilter.value;
                }
                
                this.createVolumeModal.classList.remove('hidden');
                setTimeout(() => this.createVolumeName.focus(), 50);
            });
        }
        if (this.closeCreateVolumeModalBtn) {
            this.closeCreateVolumeModalBtn.addEventListener('click', () => this.createVolumeModal.classList.add('hidden'));
        }
        if (this.cancelCreateVolumeBtn) {
            this.cancelCreateVolumeBtn.addEventListener('click', () => this.createVolumeModal.classList.add('hidden'));
        }
        if (this.createVolumeForm) {
            this.createVolumeForm.addEventListener('submit', (e) => {
                e.preventDefault();
                this.submitCreateVolume();
            });
        }

        if (this.refreshVolumesBtn) {
            this.refreshVolumesBtn.addEventListener('click', () => {
                this.loadVolumes(this.volumesVmFilter ? this.volumesVmFilter.value : '');
            });
        }
        if (this.volumesVmFilter) {
            this.volumesVmFilter.addEventListener('change', () => {
                this.loadVolumes(this.volumesVmFilter.value);
            });
        }

        // Eventos de Eliminación de Volumen
        if (this.closeDeleteVolumeModalBtn) {
            this.closeDeleteVolumeModalBtn.addEventListener('click', () => this.closeDeleteVolumeModal());
        }
        if (this.cancelDeleteVolumeBtn) {
            this.cancelDeleteVolumeBtn.addEventListener('click', () => this.closeDeleteVolumeModal());
        }
        if (this.confirmDeleteVolumeBtn) {
            this.confirmDeleteVolumeBtn.addEventListener('click', () => this.executeDeleteVolume());
        }

        // Eventos de Gestión de Redes Docker
        if (this.openCreateNetworkBtn) {
            this.openCreateNetworkBtn.addEventListener('click', () => {
                this.openCreateNetworkDialog();
            });
        }
        if (this.closeCreateNetworkModalBtn) {
            this.closeCreateNetworkModalBtn.addEventListener('click', () => this.createNetworkModal.classList.add('hidden'));
        }
        if (this.cancelCreateNetworkBtn) {
            this.cancelCreateNetworkBtn.addEventListener('click', () => this.createNetworkModal.classList.add('hidden'));
        }
        if (this.createNetworkForm) {
            this.createNetworkForm.addEventListener('submit', (e) => {
                e.preventDefault();
                this.submitCreateNetwork();
            });
        }
        if (this.refreshNetworksBtn) {
            this.refreshNetworksBtn.addEventListener('click', () => {
                this.loadNetworks(this.networksVmFilter ? this.networksVmFilter.value : '');
            });
        }
        if (this.networksVmFilter) {
            this.networksVmFilter.addEventListener('change', () => {
                this.loadNetworks(this.networksVmFilter.value);
            });
        }

        // Eventos de Detalles de Red
        if (this.closeNetworkDetailsModalBtn) {
            this.closeNetworkDetailsModalBtn.addEventListener('click', () => this.networkDetailsModal.classList.add('hidden'));
        }
        if (this.closeNetworkDetailsBtn) {
            this.closeNetworkDetailsBtn.addEventListener('click', () => this.networkDetailsModal.classList.add('hidden'));
        }

        // Eventos de Eliminación de Red
        if (this.closeDeleteNetworkModalBtn) {
            this.closeDeleteNetworkModalBtn.addEventListener('click', () => this.closeDeleteNetworkModal());
        }
        if (this.cancelDeleteNetworkBtn) {
            this.cancelDeleteNetworkBtn.addEventListener('click', () => this.closeDeleteNetworkModal());
        }
        if (this.confirmDeleteNetworkBtn) {
            this.confirmDeleteNetworkBtn.addEventListener('click', () => this.executeDeleteNetwork());
        }

        // Evento botón + Nueva red dentro del Asistente de Contenedores
        if (this.wizAddNetworkBtn) {
            this.wizAddNetworkBtn.addEventListener('click', () => {
                const vm = this.wizVmSelect ? this.wizVmSelect.value.trim() : '';
                if (!vm) {
                    this.showWizFeedback('Selecciona una máquina virtual primero.', 'error');
                    return;
                }
                this.openCreateNetworkDialog(vm, (newNetName) => {
                    this.updateWizardNetworks(vm, newNetName);
                });
            });
        }

        // Eventos de Detalles de Contenedor
        if (this.closeContainerDetailsBtn) {
            this.closeContainerDetailsBtn.addEventListener('click', () => this.closeContainerDetails());
        }
        if (this.btnCloseContainerDetails) {
            this.btnCloseContainerDetails.addEventListener('click', () => this.closeContainerDetails());
        }

        // Eventos de Subnavegación de Docker
        if (this.dockerSubnavBtns) {
            this.dockerSubnavBtns.forEach((btn) => {
                btn.addEventListener('click', () => {
                    this.switchDockerSubview(btn.dataset.subview);
                });
            });
        }

        // Eventos de Gestión de Imágenes Docker
        if (this.openCreateImageBtn) {
            this.openCreateImageBtn.addEventListener('click', () => {
                this.openBuildImageModal(this.imagesVmFilter ? this.imagesVmFilter.value : '');
            });
        }
        if (this.refreshImagesBtn) {
            this.refreshImagesBtn.addEventListener('click', () => {
                this.loadImages(this.imagesVmFilter ? this.imagesVmFilter.value : '');
            });
        }
        if (this.imagesVmFilter) {
            this.imagesVmFilter.addEventListener('change', () => {
                this.loadImages(this.imagesVmFilter.value);
            });
        }

        // Eventos del Modal Crear Imagen
        if (this.closeImageBuildBtn) {
            this.closeImageBuildBtn.addEventListener('click', () => this.closeBuildImageModal());
        }
        if (this.cancelImageBuildBtn) {
            this.cancelImageBuildBtn.addEventListener('click', () => this.closeBuildImageModal());
        }
        if (this.dfTabs) {
            this.dfTabs.forEach((tab) => {
                tab.addEventListener('click', () => {
                    this.switchDfTab(tab.dataset.dftab);
                });
            });
        }
        if (this.dfFileDropzone && this.dfFileInput) {
            this.dfFileDropzone.addEventListener('click', () => this.dfFileInput.click());
            this.dfFileInput.addEventListener('change', (e) => {
                if (e.target.files && e.target.files.length > 0) {
                    this.handlePrimaryFilesUpload(e.target.files);
                }
            });
            this.dfFileDropzone.addEventListener('dragover', (e) => {
                e.preventDefault();
                this.dfFileDropzone.classList.add('dragover');
            });
            this.dfFileDropzone.addEventListener('dragleave', () => {
                this.dfFileDropzone.classList.remove('dragover');
            });
            this.dfFileDropzone.addEventListener('drop', (e) => {
                e.preventDefault();
                this.dfFileDropzone.classList.remove('dragover');
                if (e.dataTransfer.files && e.dataTransfer.files.length > 0) {
                    this.handlePrimaryFilesUpload(e.dataTransfer.files);
                }
            });
        if (this.dfChangeFileBtn && this.dfFileInput) {
            this.dfChangeFileBtn.addEventListener('click', () => this.dfFileInput.click());
        }
        if (this.dfAttachTriggerBtn && this.dfExtraFilesInput) {
            this.dfAttachTriggerBtn.addEventListener('click', () => this.dfExtraFilesInput.click());
        }
        if (this.dfContextDropzone && this.dfExtraFilesInput) {
            this.dfContextDropzone.addEventListener('click', () => this.dfExtraFilesInput.click());
            this.dfExtraFilesInput.addEventListener('change', (e) => {
                if (e.target.files && e.target.files.length > 0) {
                    this.addAttachedFiles(e.target.files);
                    this.dfExtraFilesInput.value = '';
                }
            });
            this.dfContextDropzone.addEventListener('dragover', (e) => {
                e.preventDefault();
                this.dfContextDropzone.classList.add('dragover');
            });
            this.dfContextDropzone.addEventListener('dragleave', () => {
                this.dfContextDropzone.classList.remove('dragover');
            });
            this.dfContextDropzone.addEventListener('drop', (e) => {
                e.preventDefault();
                this.dfContextDropzone.classList.remove('dragover');
                if (e.dataTransfer.files && e.dataTransfer.files.length > 0) {
                    this.addAttachedFiles(e.dataTransfer.files);
                }
            });
        }
        if (this.dfClearAllBtn) {
            this.dfClearAllBtn.addEventListener('click', () => {
                this.attachedBuildFiles = [];
                this.renderAttachedFiles();
            });
        }
        if (this.submitImageBuildBtn) {
            this.submitImageBuildBtn.addEventListener('click', () => this.submitBuildImage(false));
        }
        if (this.submitImageBuildRunBtn) {
            this.submitImageBuildRunBtn.addEventListener('click', () => this.submitBuildImage(true));
        }

        // Eventos del Modal Eliminar Imagen
        if (this.closeDeleteImageModalBtn) {
            this.closeDeleteImageModalBtn.addEventListener('click', () => this.closeDeleteImageModal());
        }
        if (this.cancelDeleteImageBtn) {
            this.cancelDeleteImageBtn.addEventListener('click', () => this.closeDeleteImageModal());
        }
        if (this.confirmDeleteImageBtn) {
            this.confirmDeleteImageBtn.addEventListener('click', () => this.confirmDeleteImage());
        }

        // Eventos del Modal Crear Imagen desde Contenedor (Commit)
        if (this.closeCommitContainerBtn) {
            this.closeCommitContainerBtn.addEventListener('click', () => this.closeCommitContainerModal());
        }
        if (this.cancelCommitContainerBtn) {
            this.cancelCommitContainerBtn.addEventListener('click', () => this.closeCommitContainerModal());
        }
        if (this.commitContainerForm) {
            this.commitContainerForm.addEventListener('submit', (e) => {
                e.preventDefault();
                this.submitCommitContainer();
            });
        }

        // Eventos de Origen en Wizard
        if (this.wizSourceImages) {
            this.wizSourceImages.addEventListener('click', () => this.selectWizardSource('images'));
        }
        if (this.wizSourceCatalog) {
            this.wizSourceCatalog.addEventListener('click', () => this.selectWizardSource('catalog'));
        }
        if (this.wizCreateNewImageBtn) {
            this.wizCreateNewImageBtn.addEventListener('click', () => {
                this.openBuildImageModal(this.wizVmSelect ? this.wizVmSelect.value : '');
            });
        }

        // Cierre con tecla Escape
        window.addEventListener('keydown', (e) => {
            if (e.key === 'Escape') {
                if (this.isModalOpen()) {
                    this.closeModal();
                    this.closeDeleteModal();
                    this.closeDeleteKeyModal();
                    this.closeDetailsModal();
                    this.closeContainerWizard();
                    this.closeContainerLogs();
                    this.closeContainerDetails();
                    this.closeDeleteContainerModal();
                    this.closeBuildImageModal();
                    this.closeDeleteImageModal();
                    this.closeCommitContainerModal();
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
        if (this.wizardModal) {
            this.wizardModal.addEventListener('click', (e) => {
                if (e.target === this.wizardModal) this.closeContainerWizard();
            });
        }
        if (this.containerLogsModal) {
            this.containerLogsModal.addEventListener('click', (e) => {
                if (e.target === this.containerLogsModal) this.closeContainerLogs();
            });
        }
        if (this.containerDetailsModal) {
            this.containerDetailsModal.addEventListener('click', (e) => {
                if (e.target === this.containerDetailsModal) this.closeContainerDetails();
            });
        }
        if (this.deleteContainerModal) {
            this.deleteContainerModal.addEventListener('click', (e) => {
                if (e.target === this.deleteContainerModal) this.closeDeleteContainerModal();
            });
        }
        if (this.imageBuildModal) {
            this.imageBuildModal.addEventListener('click', (e) => {
                if (e.target === this.imageBuildModal) this.closeBuildImageModal();
            });
        }
        if (this.deleteImageModal) {
            this.deleteImageModal.addEventListener('click', (e) => {
                if (e.target === this.deleteImageModal) this.closeDeleteImageModal();
            });
        }
        if (this.commitContainerModal) {
            this.commitContainerModal.addEventListener('click', (e) => {
                if (e.target === this.commitContainerModal) this.closeCommitContainerModal();
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

    isModalOpen() {
        return (this.modalEl && !this.modalEl.classList.contains('hidden')) ||
            (this.deleteVmModal && !this.deleteVmModal.classList.contains('hidden')) ||
            (this.deleteKeyModal && !this.deleteKeyModal.classList.contains('hidden')) ||
            (this.vmDetailsModal && !this.vmDetailsModal.classList.contains('hidden')) ||
            (this.wizardModal && !this.wizardModal.classList.contains('hidden')) ||
            (this.containerLogsModal && !this.containerLogsModal.classList.contains('hidden')) ||
            (this.containerDetailsModal && !this.containerDetailsModal.classList.contains('hidden')) ||
            (this.deleteContainerModal && !this.deleteContainerModal.classList.contains('hidden')) ||
            (this.imageBuildModal && !this.imageBuildModal.classList.contains('hidden')) ||
            (this.deleteImageModal && !this.deleteImageModal.classList.contains('hidden')) ||
            (this.commitContainerModal && !this.commitContainerModal.classList.contains('hidden'));
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
            this.updateContainersVmFilter();
            this.updateImagesVmFilter();
        }
    }

    switchSidebarView(view) {
        const isDocker = (view === 'docker' || view === 'containers' || view === 'images' || view === 'volumes');

        // Resaltar pestañas principales
        this.sideLinks.forEach((l) => {
            if (isDocker && l.dataset.menu === 'docker') {
                l.classList.add('active');
            } else if (!isDocker && l.dataset.menu === view) {
                l.classList.add('active');
            } else {
                l.classList.remove('active');
            }
        });

        // Mostrar/ocultar barra de subnavegación de Docker
        if (this.dockerSubnav) {
            this.dockerSubnav.classList.toggle('hidden', !isDocker);
        }

        if (isDocker) {
            let subview = this.activeDockerSubnav || 'containers';
            if (view === 'images' || view === 'volumes') {
                subview = view;
            }
            this.switchDockerSubview(subview);
            return;
        }

        this.sideViews.forEach((s) => s.classList.toggle('active', s.id === `sidebar-${view}-view`));
        if (view === 'keys') {
            this.loadKeys();
        } else if (view === 'vms') {
            this.loadVirtualMachines();
        }
    }

    switchDockerSubview(subview) {
        this.activeDockerSubnav = subview;
        if (this.dockerSubnavBtns) {
            this.dockerSubnavBtns.forEach((b) => b.classList.toggle('active', b.dataset.subview === subview));
        }

        this.sideViews.forEach((s) => {
            if (subview === 'containers') {
                s.classList.toggle('active', s.id === 'sidebar-containers-view');
            } else if (subview === 'images') {
                s.classList.toggle('active', s.id === 'sidebar-images-view');
            } else if (subview === 'volumes') {
                s.classList.toggle('active', s.id === 'sidebar-volumes-view');
            } else if (subview === 'networks') {
                s.classList.toggle('active', s.id === 'sidebar-networks-view');
            } else {
                s.classList.remove('active');
            }
        });

        if (subview === 'containers') {
            this.updateContainersVmFilter();
            this.loadContainers(this.containersVmFilter ? this.containersVmFilter.value : '');
        } else if (subview === 'images') {
            this.updateImagesVmFilter();
            this.loadImages(this.imagesVmFilter ? this.imagesVmFilter.value : '');
        } else if (subview === 'volumes') {
            this.updateVolumesVmFilter();
            this.loadVolumes(this.volumesVmFilter ? this.volumesVmFilter.value : '');
        } else if (subview === 'networks') {
            this.updateNetworksVmFilter();
            this.loadNetworks(this.networksVmFilter ? this.networksVmFilter.value : '');
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
                        <span class="btn-text">Iniciar</span>
                    </button>` : ''}
                    ${isRunning ? `
                    <button class="btn btn-primary terminal-vm" data-id="${this.escapeAttr(vm.name)}" title="Abrir terminal SSH interactivo a esta VM">
                        Terminal
                    </button>
                    <button class="btn btn-warning stop-vm" data-id="${this.escapeAttr(vm.name)}" title="Apagado limpio ACPI">
                        <span class="btn-spinner hidden" aria-hidden="true"></span>
                        <span class="btn-text">Apagar</span>
                    </button>
                    <button class="btn btn-danger-outline poweroff-vm" data-id="${this.escapeAttr(vm.name)}" title="Apagado forzado inmediato">
                        Forzar apagado
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
            if (text) text.textContent = 'Iniciar';
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
            if (text) text.textContent = 'Apagar';
            await this.loadVirtualMachines();
        }
    }

    async forcePoweroffVM(name, btn) {
        if (!confirm(`¿Estás seguro de forzar el apagado inmediato de la máquina virtual '${name}'?`)) {
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
                        VM: ${this.escapeHTML(key.vmName || 'General')}
                    </span>
                </div>
                <div class="key-card-details">
                    <span><strong>Algoritmo:</strong> ${this.escapeHTML(key.algorithm || 'RSA 2048')}</span>
                    <span class="key-fingerprint"><strong>Huella:</strong> ${this.escapeHTML(key.fingerprint || '-')}</span>
                </div>
                <div class="key-card-actions">
                    <a href="/api/keys/${encodeURIComponent(key.id)}/private" class="btn btn-primary" download title="Descargar clave privada .pem">
                        Clave privada
                    </a>
                    <a href="/api/keys/${encodeURIComponent(key.id)}/public" class="btn btn-secondary" download title="Descargar clave pública .pub">
                        Clave pública
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

        // Si tenemos una línea activa con el prompt en pantalla, completarla con el comando formateado
        if (this.currentLineEl) {
            const promptText = AnsiParser.stripAnsi(this.pendingText || this.currentLineEl.textContent || '').trim();
            if (promptText && (promptText.endsWith('>') || promptText.endsWith('$') || promptText.endsWith('#'))) {
                this.lastSubmittedPrompt = promptText;
                this.lastSubmittedCommand = cmdTrimmed;
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

        // Normalizar saltos de línea evitando borrar accidentalmente el contenido previo
        const normalized = processed.replace(/\r+\n/g, '\n');
        const lines = normalized.split('\n');

        for (let i = 0; i < lines.length; i++) {
            let linePart = lines[i];

            if (linePart.includes('\r')) {
                const subParts = linePart.split(/\r+/);
                linePart = subParts[subParts.length - 1];
                this.pendingText = '';
            }

            if (linePart) {
                this.appendToCurrentLine(linePart);
            }

            if (i < lines.length - 1) {
                this.finishLine();
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

    normalizeLineForComparison(str) {
        if (!str) return '';
        let s = AnsiParser.stripAnsi(str);
        s = s.replace(/\s+/g, ' ').trim();
        s = s.replace(/\s*([>$#])\s*/g, '$1 ');
        s = s.replace(/^P+\s*PS\s+/i, 'PS ');
        s = s.replace(/^P(?=PS|[a-zA-Z]:[\\/]|dir|ls|cd)/i, '');
        return s.trim();
    }

    finishLine() {
        if (this.currentLineEl) {
            const rawText = this.currentLineEl.textContent || '';
            const trimmed = rawText.trim();

            if (trimmed === '') {
                const prev = this.currentLineEl.previousElementSibling;
                if (!prev || prev.textContent.trim() === '') {
                    this.currentLineEl.remove();
                }
            } else {
                const prev = this.currentLineEl.previousElementSibling;
                if (prev) {
                    const normPrev = this.normalizeLineForComparison(prev.textContent);
                    const normCurr = this.normalizeLineForComparison(rawText);

                    // 1. Eliminar duplicados exactos o normalizados consecutivos (ecos de ConPTY/PowerShell/CMD)
                    if (normPrev === normCurr) {
                        this.currentLineEl.remove();
                    }
                    // 2. Si la línea previa era solo un prompt y la actual es ese mismo prompt con el comando
                    else if (normCurr.startsWith(normPrev) && /[>$#]\s*$/.test(normPrev)) {
                        prev.remove();
                    }
                    // 3. Si la línea previa era prompt + comando y la actual es solo el eco del mismo comando sin prompt
                    else if (normPrev.endsWith('> ' + normCurr) || normPrev.endsWith('>' + normCurr) ||
                             normPrev.endsWith('$ ' + normCurr) || normPrev.endsWith('$' + normCurr) ||
                             normPrev.endsWith('# ' + normCurr) || normPrev.endsWith('#' + normCurr)) {
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

    formatDate(rawDate) {
        if (!rawDate || rawDate === 'n/a') return 'n/a';
        const str = String(rawDate).trim();
        if (!str || str === 'n/a') return 'n/a';

        // Detectar formato ISO o Docker: YYYY-MM-DD HH:MM:SS o YYYY-MM-DDTHH:MM:SS
        const match = str.match(/^(\d{4})-(\d{2})-(\d{2})[T\s](\d{2}):(\d{2})(?::(\d{2}))?/);
        if (match) {
            const [_, year, month, day, hour, min] = match;
            const months = ['ene', 'feb', 'mar', 'abr', 'may', 'jun', 'jul', 'ago', 'sep', 'oct', 'nov', 'dic'];
            const mIdx = parseInt(month, 10) - 1;
            const mName = months[mIdx] || month;
            return `${day} ${mName} ${year}, ${hour}:${min}`;
        }

        // Intento con Date estándar
        try {
            const d = new Date(str);
            if (!isNaN(d.getTime())) {
                const day = String(d.getDate()).padStart(2, '0');
                const months = ['ene', 'feb', 'mar', 'abr', 'may', 'jun', 'jul', 'ago', 'sep', 'oct', 'nov', 'dic'];
                const mName = months[d.getMonth()] || '';
                const year = d.getFullYear();
                const hour = String(d.getHours()).padStart(2, '0');
                const min = String(d.getMinutes()).padStart(2, '0');
                return `${day} ${mName} ${year}, ${hour}:${min}`;
            }
        } catch (e) {}

        return str;
    }

    // =========================================================================
    // Gestión de Contenedores Docker en VMs Linux
    // =========================================================================
    async loadDockerCatalog() {
        try {
            const res = await fetch('/api/docker/catalog');
            if (res.ok) {
                this.dockerCatalog = await res.json();
            }
        } catch (e) {
            console.warn('No se pudo cargar el catálogo de Docker:', e);
        }
    }

    async updateContainersVmFilter() {
        if (!this.containersVmFilter && !this.wizVmSelect) return;
        try {
            const res = await fetch('/api/vms');
            if (res.ok) {
                const vms = await res.json();
                this.vms = vms;

                if (this.containersVmFilter) {
                    const currentVal = this.containersVmFilter.value;
                    let opts = ``;
                    vms.forEach((v) => {
                        const isRunning = (v.state || '').toLowerCase() === 'running';
                        const label = `${v.name} (${isRunning ? 'En ejecución' : 'Apagada'})`;
                        opts += `<option value="${this.escapeAttr(v.name)}">${this.escapeHTML(label)}</option>`;
                    });
                    this.containersVmFilter.innerHTML = opts;
                    if (currentVal && vms.some((v) => v.name === currentVal)) {
                        this.containersVmFilter.value = currentVal;
                    }
                }

                if (this.wizVmSelect) {
                    let wOpts = `<option value="" disabled selected>Selecciona una máquina...</option>`;
                    vms.forEach((v) => {
                        wOpts += `<option value="${this.escapeAttr(v.name)}">${this.escapeHTML(v.name)}</option>`;
                    });
                    this.wizVmSelect.innerHTML = wOpts;
                }
            }
        } catch (e) {
            console.warn('No se pudieron listar VMs para el filtro de contenedores:', e);
        }
    }

    async updateImagesVmFilter() {
        if (!this.imagesVmFilter && !this.buildVmSelect) return;
        try {
            const res = await fetch('/api/vms');
            if (res.ok) {
                const vms = await res.json();
                this.vms = vms;

                if (this.imagesVmFilter) {
                    const currentVal = this.imagesVmFilter.value;
                    let opts = ``;
                    vms.forEach((v) => {
                        const isRunning = (v.state || '').toLowerCase() === 'running';
                        const label = `${v.name} (${isRunning ? 'En ejecución' : 'Apagada'})`;
                        opts += `<option value="${this.escapeAttr(v.name)}">${this.escapeHTML(label)}</option>`;
                    });
                    this.imagesVmFilter.innerHTML = opts;
                    if (currentVal && vms.some((v) => v.name === currentVal)) {
                        this.imagesVmFilter.value = currentVal;
                    }
                }

                if (this.buildVmSelect) {
                    const currentBuild = this.buildVmSelect.value;
                    let bOpts = `<option value="" disabled selected>Selecciona una máquina...</option>`;
                    vms.forEach((v) => {
                        bOpts += `<option value="${this.escapeAttr(v.name)}">${this.escapeHTML(v.name)}</option>`;
                    });
                    this.buildVmSelect.innerHTML = bOpts;
                    if (currentBuild && vms.some((v) => v.name === currentBuild)) {
                        this.buildVmSelect.value = currentBuild;
                    }
                }
            }
        } catch (e) {
            console.warn('No se pudieron listar VMs para el filtro de imágenes:', e);
        }
    }

    async updateVolumesVmFilter() {
        if (!this.volumesVmFilter && !this.createVolumeVm) return;
        try {
            const res = await fetch('/api/vms');
            if (res.ok) {
                const vms = await res.json();
                this.vms = vms;

                if (this.volumesVmFilter) {
                    const currentVal = this.volumesVmFilter.value;
                    let opts = ``;
                    vms.forEach((v) => {
                        const isRunning = (v.state || '').toLowerCase() === 'running';
                        const label = `${v.name} (${isRunning ? 'En ejecución' : 'Apagada'})`;
                        opts += `<option value="${this.escapeAttr(v.name)}">${this.escapeHTML(label)}</option>`;
                    });
                    this.volumesVmFilter.innerHTML = opts;
                    if (currentVal && vms.some((v) => v.name === currentVal)) {
                        this.volumesVmFilter.value = currentVal;
                    }
                }

                if (this.createVolumeVm) {
                    const currentBuild = this.createVolumeVm.value;
                    let bOpts = `<option value="" disabled selected>Selecciona una máquina...</option>`;
                    vms.forEach((v) => {
                        bOpts += `<option value="${this.escapeAttr(v.name)}">${this.escapeHTML(v.name)}</option>`;
                    });
                    this.createVolumeVm.innerHTML = bOpts;
                    if (currentBuild && vms.some((v) => v.name === currentBuild)) {
                        this.createVolumeVm.value = currentBuild;
                    }
                }
            }
        } catch (e) {
            console.warn('No se pudieron listar VMs para el filtro de volúmenes:', e);
        }
    }

    async loadVolumes(vmFilter) {
        if (!this.volumesListEl) return;
        this.volumesListEl.innerHTML = `<div class="empty-vm-list"><span class="btn-spinner"></span> Cargando volúmenes...</div>`;

        const filter = (vmFilter || '').trim();
        const url = filter ? `/api/volumes?vm=${encodeURIComponent(filter)}` : '/api/volumes';

        try {
            const res = await fetch(url);
            if (!res.ok) throw new Error('Error al listar volúmenes');
            let vols = await res.json();
            if (!vols) vols = [];
            this.volumes = vols;
            this.renderVolumes();
        } catch (e) {
            console.error('loadVolumes error:', e);
            this.volumesListEl.innerHTML = `<div class="empty-vm-list error">No se pudieron cargar los volúmenes.</div>`;
        }
    }

    renderVolumes() {
        if (!this.volumesListEl) return;
        if (!this.volumes || this.volumes.length === 0) {
            this.volumesListEl.innerHTML = `<div class="empty-vm-list">No hay volúmenes Docker disponibles.</div>`;
            return;
        }

        let html = '';
        this.volumes.forEach((vol) => {
            const isUsed = vol.status === 'En uso';
            const statusClass = isUsed ? 'state-running' : 'state-stopped';

            html += `
                <div class="vm-card">
                    <div class="vm-card-header">
                        <div class="vm-title-row">
                            <h3 class="vm-name">${this.escapeHTML(vol.name)}</h3>
                            <span class="vm-state-badge ${statusClass}">${this.escapeHTML(vol.status || 'Desconocido')}</span>
                        </div>
                        <div class="vm-state-actions">
                            <button class="btn btn-danger btn-icon" onclick="window.terminalApp.promptDeleteVolume('${this.escapeAttr(vol.vmName)}', '${this.escapeAttr(vol.name)}')" title="Eliminar volumen">
                                <svg viewBox="0 0 24 24" width="14" height="14" stroke="currentColor" stroke-width="2" fill="none"><polyline points="3 6 5 6 21 6"></polyline><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"></path></svg>
                            </button>
                        </div>
                    </div>
                    <div class="vm-details-grid">
                        <div class="vm-detail-item">
                            <span class="vm-detail-label">Máquina Virtual</span>
                            <span class="vm-detail-value"><span class="vm-source-tag">${this.escapeHTML(vol.vmName)}</span></span>
                        </div>
                        <div class="vm-detail-item">
                            <span class="vm-detail-label">Driver</span>
                            <span class="vm-detail-value">${this.escapeHTML(vol.driver)}</span>
                        </div>
                        <div class="vm-detail-item" style="grid-column: 1 / -1;">
                            <span class="vm-detail-label">Punto de montaje</span>
                            <span class="vm-detail-value" style="font-family: monospace; font-size: 0.85rem; word-break: break-all;">${this.escapeHTML(vol.mountpoint)}</span>
                        </div>
                    </div>
                </div>
            `;
        });
        this.volumesListEl.innerHTML = html;
    }

    async fetchVmVolumes(vmName) {
        if (!vmName) return [];
        try {
            const res = await fetch(`/api/volumes?vm=${encodeURIComponent(vmName)}`);
            if (res.ok) {
                const vols = await res.json();
                return Array.isArray(vols) ? vols : [];
            }
        } catch (e) {
            console.warn('Error obteniendo volúmenes de VM:', e);
        }
        return [];
    }

    async submitCreateVolume() {
        const vmName = this.createVolumeVm.value;
        const name = this.createVolumeName.value.trim();
        const driver = 'local';

        if (!vmName || !name) return;

        this.submitCreateVolumeBtn.disabled = true;
        this.submitCreateVolumeBtn.querySelector('.btn-spinner').classList.remove('hidden');
        this.submitCreateVolumeBtn.querySelector('.btn-text').textContent = 'Creando...';
        this.createVolumeFeedback.classList.add('hidden');

        try {
            const res = await fetch('/api/volumes', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ vmName, name, driver }),
            });
            const data = await res.json();
            if (!res.ok) throw new Error(data.error || 'Error creando volumen');

            this.createVolumeModal.classList.add('hidden');
            this.createVolumeForm.reset();
            this.loadVolumes(this.volumesVmFilter ? this.volumesVmFilter.value : '');

            if (typeof this._onVolumeCreatedCallback === 'function') {
                const cb = this._onVolumeCreatedCallback;
                this._onVolumeCreatedCallback = null;
                cb(name, vmName);
            }
        } catch (e) {
            this.createVolumeFeedback.textContent = e.message;
            this.createVolumeFeedback.className = 'modal-feedback error';
            this.createVolumeFeedback.classList.remove('hidden');
        } finally {
            this.submitCreateVolumeBtn.disabled = false;
            this.submitCreateVolumeBtn.querySelector('.btn-spinner').classList.add('hidden');
            this.submitCreateVolumeBtn.querySelector('.btn-text').textContent = 'Crear volumen';
        }
    }

    promptDeleteVolume(vmName, volumeName) {
        this.pendingDeleteVolume = { vmName, volumeName };
        this.deleteVolumeNameEl.textContent = volumeName;
        this.deleteVolumeVmTag.textContent = vmName;
        this.deleteVolumeFeedback.classList.add('hidden');
        this.deleteVolumeModal.classList.remove('hidden');
    }

    closeDeleteVolumeModal() {
        this.deleteVolumeModal.classList.add('hidden');
        this.pendingDeleteVolume = null;
    }

    async executeDeleteVolume() {
        if (!this.pendingDeleteVolume) return;
        const { vmName, volumeName } = this.pendingDeleteVolume;

        this.confirmDeleteVolumeBtn.disabled = true;
        this.confirmDeleteVolumeBtn.querySelector('.btn-spinner').classList.remove('hidden');

        try {
            const res = await fetch(`/api/volumes/${encodeURIComponent(vmName)}/${encodeURIComponent(volumeName)}`, {
                method: 'DELETE',
            });
            const data = await res.json();
            if (!res.ok) throw new Error(data.error || 'Error al eliminar');

            this.closeDeleteVolumeModal();
            this.loadVolumes(this.volumesVmFilter ? this.volumesVmFilter.value : '');
        } catch (e) {
            this.deleteVolumeFeedback.textContent = e.message;
            this.deleteVolumeFeedback.className = 'modal-feedback error';
            this.deleteVolumeFeedback.classList.remove('hidden');
        } finally {
            this.confirmDeleteVolumeBtn.disabled = false;
            this.confirmDeleteVolumeBtn.querySelector('.btn-spinner').classList.add('hidden');
        }
    }

    // =========================================================================
    // Gestión de Redes Docker
    // =========================================================================
    async updateNetworksVmFilter() {
        if (!this.networksVmFilter) return;
        try {
            const currentFilter = this.networksVmFilter.value;
            const res = await fetch('/api/vms');
            if (res.ok) {
                const vms = await res.json();
                let opts = '<option value="">Todas las máquinas virtuales</option>';
                if (Array.isArray(vms)) {
                    vms.forEach((vm) => {
                        opts += `<option value="${this.escapeAttr(vm.name)}">${this.escapeHTML(vm.name)}</option>`;
                    });
                }
                this.networksVmFilter.innerHTML = opts;
                if (currentFilter && Array.isArray(vms) && vms.some((v) => v.name === currentFilter)) {
                    this.networksVmFilter.value = currentFilter;
                }
            }
        } catch (e) {
            console.warn('No se pudieron listar VMs para el filtro de redes:', e);
        }
    }

    async loadNetworks(vmFilter) {
        if (!this.networksListEl) return;
        this.networksListEl.innerHTML = `<div class="empty-vm-list"><span class="btn-spinner"></span> Cargando redes Docker...</div>`;

        const filter = (vmFilter || '').trim();
        const url = filter ? `/api/networks?vm=${encodeURIComponent(filter)}` : '/api/networks';

        try {
            const res = await fetch(url);
            if (!res.ok) throw new Error('Error al listar redes');
            let nets = await res.json();
            if (!nets) nets = [];
            this.networks = nets;
            this.renderNetworks();
        } catch (e) {
            console.error('loadNetworks error:', e);
            this.networksListEl.innerHTML = `<div class="empty-vm-list error">No se pudieron cargar las redes.</div>`;
        }
    }

    renderNetworks() {
        if (!this.networksListEl) return;
        if (!this.networks || this.networks.length === 0) {
            this.networksListEl.innerHTML = `<div class="empty-vm-list">No hay redes Docker disponibles.</div>`;
            return;
        }

        let html = '';
        this.networks.forEach((net) => {
            const isDefault = ['bridge', 'host', 'none'].includes(net.name.toLowerCase());
            const shortId = net.id ? net.id.substring(0, 12) : '-';
            const containersCount = typeof net.containers === 'number' ? net.containers : 0;
            const subnetText = net.subnet ? net.subnet : (isDefault ? 'Automática por Docker' : 'No configurada');

            html += `
                <div class="vm-card" data-net-name="${this.escapeAttr(net.name)}" data-vm="${this.escapeAttr(net.vmName)}">
                    <div class="vm-card-header">
                        <div class="vm-card-title-group">
                            <span class="vm-name">${this.escapeHTML(net.name)}</span>
                            <span class="vm-source-tag">${this.escapeHTML(net.vmName)}</span>
                            ${isDefault ? '<span class="status-pill status-pill-active" style="font-size: 10px; padding: 2px 6px;">Predeterminada</span>' : ''}
                        </div>
                        <div class="vm-card-actions">
                            <button class="btn btn-secondary btn-sm" onclick="window.terminalApp.showNetworkDetails('${this.escapeAttr(net.vmName)}', '${this.escapeAttr(net.name)}')">
                                Detalles
                            </button>
                            ${!isDefault ? `
                                <button class="btn btn-danger btn-sm" onclick="window.terminalApp.promptDeleteNetwork('${this.escapeAttr(net.vmName)}', '${this.escapeAttr(net.name)}')">
                                    Eliminar
                                </button>
                            ` : ''}
                        </div>
                    </div>
                    <div class="vm-details-grid">
                        <div class="vm-detail-item">
                            <span class="vm-detail-label">ID</span>
                            <span class="vm-detail-value font-mono">${this.escapeHTML(shortId)}</span>
                        </div>
                        <div class="vm-detail-item">
                            <span class="vm-detail-label">Driver</span>
                            <span class="vm-detail-value">${this.escapeHTML(net.driver || 'bridge')}</span>
                        </div>
                        <div class="vm-detail-item">
                            <span class="vm-detail-label">Scope</span>
                            <span class="vm-detail-value">${this.escapeHTML(net.scope || 'local')}</span>
                        </div>
                        <div class="vm-detail-item">
                            <span class="vm-detail-label">Contenedores</span>
                            <span class="vm-detail-value font-mono">${containersCount}</span>
                        </div>
                        <div class="vm-detail-item" style="grid-column: 1 / -1;">
                            <span class="vm-detail-label">Subred</span>
                            <span class="vm-detail-value font-mono" style="font-size: 0.85rem;">${this.escapeHTML(subnetText)}</span>
                        </div>
                    </div>
                </div>
            `;
        });
        this.networksListEl.innerHTML = html;
    }

    async fetchVmNetworks(vmName) {
        if (!vmName) return [];
        try {
            const res = await fetch(`/api/networks?vm=${encodeURIComponent(vmName)}`);
            if (res.ok) {
                const nets = await res.json();
                return Array.isArray(nets) ? nets : [];
            }
        } catch (e) {
            console.warn('Error obteniendo redes de VM:', e);
        }
        return [];
    }

    openCreateNetworkDialog(presetVm = '', onCreatedCallback = null) {
        if (this.createNetworkForm) this.createNetworkForm.reset();
        if (this.createNetworkFeedback) this.createNetworkFeedback.classList.add('hidden');

        // Llenar select de VM
        if (this.createNetworkVm) {
            this.createNetworkVm.innerHTML = '<option value="" disabled selected>Selecciona una máquina...</option>';
            if (this.vms) {
                this.vms.forEach((vm) => {
                    const opt = document.createElement('option');
                    const vName = vm.name || vm.Name;
                    opt.value = vName;
                    opt.textContent = vName;
                    this.createNetworkVm.appendChild(opt);
                });
            }
            const activeVm = presetVm || (this.networksVmFilter ? this.networksVmFilter.value : '');
            if (activeVm) {
                this.createNetworkVm.value = activeVm;
            }
        }

        this._onNetworkCreatedCallback = onCreatedCallback;
        if (this.createNetworkModal) {
            this.createNetworkModal.classList.remove('hidden');
            setTimeout(() => {
                if (this.createNetworkName) this.createNetworkName.focus();
            }, 50);
        }
    }

    async submitCreateNetwork() {
        const vmName = this.createNetworkVm.value;
        const name = this.createNetworkName.value.trim();
        const driver = this.createNetworkDriver ? this.createNetworkDriver.value : 'bridge';

        if (!vmName || !name) return;

        this.submitCreateNetworkBtn.disabled = true;
        this.submitCreateNetworkBtn.querySelector('.btn-spinner').classList.remove('hidden');
        this.submitCreateNetworkBtn.querySelector('.btn-text').textContent = 'Creando...';
        this.createNetworkFeedback.classList.add('hidden');

        try {
            const res = await fetch('/api/networks', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    vmName,
                    name,
                    driver,
                }),
            });
            const data = await res.json();
            if (!res.ok) throw new Error(data.error || 'Error creando red');

            this.createNetworkModal.classList.add('hidden');
            this.createNetworkForm.reset();
            this.loadNetworks(this.networksVmFilter ? this.networksVmFilter.value : '');

            if (typeof this._onNetworkCreatedCallback === 'function') {
                const cb = this._onNetworkCreatedCallback;
                this._onNetworkCreatedCallback = null;
                cb(name, vmName);
            }
        } catch (e) {
            this.createNetworkFeedback.textContent = e.message;
            this.createNetworkFeedback.className = 'modal-feedback error';
            this.createNetworkFeedback.classList.remove('hidden');
        } finally {
            this.submitCreateNetworkBtn.disabled = false;
            this.submitCreateNetworkBtn.querySelector('.btn-spinner').classList.add('hidden');
            this.submitCreateNetworkBtn.querySelector('.btn-text').textContent = 'Crear red';
        }
    }

    async showNetworkDetails(vmName, networkName) {
        if (!this.networkDetailsModal || !this.networkDetailsBody) return;
        this.networkDetailsBody.innerHTML = `<div class="empty-vm-list"><span class="btn-spinner"></span> Cargando detalles...</div>`;
        this.networkDetailsModal.classList.remove('hidden');

        try {
            const res = await fetch(`/api/networks/${encodeURIComponent(vmName)}/${encodeURIComponent(networkName)}`);
            if (!res.ok) {
                const errData = await res.text();
                throw new Error(errData || 'Error consultando detalles de la red');
            }
            const net = await res.json();

            let containersHtml = '<span style="color: var(--text-muted); font-size: 12px;">Sin contenedores conectados</span>';
            if (net.containerList && net.containerList.length > 0) {
                containersHtml = `<div style="display:flex; flex-wrap:wrap; gap:6px; margin-top:4px;">` +
                    net.containerList.map((c) => `<span class="badge" style="background: rgba(255,255,255,0.06); padding: 3px 8px; border-radius: 4px; font-family: var(--font-mono); font-size: 11px;">${this.escapeHTML(c)}</span>`).join('') +
                    `</div>`;
            }

            this.networkDetailsBody.innerHTML = `
                <div class="vm-details-grid" style="grid-template-columns: 1fr 1fr; gap: 12px;">
                    <div class="vm-detail-item">
                        <span class="vm-detail-label">Nombre</span>
                        <span class="vm-detail-value" style="font-weight: 600;">${this.escapeHTML(net.name)}</span>
                    </div>
                    <div class="vm-detail-item">
                        <span class="vm-detail-label">Máquina Virtual</span>
                        <span class="vm-detail-value">${this.escapeHTML(net.vmName || vmName)}</span>
                    </div>
                    <div class="vm-detail-item" style="grid-column: 1 / -1;">
                        <span class="vm-detail-label">ID Completo</span>
                        <span class="vm-detail-value font-mono" style="font-size: 11px; word-break: break-all;">${this.escapeHTML(net.id || '-')}</span>
                    </div>
                    <div class="vm-detail-item">
                        <span class="vm-detail-label">Driver</span>
                        <span class="vm-detail-value">${this.escapeHTML(net.driver || 'bridge')}</span>
                    </div>
                    <div class="vm-detail-item">
                        <span class="vm-detail-label">Scope</span>
                        <span class="vm-detail-value">${this.escapeHTML(net.scope || 'local')}</span>
                    </div>
                    <div class="vm-detail-item">
                        <span class="vm-detail-label">Subred (CIDR)</span>
                        <span class="vm-detail-value font-mono">${this.escapeHTML(net.subnet || 'No configurada')}</span>
                    </div>
                    <div class="vm-detail-item">
                        <span class="vm-detail-label">Gateway</span>
                        <span class="vm-detail-value font-mono">${this.escapeHTML(net.gateway || 'No configurado')}</span>
                    </div>
                    ${net.ipRange ? `
                        <div class="vm-detail-item" style="grid-column: 1 / -1;">
                            <span class="vm-detail-label">Rango IP</span>
                            <span class="vm-detail-value font-mono">${this.escapeHTML(net.ipRange)}</span>
                        </div>
                    ` : ''}
                    <div class="vm-detail-item">
                        <span class="vm-detail-label">Interna (sin internet)</span>
                        <span class="vm-detail-value">${net.internal ? 'Sí' : 'No'}</span>
                    </div>
                    <div class="vm-detail-item">
                        <span class="vm-detail-label">Attachable</span>
                        <span class="vm-detail-value">${net.attachable ? 'Sí' : 'No'}</span>
                    </div>
                    <div class="vm-detail-item" style="grid-column: 1 / -1; margin-top: 6px;">
                        <span class="vm-detail-label">Contenedores Conectados (${net.containers || 0})</span>
                        ${containersHtml}
                    </div>
                </div>
            `;
        } catch (e) {
            this.networkDetailsBody.innerHTML = `<div class="empty-vm-list error">${this.escapeHTML(e.message)}</div>`;
        }
    }

    promptDeleteNetwork(vmName, networkName) {
        if (['bridge', 'host', 'none'].includes(networkName.toLowerCase())) {
            alert(`No está permitido eliminar las redes predeterminadas ('${networkName}').`);
            return;
        }
        this.pendingDeleteNetwork = { vmName, networkName };
        this.deleteNetworkNameEl.textContent = networkName;
        this.deleteNetworkVmTag.textContent = vmName;
        this.deleteNetworkFeedback.classList.add('hidden');
        this.deleteNetworkModal.classList.remove('hidden');
    }

    closeDeleteNetworkModal() {
        this.deleteNetworkModal.classList.add('hidden');
        this.pendingDeleteNetwork = null;
    }

    async executeDeleteNetwork() {
        if (!this.pendingDeleteNetwork) return;
        const { vmName, networkName } = this.pendingDeleteNetwork;

        this.confirmDeleteNetworkBtn.disabled = true;
        this.confirmDeleteNetworkBtn.querySelector('.btn-spinner').classList.remove('hidden');

        try {
            const res = await fetch(`/api/networks/${encodeURIComponent(vmName)}/${encodeURIComponent(networkName)}`, {
                method: 'DELETE',
            });
            const data = await res.json();
            if (!res.ok) throw new Error(data.error || 'Error al eliminar red');

            this.closeDeleteNetworkModal();
            this.loadNetworks(this.networksVmFilter ? this.networksVmFilter.value : '');
        } catch (e) {
            this.deleteNetworkFeedback.textContent = e.message;
            this.deleteNetworkFeedback.className = 'modal-feedback error';
            this.deleteNetworkFeedback.classList.remove('hidden');
        } finally {
            this.confirmDeleteNetworkBtn.disabled = false;
            this.confirmDeleteNetworkBtn.querySelector('.btn-spinner').classList.add('hidden');
        }
    }

    async updateWizardNetworks(vmName, selectNetworkName = '') {
        if (!this.wizNetworkSelect) return;
        this.wizNetworkSelect.innerHTML = '<option value="">Predeterminada (bridge)</option><option value="" disabled>Cargando redes...</option>';
        if (!vmName) {
            this.wizNetworkSelect.innerHTML = '<option value="">Predeterminada (bridge)</option>';
            return;
        }

        try {
            const nets = await this.fetchVmNetworks(vmName);
            let opts = '<option value="">Predeterminada (bridge)</option>';
            if (Array.isArray(nets)) {
                nets.forEach((n) => {
                    const isSelected = selectNetworkName && (n.name === selectNetworkName);
                    opts += `<option value="${this.escapeAttr(n.name)}" ${isSelected ? 'selected' : ''}>${this.escapeHTML(n.name)} (${this.escapeHTML(n.driver)})</option>`;
                });
            }
            this.wizNetworkSelect.innerHTML = opts;
            if (selectNetworkName) {
                this.wizNetworkSelect.value = selectNetworkName;
            }
        } catch (e) {
            console.warn('Error actualizando selector de redes del wizard:', e);
            this.wizNetworkSelect.innerHTML = '<option value="">Predeterminada (bridge)</option>';
        }
    }

    async loadContainers(vmFilter) {
        if (!this.containersListEl) return;
        this.containersListEl.innerHTML = `<div class="empty-vm-list"><span class="btn-spinner"></span> Cargando contenedores...</div>`;

        const filter = (vmFilter || '').trim();
        const url = filter ? `/api/containers?vm=${encodeURIComponent(filter)}` : '/api/containers';

        if (filter && this.dockerStatusPill) {
            try {
                fetch(`/api/vms/${encodeURIComponent(filter)}/docker/status`)
                    .then((r) => r.json())
                    .then((st) => {
                        if (st && st.available) {
                            this.dockerStatusPill.className = 'docker-status-pill';
                            this.dockerStatusPill.textContent = `Docker Activo (${st.version || 'OK'})`;
                            this.dockerStatusPill.classList.remove('hidden');
                        } else {
                            this.dockerStatusPill.className = 'docker-status-pill inactive';
                            this.dockerStatusPill.textContent = 'Docker No disponible';
                            this.dockerStatusPill.classList.remove('hidden');
                        }
                    })
                    .catch(() => {
                        this.dockerStatusPill.classList.add('hidden');
                    });
            } catch (e) {
                this.dockerStatusPill.classList.add('hidden');
            }
        } else if (this.dockerStatusPill) {
            this.dockerStatusPill.classList.add('hidden');
        }

        try {
            const res = await fetch(url);
            if (!res.ok) {
                const text = await res.text();
                this.containersListEl.innerHTML = `<div class="empty-vm-list" style="color:var(--accent-red)">Error al consultar contenedores: ${this.escapeHTML(this.parseApiError(text))}</div>`;
                return;
            }
            const containers = await res.json();
            this.containers = containers || [];
            this.renderContainers(this.containers);
        } catch (e) {
            console.warn('No se pudieron cargar contenedores:', e);
            this.containersListEl.innerHTML = `<div class="empty-vm-list" style="color:var(--accent-red)">No se pudo conectar al servidor para listar contenedores.</div>`;
        }
    }

    renderContainers(containers) {
        if (!this.containersListEl) return;
        if (!containers || containers.length === 0) {
            this.containersListEl.innerHTML = `
                <div class="empty-vm-list" style="padding:32px 16px;">
                    <p style="margin-bottom:12px;">Sin contenedores en esta máquina virtual.</p>
                    <button type="button" class="btn btn-secondary btn-sm" onclick="document.getElementById('open-create-container-btn').click()">
                        Nuevo contenedor
                    </button>
                </div>
            `;
            return;
        }

        this.containersListEl.innerHTML = containers
            .map((c) => {
                const isRunning = (c.state || '').toLowerCase() === 'running';
                const stateClass = isRunning
                    ? 'state-running'
                    : (c.state || '').toLowerCase().includes('exit')
                    ? 'state-exited'
                    : 'state-created';

                return `
                <div class="container-card" data-vm="${this.escapeAttr(c.vmName)}" data-id="${this.escapeAttr(c.id)}">
                    <div class="container-head">
                        <div class="container-title-group">
                            <span class="container-name">${this.escapeHTML(c.name || c.id)}</span>
                            <span class="vm-badge" title="Máquina Virtual">${this.escapeHTML(c.vmName || 'VM')}</span>
                        </div>
                        <span class="container-state-pill ${stateClass}">${this.escapeHTML(c.state || 'desconocido')}</span>
                    </div>
                    <div class="container-meta">
                        <span class="container-image" title="Imagen Docker">${this.escapeHTML(c.image || '')}</span>
                        ${c.ports ? `<span class="container-ports-badge" title="Puertos">${this.escapeHTML(c.ports)}</span>` : ''}
                        <span class="container-status-text" title="Estado">${this.escapeHTML(c.status || '')}</span>
                    </div>
                    <div class="container-actions">
                        ${
                            isRunning
                                ? `
                            <button type="button" class="btn btn-secondary btn-sm" title="Consola interactiva" onclick="window.terminalApp.connectContainerTerminal('${this.escapeAttr(c.vmName)}', '${this.escapeAttr(c.name || c.id)}')">
                                Consola
                            </button>
                            <button type="button" class="btn btn-secondary btn-sm" title="Reiniciar contenedor" onclick="window.terminalApp.restartContainer('${this.escapeAttr(c.vmName)}', '${this.escapeAttr(c.id)}')">
                                Reiniciar
                            </button>
                            <button type="button" class="btn btn-secondary btn-sm" title="Detener contenedor" onclick="window.terminalApp.stopContainer('${this.escapeAttr(c.vmName)}', '${this.escapeAttr(c.id)}')">
                                Detener
                            </button>
                        `
                                : `
                            <button type="button" class="btn btn-primary btn-sm" title="Iniciar contenedor" onclick="window.terminalApp.startContainer('${this.escapeAttr(c.vmName)}', '${this.escapeAttr(c.id)}')">
                                Iniciar
                            </button>
                        `
                        }
                        <button type="button" class="btn btn-ghost btn-sm" title="Ver logs" onclick="window.terminalApp.openContainerLogs('${this.escapeAttr(c.vmName)}', '${this.escapeAttr(c.id)}', '${this.escapeAttr(c.name || c.id)}')">
                            Logs
                        </button>
                        <button type="button" class="btn btn-ghost btn-sm" title="Ver detalles" onclick="window.terminalApp.openContainerDetails('${this.escapeAttr(c.vmName)}', '${this.escapeAttr(c.id)}')">
                            Detalles
                        </button>
                        <button type="button" class="btn btn-secondary btn-sm" title="Crear imagen a partir de este contenedor" onclick="window.terminalApp.openCommitContainerModal('${this.escapeAttr(c.vmName)}', '${this.escapeAttr(c.id)}', '${this.escapeAttr(c.name || c.id)}')">
                            Crear imagen
                        </button>
                        <button type="button" class="btn btn-danger btn-sm" title="Eliminar contenedor" onclick="window.terminalApp.openDeleteContainerModal('${this.escapeAttr(c.vmName)}', '${this.escapeAttr(c.id)}', '${this.escapeAttr(c.name || c.id)}')">
                            Eliminar
                        </button>
                    </div>
                </div>
            `;
            })
            .join('');
    }

    connectContainerTerminal(vmName, containerIdOrName) {
        if (!vmName || !containerIdOrName) return;
        this.clearOutput();
        this.appendInfoLine(`[Iniciando consola interactiva en contenedor '${containerIdOrName}' (${vmName})...]`);

        if (this.targetUserHostEl) {
            this.targetUserHostEl.textContent = `${vmName}:${containerIdOrName}`;
        }
        if (this.targetShellBadgeEl) {
            this.targetShellBadgeEl.textContent = 'docker';
            this.targetShellBadgeEl.className = 'badge-shell';
        }
        this.currentPath = `/${containerIdOrName}`;
        this.updateCurrentPath(this.currentPath);

        this.toggleSidebar(false);

        const params = {
            vm: vmName,
            container: containerIdOrName,
        };
        this.ws.reconnectWith(params);
    }

    async startContainer(vmName, containerId) {
        try {
            const res = await fetch(`/api/containers/${encodeURIComponent(vmName)}/${encodeURIComponent(containerId)}/start`, {
                method: 'POST',
            });
            if (res.ok) {
                this.loadContainers(this.containersVmFilter ? this.containersVmFilter.value : '');
            } else {
                const text = await res.text();
                alert(`Error iniciando contenedor: ${this.parseApiError(text)}`);
            }
        } catch (e) {
            alert(`Fallo de conexión al iniciar contenedor: ${e.message}`);
        }
    }

    async stopContainer(vmName, containerId) {
        try {
            const res = await fetch(`/api/containers/${encodeURIComponent(vmName)}/${encodeURIComponent(containerId)}/stop`, {
                method: 'POST',
            });
            if (res.ok) {
                this.loadContainers(this.containersVmFilter ? this.containersVmFilter.value : '');
            } else {
                const text = await res.text();
                alert(`Error deteniendo contenedor: ${this.parseApiError(text)}`);
            }
        } catch (e) {
            alert(`Fallo de conexión al detener contenedor: ${e.message}`);
        }
    }

    async restartContainer(vmName, containerId) {
        try {
            const res = await fetch(`/api/containers/${encodeURIComponent(vmName)}/${encodeURIComponent(containerId)}/restart`, {
                method: 'POST',
            });
            if (res.ok) {
                this.loadContainers(this.containersVmFilter ? this.containersVmFilter.value : '');
            } else {
                const text = await res.text();
                alert(`Error reiniciando contenedor: ${this.parseApiError(text)}`);
            }
        } catch (e) {
            alert(`Fallo de conexión al reiniciar contenedor: ${e.message}`);
        }
    }

    openDeleteContainerModal(vmName, containerId, containerName) {
        this.pendingDeleteContainer = { vm: vmName, id: containerId, name: containerName };
        if (this.deleteContainerNameEl) {
            this.deleteContainerNameEl.textContent = containerName || containerId;
        }
        if (this.deleteContainerVmTag) {
            this.deleteContainerVmTag.textContent = vmName;
        }
        if (this.deleteContainerForceCheckbox) {
            this.deleteContainerForceCheckbox.checked = false;
        }
        if (this.deleteContainerFeedback) {
            this.deleteContainerFeedback.className = 'modal-feedback hidden';
            this.deleteContainerFeedback.textContent = '';
        }
        if (this.deleteContainerModal) {
            this.deleteContainerModal.classList.remove('hidden');
        }
    }

    closeDeleteContainerModal() {
        if (this.deleteContainerModal) {
            this.deleteContainerModal.classList.add('hidden');
        }
        this.pendingDeleteContainer = null;
    }

    async executeDeleteContainer() {
        if (!this.pendingDeleteContainer) return;
        const { vm, id } = this.pendingDeleteContainer;
        const force = this.deleteContainerForceCheckbox && this.deleteContainerForceCheckbox.checked;

        if (this.confirmDeleteContainerBtn) {
            this.confirmDeleteContainerBtn.disabled = true;
            const spinner = this.confirmDeleteContainerBtn.querySelector('.btn-spinner');
            if (spinner) spinner.classList.remove('hidden');
        }

        try {
            const url = `/api/containers/${encodeURIComponent(vm)}/${encodeURIComponent(id)}${force ? '?force=true' : ''}`;
            const res = await fetch(url, { method: 'DELETE' });
            if (res.ok) {
                this.closeDeleteContainerModal();
                this.loadContainers(this.containersVmFilter ? this.containersVmFilter.value : '');
            } else {
                const text = await res.text();
                if (this.deleteContainerFeedback) {
                    this.deleteContainerFeedback.className = 'modal-feedback error';
                    this.deleteContainerFeedback.textContent = this.parseApiError(text);
                }
            }
        } catch (e) {
            if (this.deleteContainerFeedback) {
                this.deleteContainerFeedback.className = 'modal-feedback error';
                this.deleteContainerFeedback.textContent = e.message;
            }
        } finally {
            if (this.confirmDeleteContainerBtn) {
                this.confirmDeleteContainerBtn.disabled = false;
                const spinner = this.confirmDeleteContainerBtn.querySelector('.btn-spinner');
                if (spinner) spinner.classList.add('hidden');
            }
        }
    }

    openCommitContainerModal(vmName, containerId, containerName) {
        this.currentCommitVm = vmName;
        this.currentCommitContainerId = containerId;

        if (this.commitContainerTargetName) {
            this.commitContainerTargetName.textContent = containerName || containerId;
        }
        if (this.commitContainerVmTag) {
            this.commitContainerVmTag.textContent = vmName;
        }

        if (this.commitImageNameInput) {
            const cleanName = (containerName || 'app')
                .replace(/^\//, '')
                .replace(/[^a-zA-Z0-9_.-]/g, '-')
                .toLowerCase();
            this.commitImageNameInput.value = `${cleanName}-img`;
        }
        if (this.commitImageTagInput) {
            this.commitImageTagInput.value = 'latest';
        }
        if (this.commitImageCommentInput) {
            this.commitImageCommentInput.value = '';
        }

        this.hideCommitContainerFeedback();

        if (this.commitContainerModal) {
            this.commitContainerModal.classList.remove('hidden');
        }

        if (this.commitImageNameInput) {
            setTimeout(() => this.commitImageNameInput.focus(), 60);
        }
    }

    closeCommitContainerModal() {
        if (this.commitContainerModal) {
            this.commitContainerModal.classList.add('hidden');
        }
        this.hideCommitContainerFeedback();
        this.currentCommitVm = '';
        this.currentCommitContainerId = '';
    }

    showCommitContainerFeedback(msg, type = 'error') {
        if (!this.commitContainerFeedback) return;
        this.commitContainerFeedback.className = `modal-feedback ${type}`;
        this.commitContainerFeedback.textContent = msg;
        this.commitContainerFeedback.classList.remove('hidden');
    }

    hideCommitContainerFeedback() {
        if (!this.commitContainerFeedback) return;
        this.commitContainerFeedback.className = 'modal-feedback hidden';
        this.commitContainerFeedback.textContent = '';
    }

    async submitCommitContainer() {
        const vmName = this.currentCommitVm;
        const containerId = this.currentCommitContainerId;
        const imageName = this.commitImageNameInput ? this.commitImageNameInput.value.trim() : '';
        const tag = this.commitImageTagInput ? this.commitImageTagInput.value.trim() : 'latest';
        const comment = this.commitImageCommentInput ? this.commitImageCommentInput.value.trim() : '';

        if (!vmName || !containerId) {
            this.showCommitContainerFeedback('No se pudo identificar el contenedor origen.', 'error');
            return;
        }
        if (!imageName) {
            this.showCommitContainerFeedback('Debes ingresar un nombre para la nueva imagen.', 'error');
            return;
        }

        this.setButtonLoading(this.submitCommitContainerBtn, true);
        this.hideCommitContainerFeedback();

        try {
            const url = `/api/containers/${encodeURIComponent(vmName)}/${encodeURIComponent(containerId)}/commit`;
            const payload = {
                imageName,
                tag: tag || 'latest',
                comment,
            };

            const res = await fetch(url, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(payload),
            });

            if (res.ok) {
                const data = await res.json();
                this.closeCommitContainerModal();
                this.appendInfoLine(`[Imagen '${data.image || imageName}' guardada exitosamente en la VM '${vmName}']`);

                // Actualizar la lista de imágenes de la VM para que se refleje inmediatamente
                if (this.imagesVmFilter) {
                    this.imagesVmFilter.value = vmName;
                }
                this.loadImages(vmName);
            } else {
                const text = await res.text();
                this.showCommitContainerFeedback(`Error al crear imagen: ${this.parseApiError(text)}`, 'error');
            }
        } catch (e) {
            this.showCommitContainerFeedback(`Error de conexión: ${e.message}`, 'error');
        } finally {
            this.setButtonLoading(this.submitCommitContainerBtn, false);
        }
    }

    openContainerLogs(vmName, containerId, containerName) {
        this.activeLogsTarget = { vm: vmName, id: containerId, name: containerName };
        if (this.containerLogsTitle) {
            this.containerLogsTitle.textContent = `Logs: ${containerName || containerId}`;
        }
        if (this.containerLogsSubtitle) {
            this.containerLogsSubtitle.textContent = `VM: ${vmName} (${containerId})`;
        }
        if (this.containerLogsModal) {
            this.containerLogsModal.classList.remove('hidden');
        }
        this.refreshContainerLogs();
    }

    closeContainerLogs() {
        if (this.containerLogsModal) {
            this.containerLogsModal.classList.add('hidden');
        }
        this.activeLogsTarget = null;
    }

    async refreshContainerLogs() {
        if (!this.activeLogsTarget || !this.containerLogsOutput) return;
        const { vm, id } = this.activeLogsTarget;
        const tail = this.containerLogsTail ? this.containerLogsTail.value : 200;

        this.containerLogsOutput.textContent = 'Cargando logs...';
        try {
            const res = await fetch(`/api/containers/${encodeURIComponent(vm)}/${encodeURIComponent(id)}/logs?tail=${tail}`);
            if (res.ok) {
                const data = await res.json();
                this.containerLogsOutput.textContent = data.logs || 'Sin registros emitidos por el contenedor.';
                this.containerLogsOutput.scrollTop = this.containerLogsOutput.scrollHeight;
            } else {
                const text = await res.text();
                this.containerLogsOutput.textContent = `Error obteniendo logs: ${this.parseApiError(text)}`;
            }
        } catch (e) {
            this.containerLogsOutput.textContent = `Fallo de conexión al cargar logs: ${e.message}`;
        }
    }

    async openContainerDetails(vmName, containerId) {
        if (!this.containerDetailsModal || !this.containerDetailsContent) return;
        this.containerDetailsContent.innerHTML = `<div class="empty-vm-list"><span class="btn-spinner"></span> Obteniendo inspección...</div>`;
        this.containerDetailsModal.classList.remove('hidden');

        try {
            const res = await fetch(`/api/containers/${encodeURIComponent(vmName)}/${encodeURIComponent(containerId)}`);
            if (!res.ok) {
                const text = await res.text();
                this.containerDetailsContent.innerHTML = `<div class="empty-vm-list" style="color:var(--accent-red)">Error: ${this.escapeHTML(this.parseApiError(text))}</div>`;
                return;
            }
            const info = await res.json();
            this.renderContainerDetails(info);
        } catch (e) {
            this.containerDetailsContent.innerHTML = `<div class="empty-vm-list" style="color:var(--accent-red)">Error: ${this.escapeHTML(e.message)}</div>`;
        }
    }

    closeContainerDetails() {
        if (this.containerDetailsModal) {
            this.containerDetailsModal.classList.add('hidden');
        }
    }

    renderContainerDetails(info) {
        if (!this.containerDetailsContent || !info) return;

        const portsStr = (info.ports || []).map((p) => `${p.hostPort}:${p.containerPort}/${p.protocol || 'tcp'}`).join(', ') || 'Ninguno';
        const mountsStr = (info.mounts || []).map((m) => `${m.hostPath} &rarr; ${m.containerPath}`).join('<br>') || 'Ninguno';
        const envStr = (info.env || []).join('<br>') || 'Ninguna';

        this.containerDetailsContent.innerHTML = `
            <div class="vm-detail-item">
                <span class="vm-detail-label">Nombre</span>
                <span class="vm-detail-val highlight">${this.escapeHTML(info.name || '')}</span>
            </div>
            <div class="vm-detail-item">
                <span class="vm-detail-label">ID Contenedor</span>
                <span class="vm-detail-val">${this.escapeHTML(info.id || '')}</span>
            </div>
            <div class="vm-detail-item">
                <span class="vm-detail-label">Máquina Virtual</span>
                <span class="vm-detail-val">${this.escapeHTML(info.vmName || '')}</span>
            </div>
            <div class="vm-detail-item">
                <span class="vm-detail-label">Imagen</span>
                <span class="vm-detail-val">${this.escapeHTML(info.image || '')}</span>
            </div>
            <div class="vm-detail-item">
                <span class="vm-detail-label">Estado</span>
                <span class="vm-detail-val">${this.escapeHTML(info.state ? info.state.status : '')}</span>
            </div>
            <div class="vm-detail-item">
                <span class="vm-detail-label">Dirección IP en VM</span>
                <span class="vm-detail-val">${this.escapeHTML(info.ipAddress || 'Host bridge')}</span>
            </div>
            <div class="vm-detail-item">
                <span class="vm-detail-label">Política de Reinicio</span>
                <span class="vm-detail-val">${this.escapeHTML(info.restartPolicy || 'no')}</span>
            </div>
            <div class="vm-detail-item">
                <span class="vm-detail-label">Puertos Mapeados</span>
                <span class="vm-detail-val">${this.escapeHTML(portsStr)}</span>
            </div>
            <div class="vm-detail-item">
                <span class="vm-detail-label">Montajes de Volúmenes</span>
                <span class="vm-detail-val" style="font-size:11px;">${mountsStr}</span>
            </div>
            <div class="vm-detail-item" style="grid-column: 1 / -1;">
                <span class="vm-detail-label">Variables de Entorno</span>
                <span class="vm-detail-val" style="font-size:11px; max-height:120px; overflow-y:auto; display:block;">${envStr}</span>
            </div>
        `;
    }

    // =========================================================================
    // Gestión de Imágenes Docker (Listado, Creación, Compilación, Eliminación)
    // =========================================================================
    async loadImages(vmFilter) {
        if (!this.imagesListEl) return;
        this.imagesListEl.innerHTML = `
            <div class="empty-vm-list">
                <span class="btn-spinner"></span>
                <span>Cargando imágenes de Docker...</span>
            </div>
        `;

        const filter = (vmFilter || '').trim();
        const url = filter ? `/api/images?vm=${encodeURIComponent(filter)}` : '/api/images';

        try {
            const res = await fetch(url);
            if (!res.ok) {
                const text = await res.text();
                throw new Error(text || `Error ${res.status}`);
            }
            const images = await res.json();
            this.images = Array.isArray(images) ? images : [];
            this.renderImages(this.images);
        } catch (e) {
            console.error('Error cargando imágenes:', e);
            this.imagesListEl.innerHTML = `
                <div class="empty-vm-list" style="color:var(--accent-red)">
                    <p>No se pudieron cargar las imágenes de Docker.</p>
                    <span class="empty-subtext">${this.escapeHTML(e.message)}</span>
                </div>
            `;
        }
    }

    renderImages(images) {
        if (!this.imagesListEl) return;
        if (!Array.isArray(images) || images.length === 0) {
            this.imagesListEl.innerHTML = `
                <div class="empty-vm-list">
                    <p>No hay imágenes Docker disponibles.</p>
                    <span class="empty-subtext">Crea una imagen con "+ Crear imagen" o descarga una plantilla de contenedor.</span>
                </div>
            `;
            return;
        }

        this.imagesListEl.innerHTML = images
            .map((img) => {
                const shortId = img.id ? (img.id.startsWith('sha256:') ? img.id.substring(7, 19) : img.id.substring(0, 12)) : 'n/a';
                const tag = img.tag || '<none>';
                const repo = img.repository || '<none>';
                const size = img.size || 'n/a';
                const created = this.formatDate(img.createdAt);
                const vmName = img.vmName || '';

                return `
                    <div class="image-card" data-id="${this.escapeAttr(img.id)}">
                        <div class="image-head">
                            <div class="image-title-group">
                                <div class="image-icon">
                                    <svg viewBox="0 0 24 24" width="16" height="16" stroke="currentColor" stroke-width="2" fill="none">
                                        <polygon points="12 2 2 7 12 12 22 7 12 2"></polygon>
                                        <polyline points="2 17 12 22 22 17"></polyline>
                                        <polyline points="2 12 12 17 22 12"></polyline>
                                    </svg>
                                </div>
                                <div class="image-name">
                                    <span>${this.escapeHTML(repo)}</span>
                                    <span class="image-tag-badge">${this.escapeHTML(tag)}</span>
                                    ${vmName ? `<span class="vm-source-tag">${this.escapeHTML(vmName)}</span>` : ''}
                                </div>
                            </div>
                            <div class="image-size-badge">${this.escapeHTML(size)}</div>
                        </div>

                        <div class="image-info-grid">
                            <div class="image-info-item">
                                <span class="image-info-label">ID Imagen</span>
                                <span class="image-info-value">${this.escapeHTML(shortId)}</span>
                            </div>
                            <div class="image-info-item">
                                <span class="image-info-label">Creado</span>
                                <span class="image-info-value">${this.escapeHTML(created)}</span>
                            </div>
                        </div>

                        <div class="image-actions">
                            <button type="button" class="btn btn-secondary btn-sm" onclick="window.terminalApp.runImage('${this.escapeAttr(vmName)}', '${this.escapeAttr(repo)}', '${this.escapeAttr(tag)}')" title="Configurar y ejecutar contenedor con esta imagen">
                                <svg viewBox="0 0 24 24" width="12" height="12" stroke="currentColor" stroke-width="2.5" fill="none">
                                    <polygon points="5 3 19 12 5 21 5 3"></polygon>
                                </svg>
                                <span>Ejecutar</span>
                            </button>
                            <button type="button" class="btn btn-danger btn-sm" onclick="window.terminalApp.openDeleteImageModal('${this.escapeAttr(vmName)}', '${this.escapeAttr(img.id)}', '${this.escapeAttr(repo)}', '${this.escapeAttr(tag)}')" title="Eliminar imagen">
                                <svg viewBox="0 0 24 24" width="12" height="12" stroke="currentColor" stroke-width="2" fill="none">
                                    <polyline points="3 6 5 6 21 6"></polyline>
                                    <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"></path>
                                </svg>
                                <span>Eliminar</span>
                            </button>
                        </div>
                    </div>
                `;
            })
            .join('');
    }

    async runImage(vmName, repo, tag) {
        const fullImage = (tag && tag !== '<none>') ? `${repo}:${tag}` : repo;
        await this.openContainerWizard(vmName);
        this.selectWizardSource('images');
        this.selectedWizardImage = { repo, tag, ref: fullImage };
        if (this.wizImageInput) {
            this.wizImageInput.value = fullImage;
        }
        if (this.wizNameInput) {
            const cleanRepo = repo.replace(/[^a-zA-Z0-9_-]/g, '-').toLowerCase();
            const randSuffix = Math.random().toString(36).substring(2, 6);
            this.wizNameInput.value = `${cleanRepo}-${randSuffix}`;
        }
        this.goToWizardStep(3);
    }

    openBuildImageModal(preselectedVm = '') {
        if (!this.imageBuildModal) return;
        this.hideBuildImageFeedback();
        if (this.buildLogContainer) this.buildLogContainer.classList.add('hidden');
        if (this.buildLogOutput) this.buildLogOutput.textContent = '';

        this.updateImagesVmFilter();
        if (preselectedVm && this.buildVmSelect) {
            this.buildVmSelect.value = preselectedVm;
        } else if (this.imagesVmFilter && this.imagesVmFilter.value && this.buildVmSelect) {
            this.buildVmSelect.value = this.imagesVmFilter.value;
        }

        if (this.buildImageName) this.buildImageName.value = '';
        if (this.buildImageTag) this.buildImageTag.value = '';
        if (this.dfFileName) this.dfFileName.textContent = '';
        if (this.dfFileSize) this.dfFileSize.textContent = '';
        if (this.dfSelectedCard) this.dfSelectedCard.classList.add('hidden');
        if (this.dfFileDropzone) this.dfFileDropzone.classList.remove('hidden');
        if (this.dfFileInput) this.dfFileInput.value = '';
        if (this.dfExtraFilesInput) this.dfExtraFilesInput.value = '';
        this.attachedBuildFiles = [];
        this.renderAttachedFiles();
        this.switchDfTab('paste');

        this.imageBuildModal.classList.remove('hidden');
    }

    closeBuildImageModal() {
        if (this.imageBuildModal) {
            this.imageBuildModal.classList.add('hidden');
        }
        this.hideBuildImageFeedback();
    }

    switchDfTab(tab) {
        this.activeDfTab = tab;
        if (this.dfTabs) {
            this.dfTabs.forEach((t) => t.classList.toggle('active', t.dataset.dftab === tab));
        }
        if (this.dfUploadPanel) {
            this.dfUploadPanel.classList.toggle('hidden', tab !== 'upload');
        }
        if (this.dfPastePanel) {
            this.dfPastePanel.classList.toggle('hidden', tab !== 'paste');
        }
    }

    handlePrimaryFilesUpload(filesList) {
        if (!filesList || filesList.length === 0) return;
        const files = Array.from(filesList);

        // Buscar si entre los archivos seleccionados está el Dockerfile
        let dockerfileIdx = files.findIndex(f => {
            const name = f.name.toLowerCase();
            return name === 'dockerfile' || name.endsWith('.dockerfile');
        });

        // Si ninguno se llama Dockerfile y solo hay 1 archivo, ese es el Dockerfile
        if (dockerfileIdx === -1) {
            dockerfileIdx = 0;
        }

        const dockerfileObj = files[dockerfileIdx];
        this.handleDockerfileUpload(dockerfileObj);

        // El resto de los archivos se agregan al contexto de compilación
        const extraFiles = files.filter((_, idx) => idx !== dockerfileIdx);
        if (extraFiles.length > 0) {
            this.addAttachedFiles(extraFiles);
        }
    }

    handleDockerfileUpload(file) {
        if (!file) return;
        if (this.dfFileName) {
            this.dfFileName.textContent = file.name;
        }
        if (this.dfFileSize) {
            this.dfFileSize.textContent = this.formatBytes(file.size);
        }
        if (this.dfSelectedCard) {
            this.dfSelectedCard.classList.remove('hidden');
        }
        if (this.dfFileDropzone) {
            this.dfFileDropzone.classList.add('hidden');
        }
        const reader = new FileReader();
        reader.onload = (e) => {
            if (this.buildDockerfileContent) {
                this.buildDockerfileContent.value = e.target.result || '';
            }
        };
        reader.readAsText(file);
    }

    readFileAsBase64(file) {
        return new Promise((resolve, reject) => {
            const reader = new FileReader();
            reader.onload = () => {
                const dataUrl = reader.result;
                const base64Index = dataUrl.indexOf(';base64,');
                if (base64Index !== -1) {
                    resolve(dataUrl.substring(base64Index + 8));
                } else {
                    const commaIndex = dataUrl.indexOf(',');
                    resolve(commaIndex !== -1 ? dataUrl.substring(commaIndex + 1) : '');
                }
            };
            reader.onerror = (err) => reject(err);
            reader.readAsDataURL(file);
        });
    }

    async addAttachedFiles(fileList) {
        if (!fileList || fileList.length === 0) return;
        const files = Array.from(fileList);
        for (const file of files) {
            const cleanName = file.name.trim();
            if (!cleanName || cleanName.toLowerCase() === 'dockerfile') continue;
            try {
                const b64 = await this.readFileAsBase64(file);
                const existingIdx = this.attachedBuildFiles.findIndex(f => f.name === cleanName);
                const item = {
                    name: cleanName,
                    size: file.size,
                    content: b64
                };
                if (existingIdx !== -1) {
                    this.attachedBuildFiles[existingIdx] = item;
                } else {
                    this.attachedBuildFiles.push(item);
                }
            } catch (err) {
                console.error('Error leyendo archivo para contexto:', file.name, err);
            }
        }
        this.renderAttachedFiles();
    }

    removeAttachedFile(idx) {
        if (idx >= 0 && idx < this.attachedBuildFiles.length) {
            this.attachedBuildFiles.splice(idx, 1);
            this.renderAttachedFiles();
        }
    }

    renderAttachedFiles() {
        if (!this.dfAttachmentsList || !this.dfAttachmentsContainer || !this.dfAttachedCount) return;
        const count = this.attachedBuildFiles.length;
        this.dfAttachedCount.textContent = count;

        if (count === 0) {
            if (this.dfContextDropzone) this.dfContextDropzone.classList.remove('hidden');
            this.dfAttachmentsContainer.classList.add('hidden');
            this.dfAttachmentsList.innerHTML = '';
            return;
        }

        if (this.dfContextDropzone) this.dfContextDropzone.classList.add('hidden');
        this.dfAttachmentsContainer.classList.remove('hidden');
        this.dfAttachmentsList.innerHTML = this.attachedBuildFiles.map((file, idx) => `
            <div class="df-attached-item">
                <div class="df-attached-info">
                    <svg viewBox="0 0 24 24" width="13" height="13" stroke="currentColor" stroke-width="1.8" fill="none">
                        <path d="M13 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V9z"></path>
                        <polyline points="13 2 13 9 20 9"></polyline>
                    </svg>
                    <span class="df-attached-name" title="${this.escapeHtml(file.name)}">${this.escapeHtml(file.name)}</span>
                    <span class="df-attached-size">(${this.formatBytes(file.size)})</span>
                </div>
                <button type="button" class="df-attached-remove-btn" title="Quitar archivo" data-index="${idx}">&times;</button>
            </div>
        `).join('');

        this.dfAttachmentsList.querySelectorAll('.df-attached-remove-btn').forEach(btn => {
            btn.addEventListener('click', (e) => {
                e.stopPropagation();
                const idx = parseInt(btn.dataset.index, 10);
                this.removeAttachedFile(idx);
            });
        });
    }

    formatBytes(bytes) {
        if (!bytes || bytes === 0) return '0 B';
        if (bytes < 1024) return `${bytes} B`;
        if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
        return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
    }

    escapeHtml(str) {
        if (!str) return '';
        return String(str)
            .replace(/&/g, '&amp;')
            .replace(/</g, '&lt;')
            .replace(/>/g, '&gt;')
            .replace(/"/g, '&quot;')
            .replace(/'/g, '&#039;');
    }

    setButtonLoading(btn, isLoading) {
        if (!btn) return;
        btn.disabled = isLoading;
        if (isLoading) {
            btn.dataset.originalText = btn.innerHTML;
            btn.innerHTML = `<svg class="animate-spin -ml-1 mr-2 h-4 w-4 text-white inline-block" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24"><circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle><path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path></svg> Cargando...`;
        } else {
            if (btn.dataset.originalText) {
                btn.innerHTML = btn.dataset.originalText;
            }
        }
    }

    async submitBuildImage(andRun = false) {
        const vmName = this.buildVmSelect ? this.buildVmSelect.value : '';
        const imageName = this.buildImageName ? this.buildImageName.value.trim() : '';
        const tag = this.buildImageTag ? this.buildImageTag.value.trim() : '';
        const dockerfile = this.buildDockerfileContent ? this.buildDockerfileContent.value.trim() : '';

        if (!vmName) {
            this.showBuildImageFeedback('Debes seleccionar una máquina virtual.', 'error');
            return;
        }
        if (!imageName) {
            this.showBuildImageFeedback('Debes ingresar el nombre de la imagen.', 'error');
            return;
        }
        if (!dockerfile) {
            this.showBuildImageFeedback('El contenido del Dockerfile no puede estar vacío.', 'error');
            return;
        }

        const activeBtn = andRun ? this.submitImageBuildRunBtn : this.submitImageBuildBtn;
        this.setButtonLoading(activeBtn, true);
        this.hideBuildImageFeedback();
        if (this.buildLogContainer) this.buildLogContainer.classList.add('hidden');

        try {
            const res = await fetch('/api/images/build', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    vmName,
                    imageName,
                    tag: tag || 'latest',
                    dockerfile,
                    files: this.attachedBuildFiles.map(f => ({
                        name: f.name,
                        content: f.content
                    }))
                })
            });

            const data = await res.json();
            if (!res.ok) {
                if (data.output && this.buildLogOutput && this.buildLogContainer) {
                    this.buildLogOutput.textContent = data.output;
                    this.buildLogContainer.classList.remove('hidden');
                }
                const isMissingFileError = /falta un archivo|not found|no such file|file not found/i.test(data.error || '') ||
                                          /failed to compute cache key.*not found/i.test(data.output || '') ||
                                          /copy failed/i.test(data.output || '');
                if (isMissingFileError) {
                    this.showBuildImageFeedback(
                        'No se pudo crear la imagen: falta un archivo en el contexto de compilación requerido por COPY o ADD en el Dockerfile. Adjunta los archivos necesarios en la sección de archivos adjuntos antes de compilar.',
                        'error'
                    );
                } else {
                    this.showBuildImageFeedback(data.error || 'Error construyendo la imagen', 'error');
                }
                return;
            }

            this.closeBuildImageModal();
            this.loadImages(this.imagesVmFilter ? this.imagesVmFilter.value : '');

            if (andRun) {
                this.runImage(vmName, imageName, tag || 'latest');
            } else {
                if (this.wizardModal && !this.wizardModal.classList.contains('hidden') && this.wizardSource === 'images') {
                    this.loadWizardImages();
                }
            }
        } catch (e) {
            this.showBuildImageFeedback(e.message, 'error');
        } finally {
            this.setButtonLoading(activeBtn, false);
        }
    }

    openDeleteImageModal(vmName, id, repo, tag) {
        if (!this.deleteImageModal) return;
        this.pendingDeleteImage = { vmName, id, repo, tag };
        if (this.deleteImageNameEl) {
            this.deleteImageNameEl.textContent = (repo && repo !== '<none>') ? `${repo}:${tag || 'latest'}` : id;
        }
        if (this.deleteImageVmTag) {
            this.deleteImageVmTag.textContent = vmName ? `VM: ${vmName}` : '';
        }
        if (this.deleteImageForceCheckbox) {
            this.deleteImageForceCheckbox.checked = false;
        }
        this.hideDeleteImageFeedback();
        this.deleteImageModal.classList.remove('hidden');
    }

    closeDeleteImageModal() {
        if (this.deleteImageModal) {
            this.deleteImageModal.classList.add('hidden');
        }
        this.pendingDeleteImage = null;
    }

    async confirmDeleteImage() {
        if (!this.pendingDeleteImage) return;
        const { vmName, id, repo, tag } = this.pendingDeleteImage;
        const force = this.deleteImageForceCheckbox ? this.deleteImageForceCheckbox.checked : false;

        this.setButtonLoading(this.confirmDeleteImageBtn, true);
        this.hideDeleteImageFeedback();

        try {
            const ref = (repo && repo !== '<none>') ? `${repo}:${tag || 'latest'}` : id;
            const res = await fetch(`/api/images?vm=${encodeURIComponent(vmName)}&image=${encodeURIComponent(ref)}&force=${force}`, {
                method: 'DELETE'
            });

            if (!res.ok) {
                const text = await res.text();
                throw new Error(text || 'Error eliminando imagen');
            }

            this.closeDeleteImageModal();
            this.loadImages(this.imagesVmFilter ? this.imagesVmFilter.value : '');
        } catch (e) {
            this.showDeleteImageFeedback(e.message, 'error');
        } finally {
            this.setButtonLoading(this.confirmDeleteImageBtn, false);
        }
    }

    showBuildImageFeedback(msg, type = 'error') {
        if (!this.buildImageFeedback) return;
        this.buildImageFeedback.textContent = msg;
        this.buildImageFeedback.className = `modal-feedback ${type}`;
        this.buildImageFeedback.classList.remove('hidden');
    }

    hideBuildImageFeedback() {
        if (this.buildImageFeedback) {
            this.buildImageFeedback.classList.add('hidden');
            this.buildImageFeedback.textContent = '';
        }
    }

    showDeleteImageFeedback(msg, type = 'error') {
        if (!this.deleteImageFeedback) return;
        this.deleteImageFeedback.textContent = msg;
        this.deleteImageFeedback.className = `modal-feedback ${type}`;
        this.deleteImageFeedback.classList.remove('hidden');
    }

    hideDeleteImageFeedback() {
        if (this.deleteImageFeedback) {
            this.deleteImageFeedback.classList.add('hidden');
            this.deleteImageFeedback.textContent = '';
        }
    }

    // =========================================================================
    // Asistente Guiado de Creación de Contenedor (Wizard)
    // =========================================================================
    async openContainerWizard(preselectedVm) {
        if (!this.wizardModal) return;
        this.wizardModal.classList.remove('hidden');
        await this.updateContainersVmFilter();
        await this.updateImagesVmFilter();
        if (preselectedVm && this.wizVmSelect) {
            this.wizVmSelect.value = preselectedVm;
        }
        this.hideWizFeedback();
        this.selectedCategory = null;
        this.selectedTemplate = null;
        this.selectedWizardImage = null;

        if (this.wizNameInput) this.wizNameInput.value = '';
        if (this.wizImageInput) this.wizImageInput.value = '';
        if (this.wizPortsContainer) this.wizPortsContainer.innerHTML = '';
        if (this.wizEnvContainer) this.wizEnvContainer.innerHTML = '';
        if (this.wizVolumesContainer) this.wizVolumesContainer.innerHTML = '';

        this.selectWizardSource(this.wizardSource || 'images');
        this.goToWizardStep(1);
        this.wizardModal.classList.remove('hidden');
    }

    closeContainerWizard() {
        if (this.wizardModal) {
            this.wizardModal.classList.add('hidden');
        }
        this.hideWizFeedback();
    }

    selectWizardSource(source) {
        this.wizardSource = source;
        if (this.wizSourceImages) this.wizSourceImages.classList.toggle('active', source === 'images');
        if (this.wizSourceCatalog) this.wizSourceCatalog.classList.toggle('active', source === 'catalog');
        if (this.wizImagesPanel) this.wizImagesPanel.classList.toggle('hidden', source !== 'images');
        if (this.wizCatalogPanel) this.wizCatalogPanel.classList.toggle('hidden', source !== 'catalog');
        this.hideWizFeedback();

        if (source === 'images') {
            this.loadWizardImages();
        } else {
            this.renderWizardCategories();
        }
    }

    async loadWizardImages() {
        if (!this.wizImagesGrid) return;
        const vm = this.wizVmSelect ? this.wizVmSelect.value : '';
        if (!vm) {
            this.wizImagesGrid.innerHTML = `
                <div class="empty-vm-list">
                    <span>Selecciona una máquina virtual para ver sus imágenes disponibles.</span>
                </div>
            `;
            return;
        }

        this.wizImagesGrid.innerHTML = `
            <div class="empty-vm-list">
                <span class="btn-spinner"></span>
                <span>Cargando imágenes de ${this.escapeHTML(vm)}...</span>
            </div>
        `;

        try {
            const res = await fetch(`/api/images?vm=${encodeURIComponent(vm)}`);
            if (!res.ok) throw new Error('Error al listar imágenes');
            const images = await res.json();
            if (!Array.isArray(images) || images.length === 0) {
                this.wizImagesGrid.innerHTML = `
                    <div class="empty-vm-list">
                        <p>No se encontraron imágenes en esta VM.</p>
                        <span class="empty-subtext">Crea una con "+ Crear nueva imagen" o selecciona "Aplicaciones" para usar plantillas.</span>
                    </div>
                `;
                return;
            }

            this.wizImagesGrid.innerHTML = images
                .map((img) => {
                    const tag = img.tag || 'latest';
                    const fullRef = (img.repository && img.repository !== '<none>') ? `${img.repository}:${tag}` : img.id;
                    const isSelected = this.selectedWizardImage && (this.selectedWizardImage.id === img.id || this.selectedWizardImage.ref === fullRef);
                    return `
                        <div class="wiz-image-card ${isSelected ? 'active' : ''}" onclick="window.terminalApp.selectWizardImage('${this.escapeAttr(img.id)}', '${this.escapeAttr(img.repository)}', '${this.escapeAttr(tag)}')">
                            <div class="wiz-image-card-head">
                                <span class="wiz-image-card-title">${this.escapeHTML(img.repository || '<none>')}</span>
                                <span class="wiz-image-card-tag">${this.escapeHTML(tag)}</span>
                            </div>
                            <div class="wiz-image-card-meta">
                                <span>${this.escapeHTML(img.size || '')}</span>
                                <span>${this.escapeHTML(this.formatDate(img.createdAt))}</span>
                            </div>
                        </div>
                    `;
                })
                .join('');
        } catch (e) {
            this.wizImagesGrid.innerHTML = `
                <div class="empty-vm-list error">
                    <span>${this.escapeHTML(e.message)}</span>
                </div>
            `;
        }
    }

    selectWizardImage(id, repo, tag) {
        const fullRef = (repo && repo !== '<none>') ? `${repo}:${tag || 'latest'}` : id;
        this.selectedWizardImage = { id, repo, tag, ref: fullRef };

        if (this.wizImagesGrid) {
            const cards = this.wizImagesGrid.querySelectorAll('.wiz-image-card');
            cards.forEach((c) => c.classList.remove('active'));
            if (window.event && window.event.currentTarget) {
                window.event.currentTarget.classList.add('active');
            }
        }

        if (this.wizImageInput) {
            this.wizImageInput.value = fullRef;
        }
        if (this.wizNameInput && !this.wizNameInput.value) {
            const cleanRepo = (repo || 'app').replace(/[^a-zA-Z0-9_-]/g, '-').toLowerCase();
            const randSuffix = Math.random().toString(36).substring(2, 6);
            this.wizNameInput.value = `${cleanRepo}-${randSuffix}`;
        }
        this.hideWizFeedback();
    }

    renderWizardCategories() {
        if (!this.wizCategoriesGrid) return;
        if (!this.dockerCatalog || this.dockerCatalog.length === 0) {
            this.wizCategoriesGrid.innerHTML = `
                <div class="empty-vm-list">
                    <span class="btn-spinner"></span> Cargando catálogo...
                </div>
            `;
            this.loadDockerCatalog().then(() => this.renderWizardCategories());
            return;
        }

        this.wizCategoriesGrid.innerHTML = this.dockerCatalog
            .map(
                (cat) => `
            <div class="wiz-card ${this.selectedCategory === cat.id ? 'selected' : ''}" data-cat="${this.escapeAttr(cat.id)}" onclick="window.terminalApp.selectWizardCategory('${this.escapeAttr(cat.id)}')">
                <span class="wiz-card-title">${this.escapeHTML(cat.name)}</span>
                <span class="wiz-card-desc">${this.escapeHTML(cat.description)}</span>
            </div>
        `
            )
            .join('');
    }

    selectWizardCategory(catId) {
        this.selectedCategory = catId;
        this.renderWizardCategories();
        this.hideWizFeedback();
    }

    renderWizardApps(catId) {
        if (!this.wizAppsGrid) return;
        const cat = (this.dockerCatalog || []).find((c) => c.id === catId);
        if (!cat) {
            this.wizAppsGrid.innerHTML = `<div class="empty-vm-list">Categoría no encontrada.</div>`;
            return;
        }

        if (this.wizCategorySelectedTitle) {
            this.wizCategorySelectedTitle.textContent = `— ${cat.name}`;
        }

        this.wizAppsGrid.innerHTML = (cat.items || [])
            .map(
                (app) => `
            <div class="wiz-card ${this.selectedTemplate && this.selectedTemplate.id === app.id ? 'selected' : ''}" data-app="${this.escapeAttr(app.id)}" onclick="window.terminalApp.selectWizardTemplate('${this.escapeAttr(app.id)}')">
                <span class="wiz-card-title">${this.escapeHTML(app.name)}</span>
                <span class="wiz-card-desc">${this.escapeHTML(app.description)}</span>
                ${app.defaultImage ? `<span class="wiz-card-image">${this.escapeHTML(app.defaultImage)}</span>` : '<span class="wiz-card-image">Imagen libre</span>'}
            </div>
        `
            )
            .join('');
    }

    selectWizardTemplate(appId) {
        const cat = (this.dockerCatalog || []).find((c) => c.id === this.selectedCategory);
        if (!cat) return;
        const tmpl = (cat.items || []).find((t) => t.id === appId);
        if (!tmpl) return;

        this.selectedTemplate = tmpl;
        this.renderWizardApps(this.selectedCategory);
        this.hideWizFeedback();
    }

    applyWizardTemplate(tmpl) {
        if (!tmpl) return;
        if (this.wizNameInput) {
            const randomSuffix = Math.random().toString(36).substring(2, 6);
            this.wizNameInput.value = tmpl.id === 'custom' ? `app-${randomSuffix}` : `${tmpl.id}-${randomSuffix}`;
        }
        if (this.wizImageInput) {
            this.wizImageInput.value = tmpl.defaultImage || '';
        }
        if (this.wizRestartSelect) {
            this.wizRestartSelect.value = tmpl.defaultRestart || 'unless-stopped';
        }

        // Poblar puertos
        if (this.wizPortsContainer) {
            this.wizPortsContainer.innerHTML = '';
            (tmpl.recommendedPorts || []).forEach((p) => {
                this.addWizPortRow(p.hostPort, p.containerPort, p.protocol);
            });
        }

        // Poblar variables de entorno
        if (this.wizEnvContainer) {
            this.wizEnvContainer.innerHTML = '';
            (tmpl.recommendedEnv || []).forEach((e) => {
                this.addWizEnvRow(e.key, e.value);
            });
        }

        // Poblar volúmenes
        if (this.wizVolumesContainer) {
            this.wizVolumesContainer.innerHTML = '';
            (tmpl.recommendedVolumes || []).forEach((v) => {
                this.addWizVolumeRow(v.hostPath, v.containerPath);
            });
        }
    }

    addWizPortRow(hostPort = '', containerPort = '', proto = 'tcp') {
        if (!this.wizPortsContainer) return;
        const row = document.createElement('div');
        row.className = 'dynamic-row wiz-port-row';
        row.innerHTML = `
            <input type="number" class="port-host" placeholder="Host" value="${this.escapeAttr(hostPort)}" min="1" max="65535" style="flex:1;" />
            <span style="color:var(--text-muted);">&rarr;</span>
            <input type="number" class="port-container" placeholder="Contenedor" value="${this.escapeAttr(containerPort)}" min="1" max="65535" style="flex:1;" />
            <select class="port-proto" style="width:75px;">
                <option value="tcp" ${proto === 'tcp' ? 'selected' : ''}>TCP</option>
                <option value="udp" ${proto === 'udp' ? 'selected' : ''}>UDP</option>
            </select>
            <button type="button" class="btn-remove-row" title="Eliminar fila">&times;</button>
        `;
        row.querySelector('.btn-remove-row').addEventListener('click', () => row.remove());
        this.wizPortsContainer.appendChild(row);
    }

    addWizEnvRow(key = '', value = '') {
        if (!this.wizEnvContainer) return;
        const row = document.createElement('div');
        row.className = 'dynamic-row wiz-env-row';
        row.innerHTML = `
            <input type="text" class="env-key" placeholder="CLAVE" value="${this.escapeAttr(key)}" style="flex:1;" />
            <span style="color:var(--text-muted);">=</span>
            <input type="text" class="env-val" placeholder="Valor" value="${this.escapeAttr(value)}" style="flex:1.5;" />
            <button type="button" class="btn-remove-row" title="Eliminar fila">&times;</button>
        `;
        row.querySelector('.btn-remove-row').addEventListener('click', () => row.remove());
        this.wizEnvContainer.appendChild(row);
    }

    async addWizVolumeRow(hostPath = '', containerPath = '') {
        if (!this.wizVolumesContainer) return;
        const row = document.createElement('div');
        row.className = 'dynamic-row wiz-vol-row';
        
        const selectedVm = this.wizVmSelect ? this.wizVmSelect.value.trim() : '';

        row.innerHTML = `
            <div class="vol-source-wrap">
                <!-- Select de volumen existente -->
                <select class="vol-select">
                    <option value="" disabled ${!hostPath ? 'selected' : ''}>Cargando volúmenes...</option>
                </select>
                <!-- Botón rápido para crear nuevo volumen -->
                <button type="button" class="btn-inline-add" title="Crear nuevo volumen en esta VM">+</button>
                <!-- Input oculto para mantener el valor seleccionado para el envío -->
                <input type="hidden" class="vol-host" value="${this.escapeAttr(hostPath)}" />
            </div>
            <span style="color:var(--text-muted); padding: 0 4px; flex-shrink: 0;">&rarr;</span>
            <input type="text" class="vol-container" placeholder="Ruta en contenedor (ej. /var/lib/data)" value="${this.escapeAttr(containerPath)}" style="flex: 1;" />
            <button type="button" class="btn-remove-row" title="Eliminar fila">&times;</button>
        `;

        const volSelect = row.querySelector('.vol-select');
        const btnNewVol = row.querySelector('.btn-inline-add');
        const hiddenHost = row.querySelector('.vol-host');

        const populateOptions = (volumesList, selectVal) => {
            let opts = '<option value="" disabled ' + (!selectVal ? 'selected' : '') + '>Selecciona un volumen...</option>';
            if (volumesList.length === 0) {
                opts = '<option value="" disabled selected>No hay volúmenes (crea uno con +)</option>';
            } else {
                volumesList.forEach((v) => {
                    const vName = v.name || v.Name || v;
                    opts += `<option value="${this.escapeAttr(vName)}" ${selectVal === vName ? 'selected' : ''}>${this.escapeHTML(vName)}</option>`;
                });
            }
            volSelect.innerHTML = opts;
            if (selectVal && volumesList.some(v => (v.name || v.Name || v) === selectVal)) {
                volSelect.value = selectVal;
                hiddenHost.value = selectVal;
            } else if (!selectVal && volumesList.length > 0) {
                hiddenHost.value = '';
            }
        };

        // Cargar volúmenes de la VM seleccionada
        if (selectedVm) {
            this.fetchVmVolumes(selectedVm).then((vols) => {
                populateOptions(vols, hostPath);
            });
        } else {
            volSelect.innerHTML = '<option value="" disabled selected>Selecciona una VM primero</option>';
        }

        volSelect.addEventListener('change', () => {
            hiddenHost.value = volSelect.value;
        });

        btnNewVol.addEventListener('click', () => {
            const currentVm = this.wizVmSelect ? this.wizVmSelect.value.trim() : '';
            if (!currentVm) {
                this.showWizFeedback('Debes seleccionar una máquina virtual primero.', 'error');
                return;
            }
            if (this.createVolumeForm) this.createVolumeForm.reset();
            if (this.createVolumeFeedback) this.createVolumeFeedback.classList.add('hidden');
            if (this.createVolumeVm) {
                this.createVolumeVm.innerHTML = `<option value="${this.escapeAttr(currentVm)}" selected>${this.escapeHTML(currentVm)}</option>`;
                this.createVolumeVm.value = currentVm;
            }

            // Callback para seleccionar automáticamente el nuevo volumen en esta fila
            this._onVolumeCreatedCallback = (newVolName, vm) => {
                if (vm === currentVm) {
                    this.fetchVmVolumes(currentVm).then((vols) => {
                        populateOptions(vols, newVolName);
                    });
                }
            };

            if (this.createVolumeModal) {
                this.createVolumeModal.classList.remove('hidden');
                if (this.createVolumeName) {
                    setTimeout(() => this.createVolumeName.focus(), 50);
                }
            }
        });

        row.querySelector('.btn-remove-row').addEventListener('click', () => row.remove());
        this.wizVolumesContainer.appendChild(row);
    }

    goToWizardStep(step) {
        if (step < 1 || step > 4) return;

        // Validaciones al avanzar
        if (step > this.currentWizardStep) {
            if (this.currentWizardStep === 1) {
                if (!this.wizVmSelect || !this.wizVmSelect.value) {
                    this.showWizFeedback('Selecciona una máquina virtual.', 'error');
                    return;
                }
                if (this.wizardSource === 'images') {
                    const imgVal = this.wizImageInput ? this.wizImageInput.value.trim() : '';
                    if (!this.selectedWizardImage && !imgVal) {
                        this.showWizFeedback('Selecciona una imagen de la lista o crea una nueva.', 'error');
                        return;
                    }
                    step = 3;
                    const selectedVm = this.wizVmSelect ? this.wizVmSelect.value.trim() : '';
                    this.updateWizardNetworks(selectedVm);
                } else {
                    if (!this.selectedCategory) {
                        this.showWizFeedback('Selecciona una categoría.', 'error');
                        return;
                    }
                    this.renderWizardApps(this.selectedCategory);
                }
            } else if (this.currentWizardStep === 2) {
                if (!this.selectedTemplate) {
                    this.showWizFeedback('Selecciona una plantilla.', 'error');
                    return;
                }
                this.applyWizardTemplate(this.selectedTemplate);
                const selectedVm = this.wizVmSelect ? this.wizVmSelect.value.trim() : '';
                this.updateWizardNetworks(selectedVm);
            } else if (this.currentWizardStep === 3) {
                const name = this.wizNameInput ? this.wizNameInput.value.trim() : '';
                const image = this.wizImageInput ? this.wizImageInput.value.trim() : '';
                if (!name) {
                    this.showWizFeedback('El nombre es obligatorio.', 'error');
                    return;
                }
                if (!image) {
                    this.showWizFeedback('La imagen es obligatoria.', 'error');
                    return;
                }
                this.buildWizardSummary();
            }
        } else if (step < this.currentWizardStep) {
            if (this.currentWizardStep === 3 && this.wizardSource === 'images') {
                step = 1;
            }
        }

        this.hideWizFeedback();
        this.currentWizardStep = step;

        // Actualizar indicadores de pasos
        this.wizStepPills.forEach((p) => {
            const pStep = parseInt(p.dataset.step, 10);
            p.classList.toggle('active', pStep === step);
            p.classList.toggle('completed', pStep < step);
        });

        // Actualizar contenido visible
        this.wizStepContents.forEach((c, idx) => {
            c.classList.toggle('active', idx + 1 === step);
        });

        // Actualizar botones de navegación
        if (this.wizPrevBtn) this.wizPrevBtn.classList.toggle('hidden', step === 1);
        if (this.wizNextBtn) this.wizNextBtn.classList.toggle('hidden', step === 4);
        if (this.wizSubmitBtn) this.wizSubmitBtn.classList.toggle('hidden', step !== 4);
    }

    buildWizardSummary() {
        if (!this.wizSummaryCard) return;
        const vm = this.wizVmSelect ? this.wizVmSelect.value : '';
        const name = this.wizNameInput ? this.wizNameInput.value.trim() : '';
        const image = this.wizImageInput ? this.wizImageInput.value.trim() : '';
        const restart = this.wizRestartSelect ? this.wizRestartSelect.value : 'unless-stopped';

        // Recopilar puertos
        const portRows = Array.from(document.querySelectorAll('.wiz-port-row'));
        const ports = portRows
            .map((r) => {
                const hp = r.querySelector('.port-host').value.trim();
                const cp = r.querySelector('.port-container').value.trim();
                const proto = r.querySelector('.port-proto').value;
                if (hp && cp) return `${hp}:${cp}/${proto}`;
                return null;
            })
            .filter(Boolean);

        // Recopilar variables
        const envRows = Array.from(document.querySelectorAll('.wiz-env-row'));
        const envs = envRows
            .map((r) => {
                const k = r.querySelector('.env-key').value.trim();
                const v = r.querySelector('.env-val').value.trim();
                if (k) return `${k}=${v}`;
                return null;
            })
            .filter(Boolean);

        // Recopilar volúmenes
        const volRows = Array.from(document.querySelectorAll('.wiz-vol-row'));
        const vols = volRows
            .map((r) => {
                const h = r.querySelector('.vol-host').value.trim();
                const c = r.querySelector('.vol-container').value.trim();
                if (h && c) return `${h} &rarr; ${c}`;
                return null;
            })
            .filter(Boolean);

        this.wizSummaryCard.innerHTML = `
            <div class="summary-row">
                <span class="summary-label">Máquina:</span>
                <span class="summary-val highlight">${this.escapeHTML(vm)}</span>
            </div>
            <div class="summary-row">
                <span class="summary-label">Nombre:</span>
                <span class="summary-val">${this.escapeHTML(name)}</span>
            </div>
            <div class="summary-row">
                <span class="summary-label">Imagen:</span>
                <span class="summary-val">${this.escapeHTML(image)}</span>
            </div>
            <div class="summary-row">
                <span class="summary-label">Reinicio:</span>
                <span class="summary-val">${this.escapeHTML(restart)}</span>
            </div>
            <div class="summary-row">
                <span class="summary-label">Puertos:</span>
                <span class="summary-val">${ports.length > 0 ? this.escapeHTML(ports.join(', ')) : 'Ninguno'}</span>
            </div>
            <div class="summary-row">
                <span class="summary-label">Variables:</span>
                <span class="summary-val">${envs.length > 0 ? this.escapeHTML(envs.join(', ')) : 'Ninguna'}</span>
            </div>
            <div class="summary-row">
                <span class="summary-label">Volúmenes:</span>
                <span class="summary-val">${vols.length > 0 ? vols.join(', ') : 'Ninguno'}</span>
            </div>
            <div class="summary-row">
                <span class="summary-label">Red:</span>
                <span class="summary-val highlight">${this.escapeHTML(this.wizNetworkSelect && this.wizNetworkSelect.value ? this.wizNetworkSelect.value : 'Predeterminada (bridge)')}</span>
            </div>
        `;
    }

    async submitCreateContainer() {
        const vmName = this.wizVmSelect ? this.wizVmSelect.value.trim() : '';
        const name = this.wizNameInput ? this.wizNameInput.value.trim() : '';
        const image = this.wizImageInput ? this.wizImageInput.value.trim() : '';
        const restartPolicy = this.wizRestartSelect ? this.wizRestartSelect.value : 'unless-stopped';
        const network = this.wizNetworkSelect ? this.wizNetworkSelect.value.trim() : '';

        if (!vmName || !name || !image) {
            this.showWizFeedback('Faltan campos obligatorios para crear el contenedor.', 'error');
            return;
        }

        const ports = [];
        Array.from(document.querySelectorAll('.wiz-port-row')).forEach((r) => {
            const hp = parseInt(r.querySelector('.port-host').value, 10);
            const cp = parseInt(r.querySelector('.port-container').value, 10);
            const proto = r.querySelector('.port-proto').value || 'tcp';
            if (hp && cp) {
                ports.push({ hostPort: hp, containerPort: cp, protocol: proto });
            }
        });

        const env = [];
        Array.from(document.querySelectorAll('.wiz-env-row')).forEach((r) => {
            const k = r.querySelector('.env-key').value.trim();
            const v = r.querySelector('.env-val').value.trim();
            if (k) {
                env.push({ key: k, value: v });
            }
        });

        const volumes = [];
        Array.from(document.querySelectorAll('.wiz-vol-row')).forEach((r) => {
            const h = r.querySelector('.vol-host').value.trim();
            const c = r.querySelector('.vol-container').value.trim();
            if (h && c) {
                volumes.push({ hostPath: h, containerPath: c });
            }
        });

        const payload = {
            vmName,
            name,
            image,
            restartPolicy,
            network,
            ports,
            env,
            volumes,
        };

        if (this.wizSubmitBtn) {
            this.wizSubmitBtn.disabled = true;
            const spinner = this.wizSubmitBtn.querySelector('.btn-spinner');
            if (spinner) spinner.classList.remove('hidden');
        }

        try {
            const res = await fetch('/api/containers', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(payload),
            });

            if (res.ok) {
                this.closeContainerWizard();
                if (this.containersVmFilter) {
                    this.containersVmFilter.value = vmName;
                }
                this.loadContainers(vmName);
                this.appendInfoLine(`[Contenedor '${name}' desplegado exitosamente en VM '${vmName}']`);
            } else {
                const text = await res.text();
                this.showWizFeedback(`Error al crear contenedor: ${this.parseApiError(text)}`, 'error');
            }
        } catch (e) {
            this.showWizFeedback(`Error de conexión al crear contenedor: ${e.message}`, 'error');
        } finally {
            if (this.wizSubmitBtn) {
                this.wizSubmitBtn.disabled = false;
                const spinner = this.wizSubmitBtn.querySelector('.btn-spinner');
                if (spinner) spinner.classList.add('hidden');
            }
        }
    }

    showWizFeedback(msg, type = 'error') {
        if (!this.wizFeedback) return;
        this.wizFeedback.className = `modal-feedback ${type}`;
        this.wizFeedback.textContent = msg;
        this.wizFeedback.classList.remove('hidden');
    }

    hideWizFeedback() {
        if (!this.wizFeedback) return;
        this.wizFeedback.className = 'modal-feedback hidden';
        this.wizFeedback.textContent = '';
    }

    escapeAttr(str) {
        if (!str) return '';
        return String(str).replace(/["'<>]/g, '');
    }
}

document.addEventListener('DOMContentLoaded', () => {
    new TerminalApp();
});
