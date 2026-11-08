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
        this.vmCreateProgress = document.getElementById('vm-create-progress');
        this.vmProgressStageTitle = document.getElementById('vm-progress-stage-title');
        this.vmProgressBadge = document.getElementById('vm-progress-badge');
        this.vmProgressBarFill = document.getElementById('vm-progress-bar-fill');
        this.vmProgressMsg = document.getElementById('vm-progress-msg');
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
        this.wizNetworksContainer = document.getElementById('wiz-networks-container');
        this.wizAddNetworkRowBtn = document.getElementById('wiz-add-network-row-btn');
        this.wizAddNetworkBtn = document.getElementById('wiz-add-network-btn');
        this.wizNetAdminCheckbox = document.getElementById('wiz-net-admin');
        this.wizSummaryCard = document.getElementById('wiz-summary-card');
        this.wizFeedback = document.getElementById('wiz-feedback');
        this.currentWizardStep = 1;
        this.selectedCategory = null;
        this.selectedTemplate = null;

        // Subnavegación de Docker (Contenedores / Imágenes)
        this.dockerSubnav = document.getElementById('docker-subnav');
        this.dockerSubnavBtns = Array.from(document.querySelectorAll('.docker-subnav-btn'));
        this.activeDockerSubnav = 'containers';
        this.selectedDockerVm = '';

        // Elementos de Gestión de Imágenes Docker
        this.imagesListEl = document.getElementById('images-list');
        this.imagesVmFilter = document.getElementById('images-vm-filter');
        this.openCreateImageBtn = document.getElementById('open-create-image-btn');
        this.refreshImagesBtn = document.getElementById('refresh-images-btn');
        this.images = [];
        this.imagesTarListEl = document.getElementById('images-tar-list');
        this.imagesExportFeedbackEl = document.getElementById('images-export-feedback');
        this.imagesSubtabBtns = Array.from(document.querySelectorAll('.images-subtab-btn'));
        this.activeImagesSubtab = 'images';
        this.exportedTars = [];

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

        // Modales de Docker Hub (Credenciales y Publicación)
        this.openDockerHubCredsBtn = document.getElementById('open-dockerhub-creds-btn');
        this.dockerHubCredsModal = document.getElementById('dockerhub-creds-modal');
        this.closeDockerHubCredsModalBtn = document.getElementById('close-dockerhub-creds-modal');
        this.cancelDockerHubCredsBtn = document.getElementById('cancel-dockerhub-creds-btn');
        this.saveDockerHubCredsBtn = document.getElementById('save-dockerhub-creds-btn');
        this.dockerHubUsernameInput = document.getElementById('dockerhub-username-input');
        this.dockerHubTokenInput = document.getElementById('dockerhub-token-input');
        this.dockerHubCredsFeedback = document.getElementById('dockerhub-creds-feedback');

        this.dockerHubPublishModal = document.getElementById('dockerhub-publish-modal');
        this.closeDockerHubPublishModalBtn = document.getElementById('close-dockerhub-publish-modal');
        this.cancelDockerHubPublishBtn = document.getElementById('cancel-dockerhub-publish-btn');
        this.submitDockerHubPublishBtn = document.getElementById('submit-dockerhub-publish-btn');
        this.changeDockerHubCredsBtn = document.getElementById('change-dockerhub-creds-btn');
        this.publishVmDisplay = document.getElementById('publish-vm-display');
        this.publishSourceImageDisplay = document.getElementById('publish-source-image-display');
        this.publishUsernameDisplay = document.getElementById('publish-username-display');
        this.publishRepoInput = document.getElementById('publish-repo-input');
        this.publishTagInput = document.getElementById('publish-tag-input');
        this.publishTargetPreview = document.getElementById('publish-target-preview');
        this.dockerHubPublishFeedback = document.getElementById('dockerhub-publish-feedback');
        this.dockerHubPublishLogContainer = document.getElementById('dockerhub-publish-log-container');
        this.dockerHubPublishLogOutput = document.getElementById('dockerhub-publish-log-output');
        this.pendingPublishImage = null;
        this.dockerHubUsername = '';
        this.continueToPublishAfterCreds = false;

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
        this.toggleNetworkAdvancedBtn = document.getElementById('toggle-network-advanced-btn');
        this.networkAdvancedFields = document.getElementById('network-advanced-fields');
        this.networkAdvancedArrow = document.getElementById('network-advanced-arrow');
        this.createNetworkSubnet = document.getElementById('create-network-subnet');
        this.createNetworkGateway = document.getElementById('create-network-gateway');
        this.createNetworkIpRange = document.getElementById('create-network-iprange');
        this.createNetworkInternal = document.getElementById('create-network-internal');
        this.createNetworkAttachable = document.getElementById('create-network-attachable');
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

        // Elementos de Gestión de Docker Swarm Clusters
        this.clustersListPanel = document.getElementById('clusters-list-panel');
        this.clustersListEl = document.getElementById('clusters-list');
        this.openCreateClusterBtn = document.getElementById('open-create-cluster');
        this.refreshClustersBtn = document.getElementById('refresh-clusters-btn');
        this.clusterDetailPanel = document.getElementById('cluster-detail-panel');
        this.backToClustersBtn = document.getElementById('back-to-clusters-btn');
        this.refreshClusterDetailBtn = document.getElementById('refresh-cluster-detail-btn');
        this.clusterDetailNameEl = document.getElementById('cluster-detail-name');
        this.clusterDetailStatusPill = document.getElementById('cluster-detail-status-pill');
        this.clusterDetailSummaryEl = document.getElementById('cluster-detail-summary');
        this.clusterDetailWarningEl = document.getElementById('cluster-detail-warning');
        this.clusterManagerCardWrap = document.getElementById('cluster-manager-card-wrap');
        this.clusterWorkersCountLabel = document.getElementById('cluster-workers-count-label');
        this.clusterWorkersListEl = document.getElementById('cluster-workers-list');
        this.clusters = [];
        this.activeClusterId = null;
        this.activeClusterData = null;

        // Modal de Creación de Cluster Docker Swarm
        this.createClusterModal = document.getElementById('create-cluster-modal');
        this.closeCreateClusterModalBtn = document.getElementById('close-create-cluster-modal');
        this.cancelCreateClusterBtn = document.getElementById('cancel-create-cluster-btn');
        this.createClusterForm = document.getElementById('create-cluster-form');
        this.createClusterNameInput = document.getElementById('create-cluster-name');
        this.refreshClusterVmsBtn = document.getElementById('refresh-cluster-vms-btn');
        this.createClusterManagerList = document.getElementById('create-cluster-manager-list');
        this.createClusterWorkersList = document.getElementById('create-cluster-workers-list');
        this.createClusterWorkersCounter = document.getElementById('create-cluster-workers-counter');
        this.createClusterSingleNodeWarning = document.getElementById('create-cluster-single-node-warning');
        this.createClusterFeedback = document.getElementById('create-cluster-feedback');
        this.submitCreateClusterBtn = document.getElementById('submit-create-cluster-btn');
        this.clusterCreateProgressWrap = document.getElementById('cluster-create-progress');
        this.clusterProgressStageTitle = document.getElementById('cluster-progress-stage-title');
        this.clusterProgressStepBadge = document.getElementById('cluster-progress-step-badge');
        this.clusterProgressBarFill = document.getElementById('cluster-progress-bar-fill');
        this.clusterProgressMsg = document.getElementById('cluster-progress-msg');
        this.clusterProgressStepsList = document.getElementById('cluster-progress-steps-list');
        this.clusterEligibleVMs = [];
        this.selectedClusterManagerVM = '';
        this.selectedClusterWorkerVMs = new Set();
        this.isCreatingCluster = false;

        // Modal de Eliminación de Nodo del Cluster
        this.removeClusterNodeModal = document.getElementById('remove-cluster-node-modal');
        this.closeRemoveClusterNodeModalBtn = document.getElementById('close-remove-cluster-node-modal');
        this.cancelRemoveClusterNodeBtn = document.getElementById('cancel-remove-cluster-node');
        this.confirmRemoveClusterNodeBtn = document.getElementById('confirm-remove-cluster-node');
        this.removeClusterNodeVmNameEl = document.getElementById('remove-cluster-node-vm-name');
        this.removeClusterNodeRoleBadgeEl = document.getElementById('remove-cluster-node-role-badge');
        this.removeClusterNodeClusterNameEl = document.getElementById('remove-cluster-node-cluster-name');
        this.removeClusterNodeConsequenceEl = document.getElementById('remove-cluster-node-consequence');
        this.removeClusterNodeForceBox = document.getElementById('remove-cluster-node-force-box');
        this.removeClusterNodeForceCheck = document.getElementById('remove-cluster-node-force-check');
        this.removeClusterNodeForceText = document.getElementById('remove-cluster-node-force-text');
        this.removeClusterNodeFeedback = document.getElementById('remove-cluster-node-feedback');
        this.pendingRemoveClusterNode = null;

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
        this.ansiRemainder = '';
        this.lastKnownPromptText = '';
        this.currentPath = '~';
        this.isClearingScreen = false;
        this.clearScreenTimer = null;
        this.screenReplayIndex = null;
        this.ignoreEchoCommand = null;
        this.currentVM = null;
        this.currentContainer = null;
        this.containerSessionOpened = false;

        // Configuracion por defecto y activa
        this.defaultConfig = null;
        this.currentConfig = {};
        this.hosts = [];
        this.activeHostId = '';
        this.editingHostId = null;

        this.ws = new TerminalWebSocket('/ws', {
            onStatusChange: (status) => this.setStatus(status),
            onOpen: () => {
                if (this.currentContainer) {
                    this.containerSessionOpened = true;
                }
                this.handleResize();
            },
            onMessage: (data) => this.handleOutput(data),
            onClose: () => {
                if (this.currentContainer && this.containerSessionOpened) {
                    const closedContainer = this.currentContainer.container;
                    this.currentContainer = null;
                    this.containerSessionOpened = false;
                    this.appendInfoLine(`[Sesión en contenedor '${closedContainer}' finalizada. Reconectando al host principal...]`);
                    this.connectCurrent();
                }
            },
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
        this.currentContainer = null;
        this.containerSessionOpened = false;
        this.updateTargetPill(this.currentConfig);
        const promptChar = document.querySelector('#input-area .prompt-char');
        if (promptChar) {
            promptChar.textContent = '>';
        }
        this.currentPath = '';
        this.updateCurrentPath(this.currentConfig.sshStartPath || '~');
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
            this.hideVMCreateProgress();
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
                this.hideVMCreateProgress();
            });
        }

        this.cancelVmFormBtn.addEventListener('click', () => {
            this.vmFormWrap.classList.add('hidden');
            this.vmForm.reset();
            this.hideVMFeedback();
            this.hideVMCreateProgress();
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
                this.handleDockerVmChange(this.containersVmFilter.value);
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
                this.handleDockerVmChange(this.volumesVmFilter.value);
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
        if (this.toggleNetworkAdvancedBtn) {
            this.toggleNetworkAdvancedBtn.addEventListener('click', () => {
                if (!this.networkAdvancedFields) return;
                const isHidden = this.networkAdvancedFields.classList.toggle('hidden');
                if (this.networkAdvancedArrow) {
                    this.networkAdvancedArrow.textContent = isHidden ? '▶' : '▼';
                }
            });
        }
        if (this.refreshNetworksBtn) {
            this.refreshNetworksBtn.addEventListener('click', () => {
                this.loadNetworks(this.networksVmFilter ? this.networksVmFilter.value : '');
            });
        }
        if (this.networksVmFilter) {
            this.networksVmFilter.addEventListener('change', () => {
                this.handleDockerVmChange(this.networksVmFilter.value);
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

        // Eventos de Gestión de Docker Swarm Clusters
        if (this.openCreateClusterBtn) {
            this.openCreateClusterBtn.addEventListener('click', () => this.openCreateClusterModal());
        }
        if (this.refreshClustersBtn) {
            this.refreshClustersBtn.addEventListener('click', () => this.loadClusters());
        }
        if (this.backToClustersBtn) {
            this.backToClustersBtn.addEventListener('click', () => this.backToClustersList());
        }
        if (this.refreshClusterDetailBtn) {
            this.refreshClusterDetailBtn.addEventListener('click', () => {
                if (this.activeClusterId) {
                    this.openClusterDetails(this.activeClusterId);
                }
            });
        }
        if (this.closeCreateClusterModalBtn) {
            this.closeCreateClusterModalBtn.addEventListener('click', () => this.closeCreateClusterModal());
        }
        if (this.cancelCreateClusterBtn) {
            this.cancelCreateClusterBtn.addEventListener('click', () => this.closeCreateClusterModal());
        }
        if (this.refreshClusterVmsBtn) {
            this.refreshClusterVmsBtn.addEventListener('click', () => this.loadEligibleVMsForCluster());
        }
        if (this.createClusterForm) {
            this.createClusterForm.addEventListener('submit', (e) => {
                e.preventDefault();
                this.submitCreateCluster();
            });
        }
        if (this.closeRemoveClusterNodeModalBtn) {
            this.closeRemoveClusterNodeModalBtn.addEventListener('click', () => this.closeRemoveClusterNodeModal());
        }
        if (this.cancelRemoveClusterNodeBtn) {
            this.cancelRemoveClusterNodeBtn.addEventListener('click', () => this.closeRemoveClusterNodeModal());
        }
        if (this.confirmRemoveClusterNodeBtn) {
            this.confirmRemoveClusterNodeBtn.addEventListener('click', () => this.executeRemoveClusterNode());
        }

        // Eventos de Redes dentro del Asistente de Contenedores
        if (this.wizAddNetworkRowBtn) {
            this.wizAddNetworkRowBtn.addEventListener('click', () => {
                this.addWizNetworkRow();
            });
        }
        if (this.wizAddNetworkBtn) {
            this.wizAddNetworkBtn.addEventListener('click', () => {
                const vm = this.wizVmSelect ? this.wizVmSelect.value.trim() : '';
                if (!vm) {
                    this.showWizFeedback('Selecciona una máquina virtual primero.', 'error');
                    return;
                }
                this.openCreateNetworkDialog(vm, (newNetName) => {
                    this.addWizNetworkRow(newNetName);
                });
            });
        }

        if (this.wizVmSelect) {
            this.wizVmSelect.addEventListener('change', () => {
                const currentVm = this.wizVmSelect.value.trim();
                if (this.wizardSource === 'images') {
                    this.loadWizardImages();
                }
                this.updateWizardNetworks(currentVm);
            });
        }

        // Cerrar menús desplegables al hacer click fuera
        document.addEventListener('click', (e) => {
            if (!e.target.closest('.card-dropdown')) {
                document.querySelectorAll('.card-dropdown-menu.show').forEach((m) => m.classList.remove('show'));
                document.querySelectorAll('.btn-dots.active').forEach((b) => b.classList.remove('active'));
            }
        });

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
                this.handleDockerVmChange(this.imagesVmFilter.value);
            });
        }

        if (this.imagesSubtabBtns) {
            this.imagesSubtabBtns.forEach((btn) => {
                btn.addEventListener('click', () => {
                    this.switchImagesSubtab(btn.dataset.imagestab);
                });
            });
        }
        if (this.refreshImagesBtn) {
            this.refreshImagesBtn.addEventListener('click', () => {
                if (this.activeImagesSubtab === 'tars') {
                    this.loadExportedTars(this.imagesVmFilter ? this.imagesVmFilter.value : '');
                }
            });
        }
        if (this.imagesVmFilter) {
            this.imagesVmFilter.addEventListener('change', () => {
                if (this.activeImagesSubtab === 'tars') {
                    this.loadExportedTars(this.imagesVmFilter.value);
                }
            });
        }

        // Eventos de Docker Hub (Credenciales y Publicación)
        if (this.openDockerHubCredsBtn) {
            this.openDockerHubCredsBtn.addEventListener('click', () => this.openDockerHubCredsModal(false));
        }
        if (this.closeDockerHubCredsModalBtn) {
            this.closeDockerHubCredsModalBtn.addEventListener('click', () => this.closeDockerHubCredsModal());
        }
        if (this.cancelDockerHubCredsBtn) {
            this.cancelDockerHubCredsBtn.addEventListener('click', () => this.closeDockerHubCredsModal());
        }
        if (this.saveDockerHubCredsBtn) {
            this.saveDockerHubCredsBtn.addEventListener('click', () => this.saveDockerHubCredentials());
        }
        if (this.dockerHubUsernameInput) {
            this.dockerHubUsernameInput.addEventListener('keydown', (e) => {
                if (e.key === 'Enter') {
                    if (this.dockerHubTokenInput && !this.dockerHubTokenInput.value) {
                        this.dockerHubTokenInput.focus();
                    } else {
                        this.saveDockerHubCredentials();
                    }
                }
            });
        }
        if (this.dockerHubTokenInput) {
            this.dockerHubTokenInput.addEventListener('keydown', (e) => {
                if (e.key === 'Enter') {
                    this.saveDockerHubCredentials();
                }
            });
        }
        if (this.closeDockerHubPublishModalBtn) {
            this.closeDockerHubPublishModalBtn.addEventListener('click', () => this.closeDockerHubPublishModal());
        }
        if (this.cancelDockerHubPublishBtn) {
            this.cancelDockerHubPublishBtn.addEventListener('click', () => this.closeDockerHubPublishModal());
        }
        if (this.changeDockerHubCredsBtn) {
            this.changeDockerHubCredsBtn.addEventListener('click', () => {
                this.closeDockerHubPublishModal(true);
                this.openDockerHubCredsModal(true);
            });
        }
        if (this.publishRepoInput) {
            this.publishRepoInput.addEventListener('input', () => this.updateDockerHubTargetPreview());
            this.publishRepoInput.addEventListener('keydown', (e) => {
                if (e.key === 'Enter' && this.publishTagInput) {
                    this.publishTagInput.focus();
                }
            });
        }
        if (this.publishTagInput) {
            this.publishTagInput.addEventListener('input', () => this.updateDockerHubTargetPreview());
            this.publishTagInput.addEventListener('keydown', (e) => {
                if (e.key === 'Enter') {
                    this.submitDockerHubPublish();
                }
            });
        }
        if (this.submitDockerHubPublishBtn) {
            this.submitDockerHubPublishBtn.addEventListener('click', () => this.submitDockerHubPublish());
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
        }
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
            (this.commitContainerModal && !this.commitContainerModal.classList.contains('hidden')) ||
            (this.createClusterModal && !this.createClusterModal.classList.contains('hidden')) ||
            (this.removeClusterNodeModal && !this.removeClusterNodeModal.classList.contains('hidden'));
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
        const isDocker = (view === 'docker' || view === 'containers' || view === 'images' || view === 'volumes' || view === 'networks' || view === 'clusters');

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
            if (view === 'images' || view === 'volumes' || view === 'networks' || view === 'clusters') {
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
            } else if (subview === 'clusters') {
                s.classList.toggle('active', s.id === 'sidebar-clusters-view');
            } else {
                s.classList.remove('active');
            }
        });

        const activeVm = this.selectedDockerVm || '';
        if (subview === 'containers') {
            this.loadContainers(activeVm);
        } else if (subview === 'images') {
            if (this.activeImagesSubtab === 'exports') {
                this.loadExportedTars(activeVm);
            } else {
                this.loadImages(activeVm);
            }
        } else if (subview === 'volumes') {
            this.loadVolumes(activeVm);
        } else if (subview === 'networks') {
            this.loadNetworks(activeVm);
        } else if (subview === 'clusters') {
            if (this.activeClusterId && this.clusterDetailPanel && !this.clusterDetailPanel.classList.contains('hidden')) {
                this.openClusterDetails(this.activeClusterId);
            } else {
                this.loadClusters();
            }
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

    buildVMCardHTML(vm, clusterNodeContext = null) {
        const vmName = vm.name || vm.vmName || '';
        const stateLower = (vm.state || vm.vmState || '').toLowerCase();
        const isRunning = stateLower === 'running' || stateLower === 'ejecutando';
        const stateColor = isRunning ? 'var(--accent-green)' : stateLower === 'poweroff' ? 'var(--accent-red)' : 'var(--accent-yellow)';
        const templateLabel = vm.templateName || vm.os || 'Plantilla Base';
        const netName = vm.networkName || vm.network || 'vessel-net';
        const privateIp = vm.privateIp || vm.internalIp || vm.ip || '';

        let ipDisplayHtml = '';
        if (privateIp) {
            ipDisplayHtml = `<span style="color:var(--accent-green);font-family:monospace">${this.escapeHTML(privateIp)}</span>${!isRunning ? ' <span style="color:var(--text-muted);font-size:11px">(última)</span>' : ''}`;
        } else if (isRunning) {
            ipDisplayHtml = `<span style="color:var(--accent-yellow)" id="vm-card-ip-${this.escapeAttr(vmName)}">Detectando DHCP...</span>`;
        } else {
            ipDisplayHtml = `<span style="color:var(--text-muted)">Sin asignar (Apagada)</span>`;
        }

        let swarmBadgesHtml = '';
        let swarmDetailsHtml = '';
        let swarmRetryHtml = '';

        if (clusterNodeContext) {
            const isMgr = (clusterNodeContext.role || '').toLowerCase() === 'manager';
            const roleLabel = isMgr ? (clusterNodeContext.isLeader ? 'Manager / Leader' : 'Manager') : 'Worker';
            const swarmSt = clusterNodeContext.swarmStatus || (isRunning ? 'Ready' : 'Down');
            const swarmStLower = swarmSt.toLowerCase();
            const pillClass = swarmStLower === 'ready' ? 'ready' : (swarmStLower === 'down' || swarmStLower === 'error') ? 'down' : 'warning';

            swarmBadgesHtml = `
                <span class="cluster-role-badge ${isMgr ? 'role-manager' : 'role-worker'}">${this.escapeHTML(roleLabel)}</span>
                <span class="swarm-node-pill ${pillClass}">Swarm: ${this.escapeHTML(swarmSt)}</span>
            `;

            swarmDetailsHtml = `
                <span><strong>Rol Swarm:</strong> <span style="color:#7dd3fc">${this.escapeHTML(roleLabel)}</span></span>
                <span><strong>Estado Nodo:</strong> <span style="color:${pillClass === 'ready' ? 'var(--accent-green)' : 'var(--accent-red)'}">${this.escapeHTML(swarmSt)} (${this.escapeHTML(clusterNodeContext.availability || 'Active')})</span></span>
            `;

            if (clusterNodeContext.joinError) {
                swarmRetryHtml = `
                <div class="cluster-warning-box" style="margin-top:4px;display:flex;align-items:center;justify-content:space-between;gap:8px;flex-wrap:wrap">
                    <span style="font-size:11px;color:#fca5a5">${this.escapeHTML(clusterNodeContext.joinError)}</span>
                    <button type="button" class="btn btn-warning" style="padding:4px 8px;font-size:11px" onclick="window.terminalApp.retryClusterWorker('${this.escapeAttr(clusterNodeContext.clusterId)}', '${this.escapeAttr(vmName)}', this)">
                        <span class="btn-spinner hidden" aria-hidden="true"></span>
                        <span class="btn-text">Reintentar unión</span>
                    </button>
                </div>`;
            }
        }

        const cardDomId = clusterNodeContext
            ? `cluster-vm-card-${this.escapeAttr(vmName)}`
            : `vm-card-${this.escapeAttr(vmName)}`;

        return `
        <div class="vm-card" id="${cardDomId}">
            <div class="vm-card-header">
                <div class="vm-title-group" style="display: flex; align-items: center; gap: 8px; min-width: 0; flex-wrap: wrap;">
                    <span class="vm-name">${this.escapeHTML(vmName)}</span>
                    <span class="vm-os">${this.escapeHTML(templateLabel)}</span>
                    ${swarmBadgesHtml}
                </div>
                <div class="card-dropdown">
                    <button type="button" class="btn-dots" title="Más opciones" onclick="event.stopPropagation(); window.terminalApp.toggleVmMenu(this)">
                        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                            <circle cx="12" cy="12" r="1.5"></circle>
                            <circle cx="19" cy="12" r="1.5"></circle>
                            <circle cx="5" cy="12" r="1.5"></circle>
                        </svg>
                    </button>
                    <div class="card-dropdown-menu">
                        <button type="button" class="dropdown-item" onclick="window.terminalApp.viewVirtualMachine('${this.escapeAttr(vmName)}')">
                            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><line x1="12" y1="16" x2="12" y2="12"/><line x1="12" y1="8" x2="12.01" y2="8"/></svg>
                            <span>Detalles y Red</span>
                        </button>
                        <button type="button" class="dropdown-item" onclick="window.terminalApp.openVMContainersFromCard('${this.escapeAttr(vmName)}')">
                            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="2" y="7" width="20" height="14" rx="2" ry="2"/><path d="M16 21V5a2 2 0 0 0-2-2h-4a2 2 0 0 0-2 2v16"/></svg>
                            <span>Contenedores Docker</span>
                        </button>
                        ${isRunning ? `
                        <button type="button" class="dropdown-item" onclick="window.terminalApp.forcePoweroffVM('${this.escapeAttr(vmName)}', this)">
                            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M18.36 6.64a9 9 0 1 1-12.73 0"/><line x1="12" y1="2" x2="12" y2="12"/></svg>
                            <span>Forzar apagado</span>
                        </button>` : ''}
                        <div class="dropdown-divider"></div>
                        ${clusterNodeContext ? `
                        <button type="button" class="dropdown-item danger" onclick="window.terminalApp.openRemoveClusterNodeModal('${this.escapeAttr(clusterNodeContext.clusterId)}', '${this.escapeAttr(vmName)}', '${this.escapeAttr(clusterNodeContext.role)}', '${this.escapeAttr(clusterNodeContext.clusterName)}')">
                            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4"/><polyline points="16 17 21 12 16 7"/><line x1="21" y1="12" x2="9" y2="12"/></svg>
                            <span>Eliminar del cluster</span>
                        </button>` : `
                        <button type="button" class="dropdown-item danger" onclick="window.terminalApp.openDeleteVMModal('${this.escapeAttr(vmName)}')">
                            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="3 6 5 6 21 6"/><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/></svg>
                            <span>Eliminar</span>
                        </button>`}
                    </div>
                </div>
            </div>
            <div class="vm-card-details">
                <span><strong>CPU:</strong> ${vm.cores || 2} Cores</span>
                <span><strong>RAM:</strong> ${vm.memoryMB || 4096} MB</span>
                <span><strong>Red:</strong> ${this.escapeHTML(netName)}</span>
                <span><strong>IP Privada:</strong> ${ipDisplayHtml}</span>
                <span><strong>Usuario SSH:</strong> ${this.escapeHTML(vm.sshUser || 'bootstrap')}</span>
                <span><strong>SSH Port:</strong> ${vm.hostPort ? `${vm.host || ''}:${vm.hostPort}` : 'NAT'}</span>
                <span><strong>Estado VM:</strong> <span style="color:${stateColor}">${this.escapeHTML(vm.state || vm.vmState || 'Creada')}</span></span>
                ${swarmDetailsHtml}
            </div>
            ${swarmRetryHtml}
            <div class="vm-card-delete-loader hidden">
                <span class="vm-card-spinner"></span>
                <span>Procesando...</span>
            </div>
            <div class="vm-card-actions">
                ${isRunning ? `
                <button type="button" class="btn btn-primary terminal-vm" data-id="${this.escapeAttr(vmName)}" title="Abrir terminal SSH interactivo a esta VM" onclick="window.terminalApp.connectTerminalToVM('${this.escapeAttr(vmName)}')">
                    <svg viewBox="0 0 24 24" width="12" height="12" stroke="currentColor" stroke-width="2" fill="none"><polyline points="4 17 10 11 4 5"/><line x1="12" y1="19" x2="20" y2="19"/></svg>
                    <span>Terminal</span>
                </button>
                <button type="button" class="btn btn-stop-vm stop-vm" data-id="${this.escapeAttr(vmName)}" title="Apagado limpio ACPI" onclick="window.terminalApp.stopVM('${this.escapeAttr(vmName)}', this)">
                    <span class="btn-spinner hidden" aria-hidden="true"></span>
                    <span class="btn-text">Apagar</span>
                </button>` : `
                <button type="button" class="btn btn-success start-vm" data-id="${this.escapeAttr(vmName)}" title="Iniciar máquina virtual" onclick="window.terminalApp.startVM('${this.escapeAttr(vmName)}', this)">
                    <span class="btn-spinner hidden" aria-hidden="true"></span>
                    <span class="btn-text">Iniciar</span>
                </button>`}
            </div>
        </div>
        `;
    }

    renderVMList(vms) {
        if (!this.vmListEl) return;
        this.cachedVMs = Array.isArray(vms) ? vms : [];
        if (!Array.isArray(vms) || vms.length === 0) {
            this.vmListEl.innerHTML = `<div class="empty-vm-list">No hay máquinas virtuales creadas o registradas.</div>`;
            return;
        }

        this.vmListEl.innerHTML = vms.map((vm) => this.buildVMCardHTML(vm, null)).join('');

        // Descubrir automáticamente en segundo plano la IP privada de las VMs encendidas que aún no tengan IP registrada
        vms.forEach((vm) => {
            const stateLower = (vm.state || '').toLowerCase();
            const isRunning = stateLower === 'running' || stateLower === 'ejecutando';
            if (isRunning && !(vm.privateIp || vm.ip)) {
                this.fetchVMNetworkInfoSilent(vm.name);
            }
        });
    }

    async fetchVMNetworkInfoSilent(vmName) {
        try {
            const res = await fetch(`/api/vms/${encodeURIComponent(vmName)}/network-info`);
            if (!res.ok) return;
            const netInfo = await res.json();
            const ipEl = document.getElementById(`vm-card-ip-${vmName}`);
            if (ipEl && netInfo && netInfo.privateIp) {
                ipEl.style.color = 'var(--accent-green)';
                ipEl.style.fontFamily = 'monospace';
                ipEl.textContent = netInfo.privateIp;
                if (Array.isArray(this.cachedVMs)) {
                    const target = this.cachedVMs.find((v) => v.name === vmName);
                    if (target) {
                        target.privateIp = netInfo.privateIp;
                        target.ip = netInfo.privateIp;
                    }
                }
            } else if (ipEl) {
                ipEl.textContent = 'Pendiente DHCP';
            }
        } catch (_) {
            // Silencioso en segundo plano
        }
    }

    async viewVirtualMachine(id) {
        this.closeCardDropdowns();
        if (!this.vmDetailsModal || !this.vmDetailsContent) return;
        this.vmDetailsContent.innerHTML = `<div style="grid-column:1/-1;text-align:center;padding:16px"><span class="btn-spinner"></span> Cargando detalles y topología de red...</div>`;
        this.vmDetailsModal.classList.remove('hidden');
        this.vmDetailsModal.setAttribute('aria-hidden', 'false');

        try {
            const [res, netRes] = await Promise.all([
                fetch(`/api/vms/${encodeURIComponent(id)}`),
                fetch(`/api/vms/${encodeURIComponent(id)}/network-info`).catch(() => null),
            ]);
            if (!res.ok) {
                const text = await res.text();
                this.vmDetailsContent.innerHTML = `<div style="grid-column:1/-1;color:var(--accent-red)">Error al consultar VM: ${this.parseApiError(text)}</div>`;
                return;
            }
            const vm = await res.json();
            const netInfo = netRes && netRes.ok ? await netRes.json() : {};
            this.renderVMDetailsContent(vm, netInfo);
        } catch (e) {
            console.warn('Error al ver VM:', e);
            this.vmDetailsContent.innerHTML = `<div style="grid-column:1/-1;color:var(--accent-red)">No se pudo cargar la información de la máquina virtual.</div>`;
        }
    }

    renderVMDetailsContent(vm, netInfo = {}) {
        if (!this.vmDetailsContent) return;
        const cores = vm.cores > 0 ? vm.cores : 2;
        const memoryMB = vm.memoryMB > 0 ? vm.memoryMB : 4096;
        const templateName = vm.templateName || vm.os || 'Plantilla Base';
        const state = netInfo.state || vm.state || 'Creada';
        const isRunning = state.toLowerCase() === 'running' || state.toLowerCase() === 'ejecutando';

        const netName = netInfo.networkName || vm.networkName || vm.network || 'vessel-net';
        const netCidr = netInfo.networkCidr || vm.networkCidr || '10.250.0.0/24';
        const privateIp = netInfo.privateIp || vm.privateIp || vm.ip || '';
        const mac1 = netInfo.macAddress1 || vm.macAddress1 || 'N/A';
        const mac2 = netInfo.macAddress2 || vm.macAddress2 || 'N/A';
        const adapter1State = netInfo.adapter1State || vm.adapter1State || (isRunning ? (privateIp ? 'connected' : 'pending_dhcp') : 'disconnected');
        const adapter2State = netInfo.adapter2State || vm.adapter2State || (isRunning ? 'connected' : 'disconnected');
        const lastDiscovery = netInfo.lastNetworkDiscovery || vm.lastNetworkDiscovery || '';

        const otherVMs = (this.cachedVMs || []).filter((v) => v.name !== vm.name);
        const targetOptionsHtml = otherVMs.map((v) => {
            const vIp = v.privateIp || v.ip ? ` (${v.privateIp || v.ip})` : '';
            return `<option value="${this.escapeAttr(v.name)}">${this.escapeHTML(v.name)}${this.escapeHTML(vIp)}</option>`;
        }).join('');

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
                <span class="vm-detail-key">Estado Actual</span>
                <span class="vm-detail-val" style="color:${isRunning ? 'var(--accent-green)' : 'var(--accent-red)'}">${this.escapeHTML(state)}</span>
            </div>
            <div class="vm-detail-item">
                <span class="vm-detail-key">Usuario SSH Dedicado</span>
                <span class="vm-detail-val" style="color:var(--accent-yellow)">${this.escapeHTML(vm.sshUser || 'bootstrap')}</span>
            </div>
            <div class="vm-detail-item">
                <span class="vm-detail-key">Red Compartida (Adaptador 1)</span>
                <span class="vm-detail-val">${this.escapeHTML(netName)} <span style="color:var(--text-muted);font-size:11px">(${this.escapeHTML(netCidr)})</span></span>
            </div>
            <div class="vm-detail-item">
                <span class="vm-detail-key" style="display:flex;justify-content:space-between;align-items:center">
                    <span>IP Privada (${this.escapeHTML(netName)})</span>
                    ${isRunning ? `<button type="button" class="btn btn-secondary" style="padding:2px 6px;font-size:10px" onclick="window.terminalApp.refreshVMNetworkInfo('${this.escapeAttr(vm.name)}', this)">Refrescar IP</button>` : ''}
                </span>
                <span class="vm-detail-val" style="font-family:monospace;color:${privateIp ? 'var(--accent-green)' : 'var(--accent-yellow)'}">
                    ${privateIp ? this.escapeHTML(privateIp) : (isRunning ? 'Pendiente de concesión DHCP' : 'Sin IP activa (MV Apagada)')}
                </span>
            </div>
            <div class="vm-detail-item">
                <span class="vm-detail-key">Adaptador 1 (NAT Network)</span>
                <span class="vm-detail-val" style="font-size:12px;font-family:monospace">${this.escapeHTML(mac1)} · <span style="color:${adapter1State === 'connected' ? 'var(--accent-green)' : 'var(--text-muted)'}">${this.escapeHTML(adapter1State)}</span></span>
            </div>
            <div class="vm-detail-item">
                <span class="vm-detail-key">Adaptador 2 (SSH NAT :${vm.hostPort || 22})</span>
                <span class="vm-detail-val" style="font-size:12px;font-family:monospace">${this.escapeHTML(mac2)} · <span style="color:${adapter2State === 'connected' ? 'var(--accent-green)' : 'var(--text-muted)'}">${this.escapeHTML(adapter2State)}</span></span>
            </div>
            <div class="vm-detail-item full-width">
                <span class="vm-detail-key">Acceso SSH Administrativo (NIC 2 --natpf2)</span>
                <span class="vm-detail-val" style="font-family:monospace">${vm.hostPort ? `${this.escapeHTML(vm.host || 'Host')}:${vm.hostPort} → :${vm.guestPort || 22}` : 'NAT :22'} ${lastDiscovery ? `<span style="color:var(--text-muted);font-size:11px;margin-left:8px">(Detección: ${this.escapeHTML(lastDiscovery)})</span>` : ''}</span>
            </div>
            ${netInfo.discoveryError && isRunning ? `
            <div class="vm-detail-item full-width" style="border-color:rgba(245,158,11,0.4)">
                <span class="vm-detail-key" style="color:var(--accent-yellow)">Aviso de Descubrimiento de Red</span>
                <span class="vm-detail-val" style="font-size:12px;color:var(--accent-yellow)">${this.escapeHTML(netInfo.discoveryError)}</span>
            </div>` : ''}
            ${isRunning ? `
            <div class="vm-detail-item full-width">
                <span class="vm-detail-key">Verificar Comunicación entre VMs (${this.escapeHTML(netName)})</span>
                <div style="display:flex;gap:8px;margin-top:6px;flex-wrap:wrap;align-items:center">
                    ${otherVMs.length > 0 ? `
                    <select id="vm-conn-target-select" style="flex:1;min-width:160px;padding:6px 8px;border-radius:6px;background:var(--bg-primary);color:var(--text-primary);border:1px solid var(--border)" onchange="const inp=document.getElementById('vm-conn-target-input'); if(inp && this.value) inp.value=this.value;">
                        <option value="">-- Seleccionar otra VM --</option>
                        ${targetOptionsHtml}
                    </select>` : ''}
                    <input id="vm-conn-target-input" type="text" placeholder="Nombre de VM o IP (ej: 10.250.0.12)" value="${otherVMs.length > 0 ? this.escapeAttr(otherVMs[0].name) : ''}" style="flex:1;min-width:170px;padding:6px 8px;border-radius:6px;background:var(--bg-primary);color:var(--text-primary);border:1px solid var(--border)">
                    <button type="button" class="btn btn-primary" style="padding:6px 12px;font-size:12px" onclick="window.terminalApp.testVMConnectivity('${this.escapeAttr(vm.name)}', this)">
                        Probar Conectividad
                    </button>
                </div>
                <div id="vm-conn-result-box" class="hidden" style="margin-top:8px;padding:8px 10px;border-radius:6px;font-size:12px;font-family:monospace;white-space:pre-wrap;background:rgba(0,0,0,0.25);border:1px solid var(--border)"></div>
            </div>` : ''}
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
    }

    async refreshVMNetworkInfo(vmName, btnEl) {
        if (btnEl) {
            btnEl.disabled = true;
            btnEl.textContent = 'Detectando...';
        }
        try {
            const [vmRes, netRes] = await Promise.all([
                fetch(`/api/vms/${encodeURIComponent(vmName)}`),
                fetch(`/api/vms/${encodeURIComponent(vmName)}/network-info`, { method: 'POST' }),
            ]);
            const vm = vmRes.ok ? await vmRes.json() : { name: vmName };
            const netInfo = netRes.ok ? await netRes.json() : {};
            this.renderVMDetailsContent(vm, netInfo);
            this.loadVirtualMachines();
        } catch (e) {
            console.warn('Error refrescando red de VM:', e);
            if (btnEl) {
                btnEl.disabled = false;
                btnEl.textContent = 'Refrescar IP';
            }
        }
    }

    async testVMConnectivity(sourceVmName, btnEl) {
        const inputEl = document.getElementById('vm-conn-target-input');
        const resultBox = document.getElementById('vm-conn-result-box');
        if (!inputEl || !resultBox) return;
        const target = (inputEl.value || '').trim();
        if (!target) {
            resultBox.classList.remove('hidden');
            resultBox.style.color = 'var(--accent-red)';
            resultBox.textContent = 'Ingresa el nombre de una VM o una dirección IP privada destino.';
            return;
        }
        if (btnEl) {
            btnEl.disabled = true;
            btnEl.textContent = 'Probando...';
        }
        resultBox.classList.remove('hidden');
        resultBox.style.color = 'var(--text-secondary)';
        resultBox.textContent = `Verificando ruta en red compartida desde '${sourceVmName}' hacia '${target}'...`;

        try {
            const res = await fetch(`/api/vms/${encodeURIComponent(sourceVmName)}/connectivity`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ target, targetPort: 22 }),
            });
            const data = await res.json();
            if (!res.ok) {
                resultBox.style.color = 'var(--accent-red)';
                resultBox.textContent = `Error: ${data.error || 'No se pudo completar la prueba'}`;
            } else if (data.reachable) {
                resultBox.style.color = 'var(--accent-green)';
                const latencyTxt = data.latencyMs ? ` · Latencia media: ${data.latencyMs.toFixed(2)} ms` : '';
                resultBox.textContent = `✓ Conectividad exitosa (${data.method}) hacia ${data.targetVm ? `${data.targetVm} (${data.targetIp})` : data.targetIp}${latencyTxt}\n${data.output || ''}`.trim();
            } else {
                resultBox.style.color = 'var(--accent-red)';
                resultBox.textContent = `✕ Sin conectividad hacia ${data.targetIp}: ${data.error || 'Host inalcanzable'}\n${data.output || ''}`.trim();
            }
        } catch (e) {
            resultBox.style.color = 'var(--accent-red)';
            resultBox.textContent = `Error de red al verificar conectividad: ${e.message || e}`;
        } finally {
            if (btnEl) {
                btnEl.disabled = false;
                btnEl.textContent = 'Probar Conectividad';
            }
        }
    }

    closeDetailsModal() {
        if (!this.vmDetailsModal) return;
        this.vmDetailsModal.classList.add('hidden');
        this.vmDetailsModal.setAttribute('aria-hidden', 'true');
    }

    initVMCreateProgress(vmName) {
        if (!this.vmCreateProgress) return;
        this.vmCreateProgress.classList.remove('hidden');
        if (this.vmProgressBarFill) {
            this.vmProgressBarFill.className = 'vm-progress-bar-fill';
            this.vmProgressBarFill.style.width = '5%';
        }
        if (this.vmProgressBadge) {
            this.vmProgressBadge.className = 'vm-progress-badge';
            this.vmProgressBadge.textContent = 'Paso 1 de 6';
        }
        if (this.vmProgressStageTitle) {
            this.vmProgressStageTitle.textContent = `Aprovisionando '${vmName}'...`;
        }
        if (this.vmProgressMsg) {
            this.vmProgressMsg.textContent = 'Iniciando proceso de aprovisionamiento...';
        }
        const stepEls = this.vmCreateProgress.querySelectorAll('.vm-pstep');
        stepEls.forEach((el) => {
            el.className = 'vm-pstep';
            const icon = el.querySelector('.vm-pstep-icon');
            if (icon) icon.textContent = '○';
        });
        const firstStep = this.vmCreateProgress.querySelector('.vm-pstep[data-step="1"]');
        if (firstStep) {
            firstStep.className = 'vm-pstep active';
            const icon = firstStep.querySelector('.vm-pstep-icon');
            if (icon) icon.innerHTML = '<span class="vm-spinner-mini"></span>';
        }
    }

    updateVMCreateProgress(evt) {
        if (!this.vmCreateProgress) return;
        const { stage, step, total, message, error } = evt;

        if (this.vmProgressMsg && message) {
            this.vmProgressMsg.textContent = message;
        }

        if (stage === 'error') {
            if (this.vmProgressBarFill) {
                this.vmProgressBarFill.className = 'vm-progress-bar-fill error';
            }
            if (this.vmProgressBadge) {
                this.vmProgressBadge.textContent = 'Error';
                this.vmProgressBadge.className = 'vm-progress-badge error';
            }
            if (this.vmProgressStageTitle) {
                this.vmProgressStageTitle.textContent = 'Fallo en la creación';
            }
            const currentStepEl = this.vmCreateProgress.querySelector(`.vm-pstep[data-step="${step || 1}"]`);
            if (currentStepEl) {
                currentStepEl.className = 'vm-pstep failed';
                const icon = currentStepEl.querySelector('.vm-pstep-icon');
                if (icon) icon.textContent = '✕';
            }
            this.showVMCreateFeedback(`Error al crear VM: ${error || message}`, 'error');
            return;
        }

        const pct = total ? Math.min(Math.round((step / total) * 100), 100) : 10;
        if (this.vmProgressBarFill) {
            this.vmProgressBarFill.style.width = `${pct}%`;
            if (stage === 'done') {
                this.vmProgressBarFill.className = 'vm-progress-bar-fill success';
            }
        }

        if (this.vmProgressBadge) {
            if (stage === 'done') {
                this.vmProgressBadge.textContent = 'Completado (100%)';
                this.vmProgressBadge.className = 'vm-progress-badge success';
            } else if (step && total) {
                this.vmProgressBadge.textContent = `Paso ${step} de ${total} (${pct}%)`;
                this.vmProgressBadge.className = 'vm-progress-badge';
            }
        }

        if (this.vmProgressStageTitle) {
            if (stage === 'done') {
                this.vmProgressStageTitle.textContent = '¡Máquina virtual lista!';
            } else {
                const titles = {
                    validating: '1. Validando y generando claves Ed25519...',
                    creating_vm: '2. Creando VM en VirtualBox...',
                    starting_vm: '3. Iniciando máquina virtual...',
                    waiting_ssh: '4. Esperando servicio SSH...',
                    configuring_user: '5. Configurando usuario y accesos...',
                    configuring_docker: '6. Configurando Docker y servicios...'
                };
                this.vmProgressStageTitle.textContent = titles[stage] || `Paso ${step}...`;
            }
        }

        const stepEls = this.vmCreateProgress.querySelectorAll('.vm-pstep');
        stepEls.forEach((el) => {
            const s = parseInt(el.dataset.step, 10);
            const icon = el.querySelector('.vm-pstep-icon');
            if (stage === 'done' || s < step) {
                el.className = 'vm-pstep completed';
                if (icon) icon.textContent = '✓';
            } else if (s === step) {
                el.className = 'vm-pstep active';
                if (icon) icon.innerHTML = '<span class="vm-spinner-mini"></span>';
            } else {
                el.className = 'vm-pstep pending';
                if (icon) icon.textContent = '○';
            }
        });
    }

    hideVMCreateProgress() {
        if (!this.vmCreateProgress) return;
        this.vmCreateProgress.classList.add('hidden');
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
        this.hideVMFeedback();
        this.initVMCreateProgress(name);

        try {
            const payload = { name, templateId, cores, memoryMB };
            const res = await fetch('/api/vms', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'Accept': 'text/event-stream, application/json'
                },
                body: JSON.stringify(payload),
            });

            const contentType = res.headers.get('content-type') || '';
            if (contentType.includes('text/event-stream') && res.body) {
                const reader = res.body.getReader();
                const decoder = new TextDecoder();
                let buffer = '';
                let createdVMData = null;
                let hasError = false;

                while (true) {
                    const { done, value } = await reader.read();
                    if (done) break;
                    buffer += decoder.decode(value, { stream: true });
                    const lines = buffer.split('\n');
                    buffer = lines.pop() || '';

                    for (const line of lines) {
                        const trimmed = line.trim();
                        if (trimmed.startsWith('data:')) {
                            try {
                                const evt = JSON.parse(trimmed.substring(5).trim());
                                this.updateVMCreateProgress(evt);
                                if (evt.stage === 'error') {
                                    hasError = true;
                                } else if (evt.stage === 'done') {
                                    createdVMData = evt.vm || { name, templateName: templateId };
                                }
                            } catch (parseErr) {
                                console.warn('Error parseando SSE de creacion de VM:', parseErr, trimmed);
                            }
                        }
                    }
                }

                if (!hasError && createdVMData) {
                    this.showVMCreateFeedback(`¡Máquina virtual '${name}' creada y aprovisionada con éxito!`, 'success');
                    this.appendInfoLine(`[Crear VM] '${name}' creada exitosamente con plantilla '${createdVMData.templateName || templateId}'. Usuario: ${createdVMData.sshUser || 'propio'}, Puerto SSH: ${createdVMData.hostPort || 'NAT'}`);

                    setTimeout(() => {
                        this.vmFormWrap.classList.add('hidden');
                        this.vmForm.reset();
                        this.hideVMFeedback();
                        this.hideVMCreateProgress();
                    }, 2200);

                    await this.loadVirtualMachines();
                    await this.loadKeys();
                }
                return;
            }

            if (!res.ok) {
                const text = await res.text();
                const errMsg = this.parseApiError(text);
                this.updateVMCreateProgress({ stage: 'error', error: errMsg, message: errMsg });
                this.showVMCreateFeedback(`Error al crear VM: ${errMsg}`, 'error');
                return;
            }

            const created = await res.json();
            this.updateVMCreateProgress({ stage: 'done', step: 6, total: 6, message: `¡Máquina virtual '${name}' creada con éxito!` });
            this.showVMCreateFeedback(`¡Máquina virtual '${name}' creada y aprovisionada con éxito!`, 'success');
            this.appendInfoLine(`[Crear VM] '${name}' creada exitosamente con plantilla '${created.templateName || templateId}'. Usuario: ${created.sshUser || 'propio'}, Puerto SSH: ${created.hostPort || 'NAT'}`);

            setTimeout(() => {
                this.vmFormWrap.classList.add('hidden');
                this.vmForm.reset();
                this.hideVMFeedback();
                this.hideVMCreateProgress();
            }, 2200);

            await this.loadVirtualMachines();
            await this.loadKeys();
        } catch (err) {
            console.warn('Error al crear VM:', err);
            this.updateVMCreateProgress({ stage: 'error', error: err.message, message: 'No se pudo conectar al servidor para crear la MV.' });
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
        this.closeCardDropdowns();
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
        this.closeCardDropdowns();
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
            if (this.activeClusterId && this.clusterDetailPanel && !this.clusterDetailPanel.classList.contains('hidden')) {
                await this.openClusterDetails(this.activeClusterId);
            }
        }
    }

    async stopVM(name, btn) {
        this.closeCardDropdowns();
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
            if (this.activeClusterId && this.clusterDetailPanel && !this.clusterDetailPanel.classList.contains('hidden')) {
                await this.openClusterDetails(this.activeClusterId);
            }
        }
    }

    async forcePoweroffVM(name, btn) {
        this.closeCardDropdowns();
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
            if (this.activeClusterId && this.clusterDetailPanel && !this.clusterDetailPanel.classList.contains('hidden')) {
                await this.openClusterDetails(this.activeClusterId);
            }
        }
    }

    connectTerminalToVM(name) {
        this.clearOutput();
        this.appendInfoLine(`[Terminal VM] Conectando terminal SSH interactivo a '${name}'...`);
        this.toggleSidebar(false);
        this.currentVM = name;
        this.currentContainer = null;
        this.containerSessionOpened = false;
        if (this.targetUserHostEl) {
            this.targetUserHostEl.textContent = `[VM] ${name}`;
            this.targetUserHostEl.title = `Conectado a la máquina virtual ${name}`;
        }
        if (this.targetShellBadgeEl) {
            this.targetShellBadgeEl.textContent = 'bash';
            this.targetShellBadgeEl.className = 'badge-shell bash';
        }
        const promptChar = document.querySelector('#input-area .prompt-char');
        if (promptChar) {
            promptChar.textContent = '$';
        }
        this.currentPath = '';
        this.updateCurrentPath('~');
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
                    this.executeClearScreen();
                }
                break;
        }
    }

    getActivePromptText() {
        if (this.currentLineEl) {
            const curr = AnsiParser.stripAnsi(this.pendingText || this.currentLineEl.textContent || '').trim();
            if (curr && (curr.endsWith('>') || curr.endsWith('$') || curr.endsWith('#'))) {
                return curr;
            }
        }
        if (this.lastKnownPromptText) {
            return this.lastKnownPromptText;
        }
        const path = this.currentPath || '~';
        if (this.currentContainer) {
            return `root@${this.currentContainer.container}:${path}#`;
        }
        if (this.currentVM) {
            return `${this.currentVM}:${path}$`;
        }
        const shell = (this.currentConfig.sshShell || 'powershell').toLowerCase();
        if (shell === 'powershell' || shell === 'pwsh') {
            return `PS ${path}>`;
        }
        const user = this.currentConfig.sshUser || '';
        const host = this.currentConfig.sshHost || '';
        return user && host ? `${user}@${host} ${path}>` : `${path}>`;
    }

    clearResultsKeepPath() {
        const promptToKeep = this.getActivePromptText();
        this.outputEl.innerHTML = '';
        this.activeClasses = [];
        this.ansiRemainder = '';
        this.isClearingScreen = false;
        if (this.clearScreenTimer) {
            clearTimeout(this.clearScreenTimer);
            this.clearScreenTimer = null;
        }
        this.screenReplayIndex = null;
        this.ignoreEchoCommand = null;

        this.currentLineEl = this.createLineElement();
        this.pendingText = promptToKeep;
        const result = AnsiParser.renderToHTML(promptToKeep, []);
        this.currentLineEl.innerHTML = result.html;
        this.outputEl.appendChild(this.currentLineEl);
    }

    executeClearScreen() {
        this.clearResultsKeepPath();
        if (this.ws && this.ws.ws && this.ws.ws.readyState === WebSocket.OPEN) {
            this.isClearingScreen = true;
            if (this.clearScreenTimer) {
                clearTimeout(this.clearScreenTimer);
            }
            this.clearScreenTimer = setTimeout(() => {
                this.isClearingScreen = false;
                this.clearScreenTimer = null;
            }, 1200);
            const remoteClearCmd = (!this.currentVM && !this.currentContainer) ? 'cls\r' : 'clear\r';
            this.ws.sendInput(remoteClearCmd);
        }
    }

    submitCommand() {
        const command = this.inputEl.value;
        if (command.trim() === '') return;

        this.commandHistory.push(command);
        this.historyIndex = this.commandHistory.length;

        const cmdTrimmed = command.trim();
        const cmdLower = cmdTrimmed.toLowerCase();

        // clear / cls borra tanto los resultados locales como el buffer de pantalla remoto (ConPTY/PTY) y conserva la ruta/prompt actual
        if (cmdLower === 'cls' || cmdLower === 'clear') {
            this.executeClearScreen();
            this.inputEl.value = '';
            this.autoScroll = true;
            this.scrollToBottom();
            return;
        }

        if (this.isClearingScreen) {
            this.isClearingScreen = false;
            if (this.clearScreenTimer) {
                clearTimeout(this.clearScreenTimer);
                this.clearScreenTimer = null;
            }
        }
        this.screenReplayIndex = null;

        this.autoScroll = true;

        // Si tenemos una línea activa con el prompt en pantalla, completarla con el comando formateado
        if (this.currentLineEl) {
            const promptText = AnsiParser.stripAnsi(this.pendingText || this.currentLineEl.textContent || '').trim();
            if (promptText && (promptText.endsWith('>') || promptText.endsWith('$') || promptText.endsWith('#'))) {
                this.lastKnownPromptText = promptText;
                this.lastSubmittedPrompt = promptText;
                this.lastSubmittedCommand = cmdTrimmed;
                this.currentLineEl.innerHTML = AnsiParser.renderPromptAndCommand(promptText, cmdTrimmed);
                this.finishLine();
            } else {
                this.finishLine();
            }
        } else {
            const fallbackPrompt = this.getActivePromptText();
            this.lastKnownPromptText = fallbackPrompt;
            this.lastSubmittedPrompt = fallbackPrompt;
            this.lastSubmittedCommand = cmdTrimmed;
            this.currentLineEl = this.createLineElement();
            this.pendingText = `${fallbackPrompt} ${cmdTrimmed}`;
            this.currentLineEl.innerHTML = AnsiParser.renderPromptAndCommand(fallbackPrompt, cmdTrimmed);
            this.outputEl.appendChild(this.currentLineEl);
            this.finishLine();
        }

        // Si estamos dentro de un contenedor o VM y el usuario escribe 'exit', cerrar y volver al host automáticamente
        if (cmdLower === 'exit' && (this.currentContainer || this.currentVM)) {
            const targetLabel = this.currentContainer
                ? `contenedor '${this.currentContainer.container}'`
                : `VM '${this.currentVM}'`;
            this.currentContainer = null;
            this.containerSessionOpened = false;
            this.currentVM = null;
            this.ws.sendInput('exit\r');
            this.inputEl.value = '';
            this.appendInfoLine(`[Cerrando sesión en ${targetLabel} y volviendo al host principal...]`);
            setTimeout(() => this.connectCurrent(), 150);
            return;
        }

        this.ws.sendInput(cmdTrimmed + '\r');
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

        // Reensamblar secuencias ANSI que hayan quedado partidas entre fragmentos del WebSocket
        let streamData = (this.ansiRemainder || '') + rawData;
        const split = AnsiParser.splitIncompleteAnsi(streamData);
        this.ansiRemainder = split.remainder;
        streamData = split.complete;
        if (!streamData) return;

        const detectedPath = AnsiParser.extractPath(streamData);
        if (detectedPath) {
            this.updateCurrentPath(detectedPath);
        }

        if (this.isClearingScreen) {
            const plainChunk = AnsiParser.stripAnsi(streamData).trim();
            if (AnsiParser.hasClearScreen(streamData) || detectedPath || /[>$#]\s*$/.test(plainChunk)) {
                if (this.clearScreenTimer) {
                    clearTimeout(this.clearScreenTimer);
                }
                this.clearScreenTimer = setTimeout(() => {
                    this.isClearingScreen = false;
                    this.clearScreenTimer = null;
                }, 150);
            }
            return;
        }

        if (AnsiParser.hasClearScreen(streamData)) {
            this.clearResultsKeepPath();
            streamData = AnsiParser.removeClearScreen(streamData);
            if (!streamData || !AnsiParser.stripAnsi(streamData).trim()) return;
        }

        this.processText(streamData);
    }

    removeBackspaces(str) {
        if (!str || !str.includes('\b')) return str;
        const result = [];
        for (let i = 0; i < str.length; i++) {
            const ch = str[i];
            if (ch === '\b') {
                if (result.length > 0 && result[result.length - 1] !== '\n' && result[result.length - 1] !== '\r' && result[result.length - 1] !== '\x02') {
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

        // Limpiar secuencias de cursor/control ANSI antes de procesar \r y \n
        const prepared = AnsiParser.prepareStreamText(text);
        if (!prepared) return;

        const processed = this.removeBackspaces(prepared);
        const segments = processed.split('\x02');

        for (let sIdx = 0; sIdx < segments.length; sIdx++) {
            if (sIdx > 0) {
                // \x02 representa \x1b[H (Cursor Home a fila 1, columna 1 en ConPTY).
                // Si había una línea parcial sin salto de línea (ej. sondeo de ConPTY o prompt incompleto),
                // descartarla porque ConPTY va a repintar la pantalla desde la fila 1.
                if (this.currentLineEl) {
                    const partialPlain = AnsiParser.stripAnsi(this.pendingText || '').trim();
                    const isPrompt = partialPlain && (partialPlain.endsWith('>') || partialPlain.endsWith('$') || partialPlain.endsWith('#'));
                    this.currentLineEl.remove();
                    this.currentLineEl = null;
                    this.pendingText = '';
                    this.activeClasses = [];
                    if (isPrompt && this.outputEl.children.length === 0) {
                        this.lastKnownPromptText = partialPlain;
                    }
                }
                if (!this.currentVM && !this.currentContainer && this.outputEl.children.length > 0) {
                    this.screenReplayIndex = -1;
                }
            }

            const seg = segments[sIdx];
            if (!seg) continue;

            // Normalizar saltos de línea evitando borrar accidentalmente el contenido previo
            const normalized = seg.replace(/\r+\n/g, '\n');
            const lines = normalized.split('\n');

            for (let i = 0; i < lines.length; i++) {
                let linePart = lines[i];

                // Eliminar retornos de carro al final de la línea para no vaciar pendingText
                linePart = linePart.replace(/\r+$/, '');

                if (linePart.includes('\r')) {
                    const subParts = linePart.split(/\r+/);
                    const nonEmpty = subParts.filter((s) => AnsiParser.stripAnsi(s).trim() !== '');
                    if (nonEmpty.length > 0) {
                        linePart = nonEmpty[nonEmpty.length - 1];
                        this.pendingText = '';
                    } else {
                        linePart = subParts.join('');
                    }
                }

                if (linePart && AnsiParser.stripAnsi(linePart).length > 0) {
                    this.appendToCurrentLine(linePart);
                }

                if (i < lines.length - 1) {
                    this.finishLine();
                }
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

        const plain = AnsiParser.stripAnsi(this.pendingText).trim();
        if (plain && (plain.endsWith('>') || plain.endsWith('$') || plain.endsWith('#'))) {
            this.lastKnownPromptText = plain;
        }
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
                this.currentLineEl.remove();
            } else {
                let prev = this.currentLineEl.previousElementSibling;
                while (prev && (prev.textContent || '').trim() === '') {
                    const emptyPrev = prev;
                    prev = prev.previousElementSibling;
                    emptyPrev.remove();
                }

                const normCurr = this.normalizeLineForComparison(rawText);

                // Si ConPTY inició un repintado de pantalla completa desde \x1b[H (fila 1, col 1),
                // deduplicar las líneas que ya existen en el historial visible del DOM.
                if (this.screenReplayIndex !== null && this.screenReplayIndex !== undefined) {
                    const children = this.outputEl.children;
                    const existingCount = children.length - 1; // Excluir this.currentLineEl
                    if (existingCount <= 0) {
                        this.screenReplayIndex = null;
                    } else if (this.screenReplayIndex === -1) {
                        const searchStart = Math.max(0, existingCount - 200);
                        let matchedIdx = -1;
                        for (let k = searchStart; k < existingCount; k++) {
                            const normExisting = this.normalizeLineForComparison(children[k].textContent || '');
                            if (normExisting === normCurr) {
                                matchedIdx = k;
                                break;
                            }
                        }
                        if (matchedIdx !== -1) {
                            this.currentLineEl.remove();
                            this.currentLineEl = null;
                            this.pendingText = '';
                            this.screenReplayIndex = matchedIdx + 1;
                            return;
                        } else {
                            this.screenReplayIndex = null;
                        }
                    } else if (this.screenReplayIndex < existingCount) {
                        let matchedIdx = -1;
                        const maxLookahead = Math.min(existingCount, this.screenReplayIndex + 3);
                        for (let k = this.screenReplayIndex; k < maxLookahead; k++) {
                            const normExisting = this.normalizeLineForComparison(children[k].textContent || '');
                            if (normExisting === normCurr) {
                                matchedIdx = k;
                                break;
                            }
                        }
                        if (matchedIdx !== -1) {
                            this.currentLineEl.remove();
                            this.currentLineEl = null;
                            this.pendingText = '';
                            this.screenReplayIndex = matchedIdx + 1;
                            return;
                        } else {
                            this.screenReplayIndex = null;
                        }
                    } else {
                        this.screenReplayIndex = null;
                    }
                }

                if (prev) {
                    const normPrev = this.normalizeLineForComparison(prev.textContent);

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
        this.ansiRemainder = '';
        this.activeClasses = [];
        this.isClearingScreen = false;
        if (this.clearScreenTimer) {
            clearTimeout(this.clearScreenTimer);
            this.clearScreenTimer = null;
        }
        this.screenReplayIndex = null;
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

    handleDockerVmChange(vmName) {
        this.selectedDockerVm = (vmName || '').trim();
        if (this.containersVmFilter && this.containersVmFilter.value !== this.selectedDockerVm) {
            this.containersVmFilter.value = this.selectedDockerVm;
        }
        if (this.imagesVmFilter && this.imagesVmFilter.value !== this.selectedDockerVm) {
            this.imagesVmFilter.value = this.selectedDockerVm;
        }
        if (this.volumesVmFilter && this.volumesVmFilter.value !== this.selectedDockerVm) {
            this.volumesVmFilter.value = this.selectedDockerVm;
        }
        if (this.networksVmFilter && this.networksVmFilter.value !== this.selectedDockerVm) {
            this.networksVmFilter.value = this.selectedDockerVm;
        }

        if (this.activeDockerSubnav === 'containers') {
            this.loadContainers(this.selectedDockerVm);
        } else if (this.activeDockerSubnav === 'images') {
            if (this.activeImagesSubtab === 'exports') {
                this.loadExportedTars(this.selectedDockerVm);
            } else {
                this.loadImages(this.selectedDockerVm);
            }
        } else if (this.activeDockerSubnav === 'volumes') {
            this.loadVolumes(this.selectedDockerVm);
        } else if (this.activeDockerSubnav === 'networks') {
            this.loadNetworks(this.selectedDockerVm);
        }
    }

    async updateAllDockerVmFilters() {
        try {
            const res = await fetch('/api/vms');
            if (!res.ok) return;
            const vms = await res.json();
            this.vms = vms;

            let opts = '<option value="">Todas las máquinas virtuales</option>';
            if (Array.isArray(vms)) {
                vms.forEach((v) => {
                    const isRunning = (v.state || '').toLowerCase() === 'running';
                    const label = `${v.name} (${isRunning ? 'En ejecución' : 'Apagada'})`;
                    opts += `<option value="${this.escapeAttr(v.name)}">${this.escapeHTML(label)}</option>`;
                });
            }

            let targetVal = this.selectedDockerVm || '';
            if (targetVal && Array.isArray(vms) && !vms.some((v) => v.name === targetVal)) {
                targetVal = '';
            }
            this.selectedDockerVm = targetVal;

            if (this.containersVmFilter) {
                this.containersVmFilter.innerHTML = opts;
                this.containersVmFilter.value = targetVal;
            }
            if (this.imagesVmFilter) {
                this.imagesVmFilter.innerHTML = opts;
                this.imagesVmFilter.value = targetVal;
            }
            if (this.volumesVmFilter) {
                this.volumesVmFilter.innerHTML = opts;
                this.volumesVmFilter.value = targetVal;
            }
            if (this.networksVmFilter) {
                this.networksVmFilter.innerHTML = opts;
                this.networksVmFilter.value = targetVal;
            }

            let wOpts = '<option value="" disabled selected>Selecciona una máquina...</option>';
            if (Array.isArray(vms)) {
                vms.forEach((v) => {
                    wOpts += `<option value="${this.escapeAttr(v.name)}">${this.escapeHTML(v.name)}</option>`;
                });
            }
            if (this.wizVmSelect) this.wizVmSelect.innerHTML = wOpts;
            if (this.buildVmSelect) this.buildVmSelect.innerHTML = wOpts;
            if (this.createVolumeVm) this.createVolumeVm.innerHTML = wOpts;
        } catch (e) {
            console.warn('No se pudieron listar VMs para filtros de Docker:', e);
        }
    }

    async updateContainersVmFilter() {
        return this.updateAllDockerVmFilters();
    }

    async updateImagesVmFilter() {
        return this.updateAllDockerVmFilters();
    }

    async updateVolumesVmFilter() {
        return this.updateAllDockerVmFilters();
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
        return this.updateAllDockerVmFilters();
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
        const customNetworks = (this.networks || []).filter(
            net => !['bridge', 'host', 'none'].includes((net.name || '').toLowerCase().trim())
        );

        if (customNetworks.length === 0) {
            this.networksListEl.innerHTML = `<div class="empty-vm-list">No hay redes Docker personalizadas disponibles.</div>`;
            return;
        }

        let html = '';
        customNetworks.forEach((net) => {
            const shortId = net.id ? net.id.substring(0, 12) : '-';
            const containersCount = typeof net.containers === 'number' ? net.containers : 0;

            html += `
                <div class="vm-card" data-net-name="${this.escapeAttr(net.name)}" data-vm="${this.escapeAttr(net.vmName)}">
                    <div class="vm-card-header">
                        <div class="vm-card-title-group">
                            <span class="vm-name">${this.escapeHTML(net.name)}</span>
                            <span class="vm-source-tag">${this.escapeHTML(net.vmName)}</span>
                        </div>
                        <div class="vm-card-actions">
                            <button class="btn btn-secondary btn-sm" onclick="window.terminalApp.showNetworkDetails('${this.escapeAttr(net.vmName)}', '${this.escapeAttr(net.name)}')">
                                Detalles
                            </button>
                            <button class="btn btn-danger btn-sm" onclick="window.terminalApp.promptDeleteNetwork('${this.escapeAttr(net.vmName)}', '${this.escapeAttr(net.name)}')">
                                Eliminar
                            </button>
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
        if (this.networkAdvancedFields) this.networkAdvancedFields.classList.add('hidden');
        if (this.networkAdvancedArrow) this.networkAdvancedArrow.textContent = '▶';
        if (this.createNetworkSubnet) this.createNetworkSubnet.value = '';
        if (this.createNetworkGateway) this.createNetworkGateway.value = '';
        if (this.createNetworkIpRange) this.createNetworkIpRange.value = '';
        if (this.createNetworkInternal) this.createNetworkInternal.checked = false;
        if (this.createNetworkAttachable) this.createNetworkAttachable.checked = false;

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
        const subnet = this.createNetworkSubnet ? this.createNetworkSubnet.value.trim() : '';
        const gateway = this.createNetworkGateway ? this.createNetworkGateway.value.trim() : '';
        const ipRange = this.createNetworkIpRange ? this.createNetworkIpRange.value.trim() : '';
        const internal = this.createNetworkInternal ? this.createNetworkInternal.checked : false;
        const attachable = this.createNetworkAttachable ? this.createNetworkAttachable.checked : false;

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
                    subnet,
                    gateway,
                    ipRange,
                    internal,
                    attachable,
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

    async populateNetworkSelect(selectEl, vmName, selectNetworkName = '') {
        if (!selectEl) return;
        selectEl.innerHTML = '<option value="">Predeterminada (bridge)</option><option value="" disabled>Cargando redes...</option>';
        if (!vmName) {
            selectEl.innerHTML = '<option value="">Predeterminada (bridge)</option><option value="host">host (Red del host)</option><option value="none">none (Sin red / Aislada)</option>';
            return;
        }

        try {
            const nets = await this.fetchVmNetworks(vmName);
            let opts = '<option value="">Predeterminada (bridge)</option>';

            const customNetworks = [];
            if (Array.isArray(nets)) {
                nets.forEach((n) => {
                    const lower = (n.name || '').toLowerCase().trim();
                    if (lower === 'bridge' || lower === 'host' || lower === 'none') {
                        return;
                    }
                    customNetworks.push(n);
                });
            }

            const isHostSelected = (selectNetworkName || '').toLowerCase() === 'host';
            const isNoneSelected = (selectNetworkName || '').toLowerCase() === 'none';

            opts += `<option value="host" ${isHostSelected ? 'selected' : ''}>host (Red del host)</option>`;
            opts += `<option value="none" ${isNoneSelected ? 'selected' : ''}>none (Sin red / Aislada)</option>`;

            customNetworks.forEach((n) => {
                const isSelected = selectNetworkName && (n.name === selectNetworkName);
                opts += `<option value="${this.escapeAttr(n.name)}" ${isSelected ? 'selected' : ''}>${this.escapeHTML(n.name)} (${this.escapeHTML(n.driver || 'bridge')})</option>`;
            });

            selectEl.innerHTML = opts;
            if (selectNetworkName) {
                selectEl.value = selectNetworkName;
            }
        } catch (e) {
            console.warn('Error poblando selector de redes:', e);
            selectEl.innerHTML = '<option value="">Predeterminada (bridge)</option><option value="host">host (Red del host)</option><option value="none">none (Sin red / Aislada)</option>';
        }
    }

    async addWizNetworkRow(selectedNet = '') {
        if (!this.wizNetworksContainer) return;
        const row = document.createElement('div');
        row.className = 'dynamic-row wiz-net-row';
        row.innerHTML = `
            <div class="vol-source-wrap" style="flex: 1;">
                <select class="wiz-network-select form-select">
                    <option value="">Predeterminada (bridge)</option>
                    <option value="" disabled>Cargando redes...</option>
                </select>
                <button type="button" class="btn-inline-add" title="Crear nueva red en esta VM">+</button>
            </div>
            <button type="button" class="btn-remove-row" title="Eliminar fila de red">&times;</button>
        `;

        const netSelect = row.querySelector('.wiz-network-select');
        const btnInlineAdd = row.querySelector('.btn-inline-add');
        const btnRemove = row.querySelector('.btn-remove-row');

        const currentVm = this.wizVmSelect ? this.wizVmSelect.value.trim() : '';
        this.populateNetworkSelect(netSelect, currentVm, selectedNet);

        btnInlineAdd.addEventListener('click', () => {
            const vm = this.wizVmSelect ? this.wizVmSelect.value.trim() : '';
            if (!vm) {
                this.showWizFeedback('Selecciona una máquina virtual primero.', 'error');
                return;
            }
            this.openCreateNetworkDialog(vm, (newNetName) => {
                this.populateNetworkSelect(netSelect, vm, newNetName);
            });
        });

        btnRemove.addEventListener('click', () => {
            const allRows = this.wizNetworksContainer.querySelectorAll('.wiz-net-row');
            if (allRows.length > 1) {
                row.remove();
            } else {
                netSelect.value = '';
            }
        });

        this.wizNetworksContainer.appendChild(row);
    }

    async updateWizardNetworks(vmName, selectNetworkName = '') {
        if (!this.wizNetworksContainer) return;
        const selects = this.wizNetworksContainer.querySelectorAll('.wiz-network-select');
        if (selects.length === 0) {
            await this.addWizNetworkRow(selectNetworkName);
            return;
        }
        selects.forEach((sel, idx) => {
            const val = (idx === 0 && selectNetworkName) ? selectNetworkName : sel.value;
            this.populateNetworkSelect(sel, vmName, val);
        });
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
                        <div class="container-head-right">
                            <span class="container-state-pill ${stateClass}">${this.escapeHTML(c.state || 'desconocido')}</span>
                            <div class="card-dropdown">
                                <button type="button" class="btn-dots" title="Más opciones" onclick="event.stopPropagation(); window.terminalApp.toggleContainerMenu(this)">
                                    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                                        <circle cx="12" cy="12" r="1.5"></circle>
                                        <circle cx="19" cy="12" r="1.5"></circle>
                                        <circle cx="5" cy="12" r="1.5"></circle>
                                    </svg>
                                </button>
                                <div class="card-dropdown-menu">
                                    <button type="button" class="dropdown-item" onclick="window.terminalApp.openContainerDetails('${this.escapeAttr(c.vmName)}', '${this.escapeAttr(c.id)}')">
                                        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><line x1="12" y1="16" x2="12" y2="12"/><line x1="12" y1="8" x2="12.01" y2="8"/></svg>
                                        Detalles
                                    </button>
                                    <button type="button" class="dropdown-item" onclick="window.terminalApp.openContainerLogs('${this.escapeAttr(c.vmName)}', '${this.escapeAttr(c.id)}', '${this.escapeAttr(c.name || c.id)}')">
                                        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="4 17 10 11 4 5"/><line x1="12" y1="19" x2="20" y2="19"/></svg>
                                        Ver logs
                                    </button>
                                    ${
                                        isRunning
                                            ? `
                                    <button type="button" class="dropdown-item" onclick="window.terminalApp.restartContainer('${this.escapeAttr(c.vmName)}', '${this.escapeAttr(c.id)}', this)">
                                        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="23 4 23 10 17 10"/><path d="M20.49 15a9 9 0 1 1-2.12-9.36L23 10"/></svg>
                                        Reiniciar
                                    </button>
                                    `
                                            : ''
                                    }
                                    <button type="button" class="dropdown-item" onclick="window.terminalApp.openCommitContainerModal('${this.escapeAttr(c.vmName)}', '${this.escapeAttr(c.id)}', '${this.escapeAttr(c.name || c.id)}')">
                                        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="3" y="3" width="18" height="18" rx="2" ry="2"/><line x1="12" y1="8" x2="12" y2="16"/><line x1="8" y1="12" x2="16" y2="12"/></svg>
                                        Crear imagen
                                    </button>
                                    <div class="dropdown-divider"></div>
                                    <button type="button" class="dropdown-item danger" onclick="window.terminalApp.openDeleteContainerModal('${this.escapeAttr(c.vmName)}', '${this.escapeAttr(c.id)}', '${this.escapeAttr(c.name || c.id)}')">
                                        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="3 6 5 6 21 6"/><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/></svg>
                                        Eliminar
                                    </button>
                                </div>
                            </div>
                        </div>
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
                            <button type="button" class="btn btn-secondary btn-sm" title="Detener contenedor" onclick="window.terminalApp.stopContainer('${this.escapeAttr(c.vmName)}', '${this.escapeAttr(c.id)}', this)">
                                Detener
                            </button>
                        `
                                : `
                            <button type="button" class="btn btn-primary btn-sm" title="Iniciar contenedor" onclick="window.terminalApp.startContainer('${this.escapeAttr(c.vmName)}', '${this.escapeAttr(c.id)}', this)">
                                Iniciar
                            </button>
                        `
                        }
                    </div>
                </div>
            `;
            })
            .join('');
    }

    toggleContainerMenu(btn) {
        const menu = btn.nextElementSibling;
        if (!menu) return;
        const isShown = menu.classList.contains('show');
        this.closeCardDropdowns();
        if (!isShown) {
            menu.classList.add('show');
            btn.classList.add('active');
        }
    }

    toggleVmMenu(btn) {
        this.toggleContainerMenu(btn);
    }

    toggleImageMenu(btn) {
        this.toggleContainerMenu(btn);
    }

    closeCardDropdowns() {
        document.querySelectorAll('.card-dropdown-menu.show').forEach((m) => {
            m.classList.remove('show');
            if (m.previousElementSibling) {
                m.previousElementSibling.classList.remove('active');
            }
        });
    }

    connectContainerTerminal(vmName, containerIdOrName, showLogs = false) {
        if (!vmName || !containerIdOrName) return;

        document.querySelectorAll('.card-dropdown-menu.show').forEach((m) => {
            m.classList.remove('show');
            if (m.previousElementSibling) {
                m.previousElementSibling.classList.remove('active');
            }
        });

        this.clearOutput();
        this.appendInfoLine(`[Conectando al contenedor '${containerIdOrName}' (${vmName}) — escribe 'exit' para volver al host...]`);

        this.currentVM = null;
        this.currentContainer = { vm: vmName, container: containerIdOrName };
        this.containerSessionOpened = false;

        if (this.targetUserHostEl) {
            this.targetUserHostEl.textContent = `${vmName}:${containerIdOrName}`;
            this.targetUserHostEl.title = `Conectado al contenedor ${containerIdOrName} en ${vmName}`;
        }
        if (this.targetShellBadgeEl) {
            this.targetShellBadgeEl.textContent = 'docker';
            this.targetShellBadgeEl.className = 'badge-shell';
        }
        const promptChar = document.querySelector('#input-area .prompt-char');
        if (promptChar) {
            promptChar.textContent = '#';
        }
        this.currentPath = '';
        this.updateCurrentPath(`/${containerIdOrName}`);
        this.lastKnownPromptText = `root@${containerIdOrName}:/#`;

        this.toggleSidebar(false);
        setTimeout(() => {
            if (this.inputEl) this.inputEl.focus();
        }, 50);

        const params = {
            vm: vmName,
            container: containerIdOrName,
        };
        if (showLogs) {
            params.logs = '1';
        }
        this.ws.reconnectWith(params);
    }

    async startContainer(vmName, containerId, btnEl) {
        if (btnEl) {
            this.setButtonLoading(btnEl, true, 'Cargando...');
            const card = btnEl.closest('.container-card');
            if (card) {
                card.querySelectorAll('button').forEach((b) => (b.disabled = true));
            }
        }
        try {
            const res = await fetch(`/api/containers/${encodeURIComponent(vmName)}/${encodeURIComponent(containerId)}/start`, {
                method: 'POST',
            });
            if (res.ok) {
                await this.loadContainers(this.containersVmFilter ? this.containersVmFilter.value : '');
            } else {
                const text = await res.text();
                alert(`Error iniciando contenedor: ${this.parseApiError(text)}`);
                if (btnEl) {
                    this.setButtonLoading(btnEl, false);
                    const card = btnEl.closest('.container-card');
                    if (card) {
                        card.querySelectorAll('button').forEach((b) => (b.disabled = false));
                    }
                }
            }
        } catch (e) {
            alert(`Fallo de conexión al iniciar contenedor: ${e.message}`);
            if (btnEl) {
                this.setButtonLoading(btnEl, false);
                const card = btnEl.closest('.container-card');
                if (card) {
                    card.querySelectorAll('button').forEach((b) => (b.disabled = false));
                }
            }
        }
    }

    async stopContainer(vmName, containerId, btnEl) {
        if (btnEl) {
            this.setButtonLoading(btnEl, true, 'Cargando...');
            const card = btnEl.closest('.container-card');
            if (card) {
                card.querySelectorAll('button').forEach((b) => (b.disabled = true));
            }
        }
        try {
            const res = await fetch(`/api/containers/${encodeURIComponent(vmName)}/${encodeURIComponent(containerId)}/stop`, {
                method: 'POST',
            });
            if (res.ok) {
                await this.loadContainers(this.containersVmFilter ? this.containersVmFilter.value : '');
            } else {
                const text = await res.text();
                alert(`Error deteniendo contenedor: ${this.parseApiError(text)}`);
                if (btnEl) {
                    this.setButtonLoading(btnEl, false);
                    const card = btnEl.closest('.container-card');
                    if (card) {
                        card.querySelectorAll('button').forEach((b) => (b.disabled = false));
                    }
                }
            }
        } catch (e) {
            alert(`Fallo de conexión al detener contenedor: ${e.message}`);
            if (btnEl) {
                this.setButtonLoading(btnEl, false);
                const card = btnEl.closest('.container-card');
                if (card) {
                    card.querySelectorAll('button').forEach((b) => (b.disabled = false));
                }
            }
        }
    }

    async restartContainer(vmName, containerId, btnEl) {
        if (btnEl) {
            this.setButtonLoading(btnEl, true, 'Cargando...');
            const card = btnEl.closest('.container-card');
            if (card) {
                card.querySelectorAll('button').forEach((b) => (b.disabled = true));
                const pill = card.querySelector('.container-state-pill');
                if (pill) {
                    pill.innerHTML = `<span class="btn-spinner" aria-hidden="true" style="width:10px; height:10px; margin-right:4px; vertical-align:-1px;"></span>Cargando...`;
                }
            }
        }
        try {
            const res = await fetch(`/api/containers/${encodeURIComponent(vmName)}/${encodeURIComponent(containerId)}/restart`, {
                method: 'POST',
            });
            if (res.ok) {
                this.closeCardDropdowns();
                await this.loadContainers(this.containersVmFilter ? this.containersVmFilter.value : '');
            } else {
                const text = await res.text();
                alert(`Error reiniciando contenedor: ${this.parseApiError(text)}`);
                if (btnEl) {
                    this.setButtonLoading(btnEl, false);
                    const card = btnEl.closest('.container-card');
                    if (card) {
                        card.querySelectorAll('button').forEach((b) => (b.disabled = false));
                    }
                }
            }
        } catch (e) {
            alert(`Fallo de conexión al reiniciar contenedor: ${e.message}`);
            if (btnEl) {
                this.setButtonLoading(btnEl, false);
                const card = btnEl.closest('.container-card');
                if (card) {
                    card.querySelectorAll('button').forEach((b) => (b.disabled = false));
                }
            }
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
        this.connectContainerTerminal(vmName, containerName || containerId, true);
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

        let networksHtml = '';
        if (info.networks && info.networks.length > 0) {
            networksHtml = `
                <div class="vm-detail-item" style="grid-column: 1 / -1;">
                    <span class="vm-detail-key">Redes Docker Conectadas</span>
                    <table class="container-networks-table">
                        <thead>
                            <tr>
                                <th>Red</th>
                                <th>Dirección IP</th>
                                <th>Gateway</th>
                                <th>Dirección MAC</th>
                            </tr>
                        </thead>
                        <tbody>
                            ${info.networks.map((n) => `
                                <tr>
                                    <td><span class="container-net-badge">${this.escapeHTML(n.name || 'default')}</span></td>
                                    <td>${this.escapeHTML(n.ipAddress || 'Sin IP')}</td>
                                    <td>${this.escapeHTML(n.gateway || '-')}</td>
                                    <td style="color:#8da2bd; font-size:11px;">${this.escapeHTML(n.macAddress || '-')}</td>
                                </tr>
                            `).join('')}
                        </tbody>
                    </table>
                </div>
            `;
        } else if (info.network || info.ipAddress) {
            networksHtml = `
                <div class="vm-detail-item">
                    <span class="vm-detail-key">Red Docker</span>
                    <span class="vm-detail-val">${this.escapeHTML(info.network || 'bridge')}</span>
                </div>
                <div class="vm-detail-item">
                    <span class="vm-detail-key">Dirección IP</span>
                    <span class="vm-detail-val">${this.escapeHTML(info.ipAddress || 'Sin IP')}</span>
                </div>
            `;
        } else {
            networksHtml = `
                <div class="vm-detail-item" style="grid-column: 1 / -1;">
                    <span class="vm-detail-key">Redes Docker</span>
                    <span class="vm-detail-val" style="color:var(--text-muted)">Ninguna red conectada</span>
                </div>
            `;
        }

        this.containerDetailsContent.innerHTML = `
            <div class="vm-detail-item">
                <span class="vm-detail-key">Nombre</span>
                <span class="vm-detail-val highlight">${this.escapeHTML(info.name || '')}</span>
            </div>
            <div class="vm-detail-item">
                <span class="vm-detail-key">ID Contenedor</span>
                <span class="vm-detail-val" style="font-family:var(--font-mono); font-size:12px;">${this.escapeHTML(info.id || '')}</span>
            </div>
            <div class="vm-detail-item">
                <span class="vm-detail-key">Máquina Virtual</span>
                <span class="vm-detail-val">${this.escapeHTML(info.vmName || '')}</span>
            </div>
            <div class="vm-detail-item">
                <span class="vm-detail-key">Imagen</span>
                <span class="vm-detail-val">${this.escapeHTML(info.image || '')}</span>
            </div>
            <div class="vm-detail-item">
                <span class="vm-detail-key">Estado</span>
                <span class="vm-detail-val">${this.escapeHTML(info.state ? info.state.status : '')}</span>
            </div>
            <div class="vm-detail-item">
                <span class="vm-detail-key">Política de Reinicio</span>
                <span class="vm-detail-val">${this.escapeHTML(info.restartPolicy || 'no')}</span>
            </div>
            <div class="vm-detail-item" style="grid-column: 1 / -1;">
                <span class="vm-detail-key">Puertos Mapeados</span>
                <span class="vm-detail-val">${this.escapeHTML(portsStr)}</span>
            </div>
            ${networksHtml}
            <div class="vm-detail-item" style="grid-column: 1 / -1;">
                <span class="vm-detail-key">Montajes de Volúmenes</span>
                <span class="vm-detail-val" style="font-size:11px;">${mountsStr}</span>
            </div>
            <div class="vm-detail-item" style="grid-column: 1 / -1;">
                <span class="vm-detail-key">Variables de Entorno</span>
                <span class="vm-detail-val" style="font-size:11px; max-height:120px; overflow-y:auto; display:block; font-family:var(--font-mono);">${envStr}</span>
            </div>
        `;
    }

    async ensureDockerHubUsernameLoaded() {
        if (this.dockerHubUsername) return this.dockerHubUsername;
        try {
            const res = await fetch('/api/dockerhub/credentials');
            if (res.ok) {
                const creds = await res.json();
                if (creds && creds.username) {
                    this.dockerHubUsername = creds.username.trim();
                }
            }
        } catch (_) {}
        return this.dockerHubUsername;
    }

    isDockerHubImage(repo) {
        if (!repo || repo === '<none>') return false;
        const lower = repo.toLowerCase().trim();
        if (lower.startsWith('docker.io/') || lower.startsWith('index.docker.io/')) {
            return true;
        }
        if (this.dockerHubUsername && lower.startsWith((this.dockerHubUsername + '/').toLowerCase())) {
            return true;
        }
        const slashParts = lower.split('/');
        if (slashParts.length === 2 && !slashParts[0].includes('.') && !slashParts[0].includes(':')) {
            return true;
        }
        return false;
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
            await this.ensureDockerHubUsernameLoaded();
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
        const visibleImages = Array.isArray(images) ? images.filter((img) => !this.isDockerHubImage(img.repository)) : [];
        if (visibleImages.length === 0) {
            this.imagesListEl.innerHTML = `
                <div class="empty-vm-list">
                    <p>No hay imágenes Docker disponibles.</p>
                    <span class="empty-subtext">Crea una imagen con "+ Crear imagen" o descarga una plantilla de contenedor.</span>
                </div>
            `;
            return;
        }

        this.imagesListEl.innerHTML = visibleImages
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
                            <div class="image-head-right">
                                <div class="image-size-badge">${this.escapeHTML(size)}</div>
                                <div class="card-dropdown">
                                    <button type="button" class="btn-dots" title="Más opciones" onclick="event.stopPropagation(); window.terminalApp.toggleImageMenu(this)">
                                        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                                            <circle cx="12" cy="12" r="1.5"></circle>
                                            <circle cx="19" cy="12" r="1.5"></circle>
                                            <circle cx="5" cy="12" r="1.5"></circle>
                                        </svg>
                                    </button>
                                    <div class="card-dropdown-menu">
                                        <button type="button" class="dropdown-item" onclick="window.terminalApp.openPublishDockerHubFlow('${this.escapeAttr(vmName)}', '${this.escapeAttr(img.id)}', '${this.escapeAttr(repo)}', '${this.escapeAttr(tag)}')">
                                            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                                                <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"></path>
                                                <polyline points="17 8 12 3 7 8"></polyline>
                                                <line x1="12" y1="3" x2="12" y2="15"></line>
                                            </svg>
                                            <span>Subir a Docker Hub</span>
                                        </button>
                                        <button type="button" class="dropdown-item" onclick="window.terminalApp.exportImageToTar('${this.escapeAttr(vmName)}', '${this.escapeAttr(img.id)}', '${this.escapeAttr(repo)}', '${this.escapeAttr(tag)}', this)">
                                            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                                                <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"></path>
                                                <polyline points="7 10 12 15 17 10"></polyline>
                                                <line x1="12" y1="15" x2="12" y2="3"></line>
                                            </svg>
                                            <span class="btn-text">Exportar TAR</span>
                                        </button>
                                    </div>
                                </div>
                            </div>
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

    switchImagesSubtab(tab) {
        this.activeImagesSubtab = tab || 'images';
        if (this.imagesSubtabBtns) {
            this.imagesSubtabBtns.forEach((b) => {
                b.classList.toggle('active', b.dataset.imagestab === this.activeImagesSubtab);
            });
        }
        if (this.imagesListEl) {
            this.imagesListEl.classList.toggle('hidden', this.activeImagesSubtab !== 'images');
        }
        if (this.imagesTarListEl) {
            this.imagesTarListEl.classList.toggle('hidden', this.activeImagesSubtab !== 'tars');
        }
        const filter = this.imagesVmFilter ? this.imagesVmFilter.value : '';
        if (this.activeImagesSubtab === 'tars') {
            this.loadExportedTars(filter);
        } else {
            this.loadImages(filter);
        }
    }

    showImagesExportFeedback(msg, type = 'success') {
        if (!this.imagesExportFeedbackEl) return;
        if (this.exportFeedbackTimeout) {
            clearTimeout(this.exportFeedbackTimeout);
            this.exportFeedbackTimeout = null;
        }
        this.imagesExportFeedbackEl.textContent = msg;
        this.imagesExportFeedbackEl.className = `modal-feedback ${type}`;
        this.imagesExportFeedbackEl.classList.remove('hidden');
        this.exportFeedbackTimeout = setTimeout(() => {
            this.hideImagesExportFeedback();
        }, 8000);
    }

    hideImagesExportFeedback() {
        if (this.exportFeedbackTimeout) {
            clearTimeout(this.exportFeedbackTimeout);
            this.exportFeedbackTimeout = null;
        }
        if (!this.imagesExportFeedbackEl) return;
        this.imagesExportFeedbackEl.classList.add('hidden');
        this.imagesExportFeedbackEl.textContent = '';
    }

    formatTarDate(dateStr) {
        if (!dateStr || dateStr === 'n/a') return 'Sin fecha';
        try {
            const isoCandidate = String(dateStr).trim().replace(' ', 'T');
            const parsed = new Date(isoCandidate);
            if (!isNaN(parsed.getTime())) {
                return parsed.toLocaleString('es-ES', {
                    day: '2-digit',
                    month: 'short',
                    year: 'numeric',
                    hour: '2-digit',
                    minute: '2-digit'
                });
            }
        } catch (_) {}
        return dateStr;
    }

    async exportImageToTar(vmName, id, repo, tag, btnEl) {
        this.closeCardDropdowns();
        const ref = (repo && repo !== '<none>')
            ? `${repo}:${(tag && tag !== '<none>') ? tag : 'latest'}`
            : id;

        if (!vmName || !ref) {
            this.showImagesExportFeedback('Faltan datos de la máquina virtual o de la imagen para exportar.', 'error');
            return;
        }

        this.hideImagesExportFeedback();
        if (btnEl) {
            this.setButtonLoading(btnEl, true);
        }

        try {
            const res = await fetch('/api/images/export', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    vmName,
                    image: ref
                })
            });

            const text = await res.text();
            let data = {};
            try {
                data = JSON.parse(text);
            } catch (_) {
                data = { error: text };
            }

            if (!res.ok || !data.success) {
                throw new Error(data.error || this.parseApiError(text) || `Error ${res.status}`);
            }

            this.showImagesExportFeedback(
                `Imagen '${ref}' exportada exitosamente: ${data.fileName} (${data.size}) en ${data.filePath}`,
                'success'
            );
            this.appendInfoLine(`[Docker TAR] Imagen '${ref}' exportada en VM '${vmName}' -> ${data.filePath} (${data.size})`);

            if (this.activeImagesSubtab === 'tars') {
                this.loadExportedTars(this.imagesVmFilter ? this.imagesVmFilter.value : '');
            }
        } catch (e) {
            console.error('Error exportando imagen a TAR:', e);
            this.showImagesExportFeedback(`Error al exportar '${ref}': ${e.message}`, 'error');
        } finally {
            if (btnEl) {
                this.setButtonLoading(btnEl, false);
            }
        }
    }

    async loadExportedTars(vmFilter) {
        if (!this.imagesTarListEl) return;
        this.imagesTarListEl.innerHTML = `
            <div class="empty-vm-list">
                <span class="btn-spinner"></span>
                <span>Consultando archivos TAR en la VM...</span>
            </div>
        `;

        const filter = (vmFilter || '').trim();
        const url = filter ? `/api/images/exports?vm=${encodeURIComponent(filter)}` : '/api/images/exports';

        try {
            const res = await fetch(url);
            const text = await res.text();
            if (!res.ok) {
                let errMsg = text;
                try {
                    const parsed = JSON.parse(text);
                    if (parsed && parsed.error) errMsg = parsed.error;
                } catch (_) {}
                throw new Error(errMsg || `Error ${res.status}`);
            }
            const tars = JSON.parse(text);
            this.exportedTars = Array.isArray(tars) ? tars : [];
            this.renderExportedTars(this.exportedTars);
        } catch (e) {
            console.error('Error cargando archivos TAR:', e);
            this.imagesTarListEl.innerHTML = `
                <div class="empty-vm-list" style="color:var(--accent-red)">
                    <p>No se pudieron consultar los archivos TAR en la VM.</p>
                    <span class="empty-subtext">${this.escapeHTML(e.message)}</span>
                </div>
            `;
        }
    }

    renderExportedTars(tars) {
        if (!this.imagesTarListEl) return;
        if (!Array.isArray(tars) || tars.length === 0) {
            this.imagesTarListEl.innerHTML = `
                <div class="empty-vm-list">
                    <p>No hay archivos TAR exportados en ~/docker-image-exports/.</p>
                    <span class="empty-subtext">Usa el botón "Exportar TAR" en cualquier imagen disponible para generar un archivo .tar dentro de la VM.</span>
                </div>
            `;
            return;
        }

        this.imagesTarListEl.innerHTML = tars
            .map((tar) => {
                const fileName = tar.fileName || 'archivo.tar';
                const filePath = tar.filePath || '~/docker-image-exports/' + fileName;
                const size = tar.size || '0 B';
                const formattedDate = this.formatTarDate(tar.modifiedAt);
                const status = tar.status || 'Verificado';
                const vmName = tar.vmName || '';

                return `
                    <div class="tar-card" data-tar="${this.escapeAttr(fileName)}">
                        <div class="tar-card-head">
                            <div class="tar-card-icon">
                                <svg viewBox="0 0 24 24" width="16" height="16" stroke="currentColor" stroke-width="2" fill="none">
                                    <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"></path>
                                    <polyline points="7 10 12 15 17 10"></polyline>
                                    <line x1="12" y1="15" x2="12" y2="3"></line>
                                </svg>
                            </div>
                            <div class="tar-card-title-col">
                                <div class="tar-card-filename">${this.escapeHTML(fileName)}</div>
                                <div class="tar-card-badges">
                                    ${vmName ? `<span class="vm-source-tag">${this.escapeHTML(vmName)}</span>` : ''}
                                    <span class="tar-status-badge">${this.escapeHTML(status)}</span>
                                    <span class="image-size-badge" title="${Number(tar.sizeBytes || 0).toLocaleString()} bytes">${this.escapeHTML(size)}</span>
                                </div>
                            </div>
                        </div>

                        <div class="tar-card-details">
                            <div class="tar-detail-row-inline">
                                <span class="image-info-label">Fecha de exportación</span>
                                <span class="image-info-value">${this.escapeHTML(formattedDate)}</span>
                            </div>
                            <div class="tar-detail-path-block">
                                <span class="image-info-label">Ruta en la VM</span>
                                <div class="tar-path-value">${this.escapeHTML(filePath)}</div>
                            </div>
                        </div>

                        <div class="tar-card-actions">
                            <button class="btn btn-primary btn-sm" onclick="window.terminalApp.downloadExportedTar('${this.escapeAttr(vmName)}', '${this.escapeAttr(fileName)}')" title="Descargar archivo TAR al equipo local">
                                <svg viewBox="0 0 24 24" width="13" height="13" stroke="currentColor" stroke-width="2" fill="none">
                                    <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"></path>
                                    <polyline points="7 10 12 15 17 10"></polyline>
                                    <line x1="12" y1="15" x2="12" y2="3"></line>
                                </svg>
                                <span>Descargar</span>
                            </button>
                            <button class="btn btn-danger btn-sm" onclick="window.terminalApp.deleteExportedTar('${this.escapeAttr(vmName)}', '${this.escapeAttr(fileName)}', this)" title="Eliminar archivo TAR de la VM">
                                <svg viewBox="0 0 24 24" width="13" height="13" stroke="currentColor" stroke-width="2" fill="none">
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

    downloadExportedTar(vmName, fileName) {
        if (!vmName || !fileName) {
            this.showImagesExportFeedback('Faltan datos de la VM o del archivo TAR para iniciar la descarga.', 'error');
            return;
        }
        const url = `/api/images/exports/download?vm=${encodeURIComponent(vmName)}&file=${encodeURIComponent(fileName)}`;
        const a = document.createElement('a');
        a.href = url;
        a.download = fileName;
        document.body.appendChild(a);
        a.click();
        document.body.removeChild(a);
        this.showImagesExportFeedback(`Iniciando descarga de '${fileName}' desde la VM '${vmName}'...`, 'success');
    }

    async deleteExportedTar(vmName, fileName, btnEl) {
        if (!vmName || !fileName) {
            this.showImagesExportFeedback('Faltan datos de la VM o del archivo TAR para eliminar.', 'error');
            return;
        }

        if (btnEl) {
            this.setButtonLoading(btnEl, true);
        }

        try {
            const url = `/api/images/exports?vm=${encodeURIComponent(vmName)}&file=${encodeURIComponent(fileName)}`;
            const res = await fetch(url, { method: 'DELETE' });
            const text = await res.text();
            let data = {};
            try {
                data = JSON.parse(text);
            } catch (_) {
                data = { error: text };
            }

            if (!res.ok || !data.success) {
                throw new Error(data.error || this.parseApiError(text) || `Error ${res.status}`);
            }

            this.showImagesExportFeedback(`Archivo TAR '${fileName}' eliminado de la VM '${vmName}'.`, 'success');
            this.loadExportedTars(this.imagesVmFilter ? this.imagesVmFilter.value : '');
        } catch (e) {
            console.error('Error eliminando archivo TAR:', e);
            this.showImagesExportFeedback(`No se pudo eliminar '${fileName}': ${e.message}`, 'error');
            if (btnEl) {
                this.setButtonLoading(btnEl, false);
            }
        }
    }

    // =========================================================================
    // Publicación de Imágenes en Docker Hub (Credenciales PAT + Tag + Push)
    // =========================================================================
    async openPublishDockerHubFlow(vmName, id, repo, tag) {
        this.closeCardDropdowns();
        const cleanRepo = (repo && repo !== '<none>') ? repo : '';
        const cleanTag = (tag && tag !== '<none>') ? tag : 'latest';
        const sourceImage = cleanRepo ? `${cleanRepo}:${cleanTag}` : id;

        if (!vmName || !sourceImage) {
            this.showImagesExportFeedback('Faltan datos de la máquina virtual o de la imagen para publicar en Docker Hub.', 'error');
            return;
        }

        let suggestedRepo = cleanRepo;
        if (suggestedRepo.includes('/')) {
            const parts = suggestedRepo.split('/');
            suggestedRepo = parts[parts.length - 1];
        }
        if (!suggestedRepo) {
            suggestedRepo = 'mi-imagen';
        }

        this.pendingPublishImage = {
            vmName,
            id,
            repo: cleanRepo,
            tag: cleanTag,
            sourceImage,
            suggestedRepo
        };

        try {
            const res = await fetch('/api/dockerhub/credentials');
            if (res.ok) {
                const status = await res.json();
                if (status && status.configured && status.username) {
                    this.dockerHubUsername = status.username;
                    this.openDockerHubPublishModal(this.pendingPublishImage, status.username);
                    return;
                }
            }
        } catch (e) {
            console.warn('Error consultando estado de credenciales de Docker Hub:', e);
        }

        this.openDockerHubCredsModal(true);
    }

    async openDockerHubCredsModal(continueToPublish = false) {
        if (!this.dockerHubCredsModal) return;
        this.continueToPublishAfterCreds = Boolean(continueToPublish);
        this.hideDockerHubCredsFeedback();

        if (this.dockerHubTokenInput) {
            this.dockerHubTokenInput.value = '';
        }

        try {
            const res = await fetch('/api/dockerhub/credentials');
            if (res.ok) {
                const status = await res.json();
                if (status && status.username && this.dockerHubUsernameInput) {
                    this.dockerHubUsernameInput.value = status.username;
                    this.dockerHubUsername = status.username;
                }
            }
        } catch (_) {}

        this.dockerHubCredsModal.classList.remove('hidden');
        if (this.dockerHubUsernameInput && !this.dockerHubUsernameInput.value) {
            this.dockerHubUsernameInput.focus();
        } else if (this.dockerHubTokenInput) {
            this.dockerHubTokenInput.focus();
        }
    }

    closeDockerHubCredsModal() {
        if (this.dockerHubCredsModal) {
            this.dockerHubCredsModal.classList.add('hidden');
        }
        if (this.dockerHubTokenInput) {
            this.dockerHubTokenInput.value = '';
        }
        this.hideDockerHubCredsFeedback();
    }

    showDockerHubCredsFeedback(msg, type = 'error') {
        if (!this.dockerHubCredsFeedback) return;
        this.dockerHubCredsFeedback.textContent = msg;
        this.dockerHubCredsFeedback.className = `modal-feedback ${type}`;
        this.dockerHubCredsFeedback.classList.remove('hidden');
    }

    hideDockerHubCredsFeedback() {
        if (!this.dockerHubCredsFeedback) return;
        this.dockerHubCredsFeedback.classList.add('hidden');
        this.dockerHubCredsFeedback.textContent = '';
    }

    async saveDockerHubCredentials() {
        const username = this.dockerHubUsernameInput ? this.dockerHubUsernameInput.value.trim() : '';
        const token = this.dockerHubTokenInput ? this.dockerHubTokenInput.value.trim() : '';

        if (!username) {
            this.showDockerHubCredsFeedback('Debes ingresar tu usuario (Docker ID) de Docker Hub.', 'error');
            return;
        }
        if (!token) {
            this.showDockerHubCredsFeedback('Debes ingresar tu Personal Access Token (PAT) de Docker Hub.', 'error');
            return;
        }

        this.setButtonLoading(this.saveDockerHubCredsBtn, true);
        this.hideDockerHubCredsFeedback();

        try {
            const res = await fetch('/api/dockerhub/credentials', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    username,
                    access_token: token
                })
            });

            const text = await res.text();
            let data = {};
            try {
                data = JSON.parse(text);
            } catch (_) {
                data = { error: text };
            }

            if (!res.ok) {
                throw new Error(data.error || this.parseApiError(text) || `Error ${res.status}`);
            }

            this.dockerHubUsername = data.username || username;
            if (this.dockerHubTokenInput) {
                this.dockerHubTokenInput.value = '';
            }

            const shouldContinue = this.continueToPublishAfterCreds && this.pendingPublishImage;
            this.closeDockerHubCredsModal();

            if (shouldContinue) {
                this.openDockerHubPublishModal(this.pendingPublishImage, this.dockerHubUsername);
            } else {
                this.showImagesExportFeedback(`Credenciales de Docker Hub configuradas para '${this.dockerHubUsername}'.`, 'success');
            }
        } catch (e) {
            this.showDockerHubCredsFeedback(e.message, 'error');
        } finally {
            this.setButtonLoading(this.saveDockerHubCredsBtn, false);
        }
    }

    openDockerHubPublishModal(imageCtx, username) {
        if (!this.dockerHubPublishModal || !imageCtx) return;
        this.pendingPublishImage = imageCtx;
        this.dockerHubUsername = username || this.dockerHubUsername || '';

        this.hideDockerHubPublishFeedback();
        if (this.dockerHubPublishLogContainer) {
            this.dockerHubPublishLogContainer.classList.add('hidden');
        }
        if (this.dockerHubPublishLogOutput) {
            this.dockerHubPublishLogOutput.textContent = '';
        }

        if (this.publishVmDisplay) {
            this.publishVmDisplay.textContent = imageCtx.vmName || '-';
        }
        if (this.publishSourceImageDisplay) {
            this.publishSourceImageDisplay.textContent = imageCtx.sourceImage || '-';
        }
        if (this.publishUsernameDisplay) {
            this.publishUsernameDisplay.textContent = this.dockerHubUsername || '-';
        }
        if (this.publishRepoInput) {
            this.publishRepoInput.value = imageCtx.suggestedRepo || 'mi-imagen';
        }
        if (this.publishTagInput) {
            this.publishTagInput.value = imageCtx.tag || 'latest';
        }

        this.updateDockerHubTargetPreview();
        this.dockerHubPublishModal.classList.remove('hidden');
        if (this.publishRepoInput) {
            this.publishRepoInput.focus();
        }
    }

    closeDockerHubPublishModal(keepPending = false) {
        if (this.dockerHubPublishModal) {
            this.dockerHubPublishModal.classList.add('hidden');
        }
        this.hideDockerHubPublishFeedback();
        if (this.dockerHubPublishLogContainer) {
            this.dockerHubPublishLogContainer.classList.add('hidden');
        }
        if (!keepPending) {
            this.pendingPublishImage = null;
        }
    }

    updateDockerHubTargetPreview() {
        if (!this.publishTargetPreview) return;
        const user = this.dockerHubUsername || 'usuario';
        let repo = this.publishRepoInput ? this.publishRepoInput.value.trim() : '';
        if (repo.toLowerCase().startsWith((user + '/').toLowerCase())) {
            repo = repo.substring(user.length + 1).trim();
        }
        if (!repo) repo = 'repositorio';
        const tag = (this.publishTagInput ? this.publishTagInput.value.trim() : '') || 'latest';
        this.publishTargetPreview.textContent = `${user}/${repo}:${tag}`;
    }

    showDockerHubPublishFeedback(msg, type = 'error') {
        if (!this.dockerHubPublishFeedback) return;
        this.dockerHubPublishFeedback.textContent = msg;
        this.dockerHubPublishFeedback.className = `modal-feedback ${type}`;
        this.dockerHubPublishFeedback.classList.remove('hidden');
    }

    hideDockerHubPublishFeedback() {
        if (!this.dockerHubPublishFeedback) return;
        this.dockerHubPublishFeedback.classList.add('hidden');
        this.dockerHubPublishFeedback.textContent = '';
    }

    async submitDockerHubPublish() {
        if (!this.pendingPublishImage) return;
        const { vmName, sourceImage } = this.pendingPublishImage;
        const repository = this.publishRepoInput ? this.publishRepoInput.value.trim() : '';
        const tag = (this.publishTagInput ? this.publishTagInput.value.trim() : '') || 'latest';

        if (!repository) {
            this.showDockerHubPublishFeedback('Debes indicar el nombre del repositorio de destino.', 'error');
            return;
        }

        this.setButtonLoading(this.submitDockerHubPublishBtn, true);
        this.hideDockerHubPublishFeedback();
        if (this.dockerHubPublishLogContainer) {
            this.dockerHubPublishLogContainer.classList.add('hidden');
        }

        try {
            const res = await fetch('/api/images/push', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    vmName,
                    sourceImage,
                    repository,
                    tag
                })
            });

            const text = await res.text();
            let data = {};
            try {
                data = JSON.parse(text);
            } catch (_) {
                data = { error: text };
            }

            if (!res.ok || !data.success) {
                if (data.output && this.dockerHubPublishLogOutput && this.dockerHubPublishLogContainer) {
                    this.dockerHubPublishLogOutput.textContent = data.output;
                    this.dockerHubPublishLogContainer.classList.remove('hidden');
                }
                const errMsg = data.error || this.parseApiError(text) || `Error ${res.status}`;
                this.showDockerHubPublishFeedback(errMsg, 'error');
                return;
            }

            const targetRef = data.targetImage || `${this.dockerHubUsername}/${repository}:${tag}`;
            this.closeDockerHubPublishModal();
            this.showImagesExportFeedback(
                `Imagen publicada exitosamente en Docker Hub: ${targetRef} (desde VM '${vmName}')`,
                'success'
            );
            this.appendInfoLine(`[Docker Hub] Imagen '${sourceImage}' publicada como '${targetRef}' desde VM '${vmName}'`);
            this.loadImages(this.imagesVmFilter ? this.imagesVmFilter.value : '');
        } catch (e) {
            this.showDockerHubPublishFeedback(`Error de red al publicar la imagen: ${e.message}`, 'error');
        } finally {
            this.setButtonLoading(this.submitDockerHubPublishBtn, false);
        }
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

    setButtonLoading(btn, isLoading, loadingText = 'Cargando...') {
        if (!btn) return;
        btn.disabled = Boolean(isLoading);
        if (isLoading) {
            if (!btn.dataset.originalText) {
                btn.dataset.originalText = btn.innerHTML;
            }
            btn.innerHTML = `<span class="btn-spinner" aria-hidden="true" style="margin-right: 5px; vertical-align: -2px;"></span><span>${this.escapeHTML(loadingText)}</span>`;
        } else {
            if (btn.dataset.originalText) {
                btn.innerHTML = btn.dataset.originalText;
                delete btn.dataset.originalText;
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
        if (this.wizNetAdminCheckbox) this.wizNetAdminCheckbox.checked = false;
        if (this.wizNetworksContainer) {
            this.wizNetworksContainer.innerHTML = '';
            await this.addWizNetworkRow();
        }

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
            const allImages = Array.isArray(images) ? images : [];
            const visibleImages = allImages.filter((img) => !this.isDockerHubImage(img.repository));
            if (visibleImages.length === 0) {
                this.wizImagesGrid.innerHTML = `
                    <div class="empty-vm-list">
                        <p>No se encontraron imágenes en esta VM.</p>
                        <span class="empty-subtext">Crea una con "+ Crear nueva imagen" o selecciona "Aplicaciones" para usar plantillas.</span>
                    </div>
                `;
                return;
            }

            this.wizImagesGrid.innerHTML = visibleImages
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

        // Recopilar redes
        const netSelects = Array.from(document.querySelectorAll('.wiz-net-row .wiz-network-select'));
        const chosenNetworks = [];
        netSelects.forEach((sel) => {
            const val = sel.value.trim();
            if (val && !chosenNetworks.includes(val)) {
                chosenNetworks.push(val);
            }
        });
        const networksSummaryText = chosenNetworks.length > 0
            ? chosenNetworks.map((n) => `<span class="container-net-badge">${this.escapeHTML(n)}</span>`).join(' ')
            : 'Predeterminada (bridge)';

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
                <span class="summary-label">Redes:</span>
                <span class="summary-val">${networksSummaryText}</span>
            </div>
            <div class="summary-row">
                <span class="summary-label">Capacidad Red:</span>
                <span class="summary-val">${this.wizNetAdminCheckbox && this.wizNetAdminCheckbox.checked ? '<span class="container-net-badge">NET_ADMIN (--cap-add NET_ADMIN)</span>' : 'Estándar'}</span>
            </div>
        `;
    }

    async submitCreateContainer() {
        const vmName = this.wizVmSelect ? this.wizVmSelect.value.trim() : '';
        const name = this.wizNameInput ? this.wizNameInput.value.trim() : '';
        const image = this.wizImageInput ? this.wizImageInput.value.trim() : '';
        const restartPolicy = this.wizRestartSelect ? this.wizRestartSelect.value : 'unless-stopped';

        // Recopilar redes (soporte multi-red)
        const netSelects = Array.from(document.querySelectorAll('.wiz-net-row .wiz-network-select'));
        const networks = [];
        netSelects.forEach((sel) => {
            const val = sel.value.trim();
            if (val && !networks.includes(val)) {
                networks.push(val);
            }
        });
        const network = networks.length > 0 ? networks[0] : '';
        const netAdmin = this.wizNetAdminCheckbox ? this.wizNetAdminCheckbox.checked : false;

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
            netAdmin,
            network,
            networks,
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

    // =========================================================================
    // Módulo Docker Swarm Clusters (Listado, Creación SSE 11 Pasos, Detalles y Nodos)
    // =========================================================================

    openVMContainersFromCard(vmName) {
        this.closeCardDropdowns();
        if (!vmName) return;
        this.selectedDockerVm = vmName;
        if (this.containersVmFilter) this.containersVmFilter.value = vmName;
        if (this.imagesVmFilter) this.imagesVmFilter.value = vmName;
        if (this.volumesVmFilter) this.volumesVmFilter.value = vmName;
        if (this.networksVmFilter) this.networksVmFilter.value = vmName;
        this.switchSidebarView('containers');
    }

    async loadClusters() {
        if (!this.clustersListEl) return;
        if (this.clustersListPanel) this.clustersListPanel.classList.remove('hidden');
        if (this.clusterDetailPanel) this.clusterDetailPanel.classList.add('hidden');
        this.activeClusterId = null;

        this.clustersListEl.innerHTML = `<div class="empty-vm-list"><span class="btn-spinner"></span> Sincronizando clusters Docker Swarm...</div>`;
        try {
            const res = await fetch('/api/clusters');
            if (!res.ok) {
                const text = await res.text();
                this.clustersListEl.innerHTML = `<div class="empty-vm-list" style="color:var(--accent-red)">Error al consultar clusters: ${this.parseApiError(text)}</div>`;
                return;
            }
            const data = await res.json();
            this.clusters = Array.isArray(data) ? data : [];
            this.renderClustersList(this.clusters);
        } catch (e) {
            console.warn('Error al cargar clusters:', e);
            this.clustersListEl.innerHTML = `<div class="empty-vm-list" style="color:var(--accent-red)">No se pudo conectar al servidor para listar los clusters.</div>`;
        }
    }

    getClusterStatusBadgeClass(status) {
        const st = (status || '').toLowerCase();
        if (st === 'active') return 'status-active';
        if (st === 'creating') return 'status-creating';
        if (st === 'degraded') return 'status-degraded';
        return 'status-error';
    }

    renderClustersList(clusters) {
        if (!this.clustersListEl) return;
        if (!Array.isArray(clusters) || clusters.length === 0) {
            this.clustersListEl.innerHTML = `
                <div class="empty-vm-list" style="display:flex;flex-direction:column;align-items:center;gap:10px;padding:30px 18px;">
                    <svg viewBox="0 0 24 24" width="32" height="32" stroke="var(--accent)" stroke-width="1.6" fill="none">
                        <circle cx="12" cy="5" r="3"></circle>
                        <circle cx="5" cy="19" r="3"></circle>
                        <circle cx="19" cy="19" r="3"></circle>
                        <line x1="10.5" y1="7.5" x2="6.5" y2="16.5"></line>
                        <line x1="13.5" y1="7.5" x2="17.5" y2="16.5"></line>
                    </svg>
                    <strong style="color:#f8fafc;font-size:14px;">Aún no existen clusters Docker Swarm</strong>
                    <span style="max-width:440px;line-height:1.45;color:var(--text-muted);">
                        Crea tu primer cluster seleccionando una máquina virtual existente como <strong>Manager</strong> y otras VMs como <strong>Workers</strong> conectadas a través de la red privada <code>vessel-net</code>.
                    </span>
                    <button type="button" class="btn btn-primary" style="margin-top:4px;" onclick="window.terminalApp.openCreateClusterModal()">
                        + Crear primer cluster
                    </button>
                </div>
            `;
            return;
        }

        this.clustersListEl.innerHTML = clusters.map((c) => {
            const status = c.status || 'Active';
            const badgeClass = this.getClusterStatusBadgeClass(status);
            const workersConnected = typeof c.workersConnected === 'number' ? c.workersConnected : 0;
            const workersTotal = typeof c.workersTotal === 'number' ? c.workersTotal : 0;
            const totalNodes = typeof c.totalNodes === 'number' ? c.totalNodes : (1 + workersTotal);

            return `
            <div class="cluster-card" onclick="window.terminalApp.openClusterDetails('${this.escapeAttr(c.id)}')">
                <div class="cluster-card-head">
                    <span class="cluster-card-name">${this.escapeHTML(c.name)}</span>
                    <span class="cluster-status-badge ${badgeClass}">${this.escapeHTML(status)}</span>
                </div>
                <div class="cluster-card-summary">
                    <div class="cluster-summary-item">
                        <span class="cluster-summary-label">VM Manager</span>
                        <span class="cluster-summary-value" style="color:var(--accent)">${this.escapeHTML(c.managerVmName || '-')}</span>
                    </div>
                    <div class="cluster-summary-item">
                        <span class="cluster-summary-label">Workers Conectados</span>
                        <span class="cluster-summary-value">${workersConnected} / ${workersTotal}</span>
                    </div>
                    <div class="cluster-summary-item">
                        <span class="cluster-summary-label">Total de Nodos</span>
                        <span class="cluster-summary-value">${totalNodes}</span>
                    </div>
                </div>
                <div class="cluster-card-footer">
                    <span>Red: <code>${this.escapeHTML(c.networkName || 'vessel-net')}</code></span>
                    <span class="cluster-open-link">Ver detalles ›</span>
                </div>
            </div>
            `;
        }).join('');
    }

    async openClusterDetails(clusterId) {
        if (!clusterId) return;
        this.activeClusterId = clusterId;
        if (this.clustersListPanel) this.clustersListPanel.classList.add('hidden');
        if (this.clusterDetailPanel) this.clusterDetailPanel.classList.remove('hidden');

        if (this.clusterManagerCardWrap) {
            this.clusterManagerCardWrap.innerHTML = `<div class="empty-vm-list"><span class="btn-spinner"></span> Consultando estado real de los nodos en Docker Swarm...</div>`;
        }
        if (this.clusterWorkersListEl) {
            this.clusterWorkersListEl.innerHTML = '';
        }

        try {
            const res = await fetch(`/api/clusters/${encodeURIComponent(clusterId)}`);
            if (!res.ok) {
                const text = await res.text();
                const errMsg = this.parseApiError(text);
                if (this.clusterManagerCardWrap) {
                    this.clusterManagerCardWrap.innerHTML = `<div class="empty-vm-list" style="color:var(--accent-red)">Error al cargar el cluster: ${this.escapeHTML(errMsg)}</div>`;
                }
                return;
            }
            const cluster = await res.json();
            this.activeClusterData = cluster;
            this.renderClusterDetailView(cluster);
        } catch (e) {
            console.warn('Error al cargar detalle del cluster:', e);
            if (this.clusterManagerCardWrap) {
                this.clusterManagerCardWrap.innerHTML = `<div class="empty-vm-list" style="color:var(--accent-red)">No se pudo conectar al servidor para consultar el cluster.</div>`;
            }
        }
    }

    backToClustersList() {
        this.activeClusterId = null;
        this.activeClusterData = null;
        if (this.clusterDetailPanel) this.clusterDetailPanel.classList.add('hidden');
        if (this.clustersListPanel) this.clustersListPanel.classList.remove('hidden');
        this.loadClusters();
    }

    renderClusterDetailView(cluster) {
        if (!cluster) return;
        if (this.clusterDetailNameEl) {
            this.clusterDetailNameEl.textContent = cluster.name || '-';
        }
        if (this.clusterDetailStatusPill) {
            const st = cluster.status || 'Active';
            this.clusterDetailStatusPill.textContent = st;
            this.clusterDetailStatusPill.className = `cluster-status-badge ${this.getClusterStatusBadgeClass(st)}`;
        }

        const nodes = Array.isArray(cluster.nodes) ? cluster.nodes : [];
        const managerNode = nodes.find((n) => (n.role || '').toLowerCase() === 'manager') || {
            vmName: cluster.managerVmName,
            internalIp: cluster.managerIp,
            role: 'manager',
            isLeader: true,
            swarmStatus: 'Unknown',
        };
        const workerNodes = nodes.filter((n) => (n.role || '').toLowerCase() === 'worker');

        if (this.clusterDetailSummaryEl) {
            this.clusterDetailSummaryEl.innerHTML = `
                <span>Manager: <strong>${this.escapeHTML(cluster.managerVmName || '-')}</strong></span>
                <span>IP Manager (vessel-net): <strong style="color:var(--accent-green)">${this.escapeHTML(cluster.managerIp || managerNode.internalIp || '-')}</strong></span>
                <span>Workers conectados: <strong>${cluster.workersConnected ?? 0} / ${cluster.workersTotal ?? workerNodes.length}</strong></span>
                <span>Total de nodos: <strong>${cluster.totalNodes ?? nodes.length}</strong></span>
                ${cluster.swarmClusterId ? `<span>Swarm ID: <strong>${this.escapeHTML(cluster.swarmClusterId.slice(0, 14))}...</strong></span>` : ''}
            `;
        }

        if (this.clusterDetailWarningEl) {
            const warnMessages = [];
            if (cluster.statusReason) warnMessages.push(cluster.statusReason);
            if (cluster.singleNodeWarning) warnMessages.push(cluster.singleNodeWarning);
            if (warnMessages.length > 0) {
                this.clusterDetailWarningEl.textContent = warnMessages.join(' · ');
                this.clusterDetailWarningEl.classList.remove('hidden');
            } else {
                this.clusterDetailWarningEl.textContent = '';
                this.clusterDetailWarningEl.classList.add('hidden');
            }
        }

        // Renderizar la tarjeta de la VM Manager reutilizando buildVMCardHTML
        if (this.clusterManagerCardWrap) {
            const mgrVM = {
                ...(managerNode.vm || {}),
                name: managerNode.vmName || (managerNode.vm && managerNode.vm.name) || cluster.managerVmName,
                state: managerNode.vmState || (managerNode.vm && managerNode.vm.state) || 'unknown',
                privateIp: managerNode.internalIp || (managerNode.vm && managerNode.vm.privateIp) || cluster.managerIp,
            };
            this.clusterManagerCardWrap.innerHTML = this.buildVMCardHTML(mgrVM, {
                clusterId: cluster.id,
                clusterName: cluster.name,
                role: 'manager',
                isLeader: managerNode.isLeader !== false,
                swarmStatus: managerNode.swarmStatus || 'Ready',
                availability: managerNode.availability || 'Active',
                nodeId: managerNode.nodeId || '',
            });
        }

        // Renderizar la lista de Workers reutilizando buildVMCardHTML
        if (this.clusterWorkersCountLabel) {
            this.clusterWorkersCountLabel.textContent = `${workerNodes.length} nodo(s) Worker (${cluster.workersConnected ?? 0} conectado(s))`;
        }
        if (this.clusterWorkersListEl) {
            if (workerNodes.length === 0) {
                this.clusterWorkersListEl.innerHTML = `
                    <div class="empty-vm-list">
                        Este cluster no tiene nodos Worker asignados (opera únicamente con el nodo Manager).
                    </div>
                `;
            } else {
                this.clusterWorkersListEl.innerHTML = workerNodes.map((wNode) => {
                    const wrkVM = {
                        ...(wNode.vm || {}),
                        name: wNode.vmName || (wNode.vm && wNode.vm.name),
                        state: wNode.vmState || (wNode.vm && wNode.vm.state) || 'unknown',
                        privateIp: wNode.internalIp || (wNode.vm && wNode.vm.privateIp) || '',
                    };
                    return this.buildVMCardHTML(wrkVM, {
                        clusterId: cluster.id,
                        clusterName: cluster.name,
                        role: 'worker',
                        isLeader: false,
                        swarmStatus: wNode.swarmStatus || 'Unknown',
                        availability: wNode.availability || 'Active',
                        nodeId: wNode.nodeId || '',
                        joinError: wNode.joinError || '',
                    });
                }).join('');
            }
        }
    }

    openCreateClusterModal() {
        if (!this.createClusterModal) return;
        this.isCreatingCluster = false;
        this.selectedClusterManagerVM = '';
        this.selectedClusterWorkerVMs.clear();

        if (this.createClusterForm) this.createClusterForm.reset();
        this.hideCreateClusterFeedback();
        this.hideClusterCreateProgress();
        this.setCreateClusterLoading(false);
        if (this.createClusterSingleNodeWarning) {
            this.createClusterSingleNodeWarning.classList.add('hidden');
        }

        this.createClusterModal.classList.remove('hidden');
        this.createClusterModal.setAttribute('aria-hidden', 'false');
        this.loadEligibleVMsForCluster();
        setTimeout(() => {
            if (this.createClusterNameInput) this.createClusterNameInput.focus();
        }, 50);
    }

    closeCreateClusterModal() {
        if (this.isCreatingCluster) return;
        if (!this.createClusterModal) return;
        this.createClusterModal.classList.add('hidden');
        this.createClusterModal.setAttribute('aria-hidden', 'true');
        this.hideCreateClusterFeedback();
        this.hideClusterCreateProgress();
    }

    async loadEligibleVMsForCluster() {
        if (this.createClusterManagerList) {
            this.createClusterManagerList.innerHTML = `<div style="padding:12px;text-align:center;font-size:12px;color:var(--text-muted)"><span class="btn-spinner"></span> Evaluando disponibilidad, red vessel-net y estado Docker de las VMs...</div>`;
        }
        if (this.createClusterWorkersList) {
            this.createClusterWorkersList.innerHTML = `<div style="padding:12px;text-align:center;font-size:12px;color:var(--text-muted)">Esperando evaluación de VMs...</div>`;
        }

        try {
            const res = await fetch('/api/clusters/eligible-vms');
            if (!res.ok) {
                const text = await res.text();
                const errMsg = this.parseApiError(text);
                if (this.createClusterManagerList) {
                    this.createClusterManagerList.innerHTML = `<div style="padding:12px;color:var(--accent-red);font-size:12px">Error al evaluar VMs: ${this.escapeHTML(errMsg)}</div>`;
                }
                return;
            }
            const items = await res.json();
            this.clusterEligibleVMs = Array.isArray(items) ? items : [];

            // Si la VM seleccionada previamente ya no es elegible, limpiarla
            const mgrValid = this.clusterEligibleVMs.some((v) => v.vmName === this.selectedClusterManagerVM && v.eligible);
            if (!mgrValid) {
                this.selectedClusterManagerVM = '';
            }
            for (const wName of Array.from(this.selectedClusterWorkerVMs)) {
                const wValid = this.clusterEligibleVMs.some((v) => v.vmName === wName && v.eligible) && wName !== this.selectedClusterManagerVM;
                if (!wValid) {
                    this.selectedClusterWorkerVMs.delete(wName);
                }
            }

            this.renderClusterVMSelectors();
        } catch (e) {
            console.warn('Error cargando VMs elegibles para cluster:', e);
            if (this.createClusterManagerList) {
                this.createClusterManagerList.innerHTML = `<div style="padding:12px;color:var(--accent-red);font-size:12px">No se pudo conectar al servidor para evaluar VMs.</div>`;
            }
        }
    }

    renderClusterVMSelectors() {
        const list = this.clusterEligibleVMs || [];

        if (this.createClusterManagerList) {
            if (list.length === 0) {
                this.createClusterManagerList.innerHTML = `<div style="padding:12px;color:var(--text-muted);font-size:12px">No existen máquinas virtuales registradas en Vessel.</div>`;
            } else {
                this.createClusterManagerList.innerHTML = list.map((vm) => {
                    const isSelected = vm.eligible && this.selectedClusterManagerVM === vm.vmName;
                    const disabledClass = vm.eligible ? '' : 'disabled';
                    const selectedClass = isSelected ? 'selected' : '';
                    const ipBadge = vm.privateIp ? `<span class="vm-badge">${this.escapeHTML(vm.privateIp)}</span>` : '';

                    return `
                    <label class="cluster-vm-option ${disabledClass} ${selectedClass}">
                        <div class="cluster-vm-option-left">
                            <input type="radio" name="cluster-manager-vm" value="${this.escapeAttr(vm.vmName)}" ${isSelected ? 'checked' : ''} ${vm.eligible ? '' : 'disabled'} onchange="window.terminalApp.handleSelectClusterManager('${this.escapeAttr(vm.vmName)}')" />
                            <div class="cluster-vm-option-info">
                                <div class="cluster-vm-option-name">
                                    <span>${this.escapeHTML(vm.vmName)}</span>
                                    ${ipBadge}
                                </div>
                                <span class="cluster-vm-option-reason">${this.escapeHTML(vm.reason || '')}</span>
                            </div>
                        </div>
                        <span class="swarm-node-pill ${vm.eligible ? 'ready' : 'down'}">${vm.eligible ? 'Elegible' : 'No disponible'}</span>
                    </label>
                    `;
                }).join('');
            }
        }

        if (this.createClusterWorkersList) {
            const workerCandidates = list.filter((vm) => vm.vmName !== this.selectedClusterManagerVM);
            if (!this.selectedClusterManagerVM) {
                this.createClusterWorkersList.innerHTML = `<div style="padding:12px;color:var(--text-muted);font-size:12px">Selecciona primero la VM Manager para habilitar la selección de Workers.</div>`;
            } else if (workerCandidates.length === 0) {
                this.createClusterWorkersList.innerHTML = `<div style="padding:12px;color:var(--text-muted);font-size:12px">No hay más máquinas virtuales registradas además del Manager seleccionado.</div>`;
            } else {
                this.createClusterWorkersList.innerHTML = workerCandidates.map((vm) => {
                    const isChecked = vm.eligible && this.selectedClusterWorkerVMs.has(vm.vmName);
                    const disabledClass = vm.eligible ? '' : 'disabled';
                    const selectedClass = isChecked ? 'selected' : '';
                    const ipBadge = vm.privateIp ? `<span class="vm-badge">${this.escapeHTML(vm.privateIp)}</span>` : '';

                    return `
                    <label class="cluster-vm-option ${disabledClass} ${selectedClass}">
                        <div class="cluster-vm-option-left">
                            <input type="checkbox" value="${this.escapeAttr(vm.vmName)}" ${isChecked ? 'checked' : ''} ${vm.eligible ? '' : 'disabled'} onchange="window.terminalApp.handleToggleClusterWorker('${this.escapeAttr(vm.vmName)}', this.checked)" />
                            <div class="cluster-vm-option-info">
                                <div class="cluster-vm-option-name">
                                    <span>${this.escapeHTML(vm.vmName)}</span>
                                    ${ipBadge}
                                </div>
                                <span class="cluster-vm-option-reason">${this.escapeHTML(vm.reason || '')}</span>
                            </div>
                        </div>
                        <span class="swarm-node-pill ${vm.eligible ? 'ready' : 'down'}">${vm.eligible ? 'Elegible' : 'No disponible'}</span>
                    </label>
                    `;
                }).join('');
            }
        }

        if (this.createClusterWorkersCounter) {
            const count = this.selectedClusterWorkerVMs.size;
            this.createClusterWorkersCounter.textContent = `${count} seleccionado${count === 1 ? '' : 's'}`;
        }

        this.updateSingleNodeWarning();
    }

    handleSelectClusterManager(vmName) {
        this.selectedClusterManagerVM = vmName;
        // Excluir automáticamente el Manager de la selección de Workers
        if (this.selectedClusterWorkerVMs.has(vmName)) {
            this.selectedClusterWorkerVMs.delete(vmName);
        }
        this.renderClusterVMSelectors();
    }

    handleToggleClusterWorker(vmName, checked) {
        if (!vmName || vmName === this.selectedClusterManagerVM) return;
        if (checked) {
            this.selectedClusterWorkerVMs.add(vmName);
        } else {
            this.selectedClusterWorkerVMs.delete(vmName);
        }
        this.renderClusterVMSelectors();
    }

    updateSingleNodeWarning() {
        if (!this.createClusterSingleNodeWarning) return;
        const showWarning = Boolean(this.selectedClusterManagerVM) && this.selectedClusterWorkerVMs.size === 0;
        this.createClusterSingleNodeWarning.classList.toggle('hidden', !showWarning);
    }

    showCreateClusterFeedback(msg, type = 'error') {
        if (!this.createClusterFeedback) return;
        this.createClusterFeedback.textContent = msg;
        this.createClusterFeedback.className = `modal-feedback ${type}`;
    }

    hideCreateClusterFeedback() {
        if (!this.createClusterFeedback) return;
        this.createClusterFeedback.textContent = '';
        this.createClusterFeedback.className = 'modal-feedback hidden';
    }

    setCreateClusterLoading(loading) {
        if (!this.submitCreateClusterBtn) return;
        const spinner = this.submitCreateClusterBtn.querySelector('.btn-spinner');
        const text = this.submitCreateClusterBtn.querySelector('.btn-text');

        this.submitCreateClusterBtn.disabled = loading;
        if (this.cancelCreateClusterBtn) this.cancelCreateClusterBtn.disabled = loading;
        if (this.closeCreateClusterModalBtn) this.closeCreateClusterModalBtn.disabled = loading;
        if (this.createClusterNameInput) this.createClusterNameInput.disabled = loading;
        if (this.refreshClusterVmsBtn) this.refreshClusterVmsBtn.disabled = loading;

        if (spinner) spinner.classList.toggle('hidden', !loading);
        if (text) text.textContent = loading ? 'Configurando Swarm...' : 'Crear Cluster';
    }

    initClusterCreateProgress() {
        if (!this.clusterCreateProgressWrap) return;
        this.clusterCreateProgressWrap.classList.remove('hidden');

        if (this.clusterProgressStageTitle) {
            this.clusterProgressStageTitle.textContent = 'Inicializando cluster Docker Swarm...';
        }
        if (this.clusterProgressStepBadge) {
            this.clusterProgressStepBadge.textContent = 'Paso 1 de 11';
            this.clusterProgressStepBadge.className = 'vm-progress-badge';
        }
        if (this.clusterProgressBarFill) {
            this.clusterProgressBarFill.style.width = '5%';
            this.clusterProgressBarFill.className = 'vm-progress-bar-fill';
        }
        if (this.clusterProgressMsg) {
            this.clusterProgressMsg.textContent = 'Validando la configuración del cluster...';
        }

        if (this.clusterProgressStepsList) {
            const steps = this.clusterProgressStepsList.querySelectorAll('.vm-pstep');
            steps.forEach((el) => {
                const sNum = parseInt(el.getAttribute('data-cstep'), 10);
                el.classList.remove('active', 'completed', 'failed', 'warning', 'skipped');
                const icon = el.querySelector('.vm-pstep-icon');
                if (sNum === 1) {
                    el.classList.add('active');
                    if (icon) icon.innerHTML = `<span class="vm-spinner-mini"></span>`;
                } else if (icon) {
                    icon.textContent = String(sNum);
                }
            });
        }
    }

    hideClusterCreateProgress() {
        if (!this.clusterCreateProgressWrap) return;
        this.clusterCreateProgressWrap.classList.add('hidden');
    }

    updateClusterCreateProgress(ev) {
        if (!this.clusterCreateProgressWrap || !ev) return;
        this.clusterCreateProgressWrap.classList.remove('hidden');

        const step = ev.step || 1;
        const total = ev.total || 11;
        const status = (ev.status || '').toLowerCase();

        if (ev.stage === 'error' || status === 'failed') {
            if (this.clusterProgressStageTitle) {
                this.clusterProgressStageTitle.textContent = 'Proceso detenido por error';
            }
            if (this.clusterProgressStepBadge) {
                this.clusterProgressStepBadge.textContent = `Fallo en paso ${step}`;
                this.clusterProgressStepBadge.className = 'vm-progress-badge error';
            }
            if (this.clusterProgressBarFill) {
                this.clusterProgressBarFill.className = 'vm-progress-bar-fill error';
            }
            if (this.clusterProgressMsg) {
                this.clusterProgressMsg.textContent = ev.error || ev.message || 'Ocurrió un error al crear el cluster.';
            }
            if (this.clusterProgressStepsList) {
                const stepEl = this.clusterProgressStepsList.querySelector(`.vm-pstep[data-cstep="${step}"]`) ||
                    this.clusterProgressStepsList.querySelector('.vm-pstep.active');
                if (stepEl) {
                    stepEl.classList.remove('active', 'completed');
                    stepEl.classList.add('failed');
                    const icon = stepEl.querySelector('.vm-pstep-icon');
                    if (icon) icon.textContent = '✕';
                }
            }
            return;
        }

        if (ev.stage === 'done') {
            const isDegraded = ev.cluster && ev.cluster.status === 'Degraded';
            if (this.clusterProgressStageTitle) {
                this.clusterProgressStageTitle.textContent = isDegraded
                    ? '¡Cluster creado con advertencias (Degraded)!'
                    : '¡Cluster Docker Swarm creado con éxito!';
            }
            if (this.clusterProgressStepBadge) {
                this.clusterProgressStepBadge.textContent = isDegraded ? 'Parcial (Degraded)' : 'Completado';
                this.clusterProgressStepBadge.className = `vm-progress-badge ${isDegraded ? 'warning' : 'success'}`;
            }
            if (this.clusterProgressBarFill) {
                this.clusterProgressBarFill.style.width = '100%';
                this.clusterProgressBarFill.className = `vm-progress-bar-fill ${isDegraded ? 'warning' : 'success'}`;
            }
            if (this.clusterProgressMsg && ev.message) {
                this.clusterProgressMsg.textContent = ev.message;
            }
            return;
        }

        const pct = Math.min(96, Math.max(8, Math.round((step / total) * 100)));
        if (this.clusterProgressStageTitle) {
            this.clusterProgressStageTitle.textContent = `Configurando Docker Swarm (Paso ${step}/${total})...`;
        }
        if (this.clusterProgressStepBadge) {
            this.clusterProgressStepBadge.textContent = `Paso ${step} de ${total}`;
            this.clusterProgressStepBadge.className = 'vm-progress-badge';
        }
        if (this.clusterProgressBarFill) {
            this.clusterProgressBarFill.style.width = `${pct}%`;
        }
        if (this.clusterProgressMsg && ev.message) {
            this.clusterProgressMsg.textContent = ev.message;
        }

        if (this.clusterProgressStepsList) {
            const stepEl = this.clusterProgressStepsList.querySelector(`.vm-pstep[data-cstep="${step}"]`);
            if (stepEl) {
                const icon = stepEl.querySelector('.vm-pstep-icon');
                stepEl.classList.remove('active', 'completed', 'failed', 'warning', 'skipped');
                if (status === 'in_progress') {
                    stepEl.classList.add('active');
                    if (icon) icon.innerHTML = `<span class="vm-spinner-mini"></span>`;
                } else if (status === 'completed') {
                    stepEl.classList.add('completed');
                    if (icon) icon.textContent = '✓';
                } else if (status === 'warning') {
                    stepEl.classList.add('warning');
                    if (icon) icon.textContent = '!';
                } else if (status === 'skipped') {
                    stepEl.classList.add('skipped');
                    if (icon) icon.textContent = '–';
                }
            }
        }
    }

    async submitCreateCluster() {
        if (this.isCreatingCluster) return;

        const name = this.createClusterNameInput ? this.createClusterNameInput.value.trim() : '';
        const managerVm = this.selectedClusterManagerVM;
        const workerVms = Array.from(this.selectedClusterWorkerVMs);

        if (!name) {
            this.showCreateClusterFeedback('Ingresa un nombre válido para el cluster.', 'error');
            return;
        }
        if (!managerVm) {
            this.showCreateClusterFeedback('Selecciona una máquina virtual elegible para funcionar como Manager.', 'error');
            return;
        }

        this.hideCreateClusterFeedback();
        this.isCreatingCluster = true;
        this.setCreateClusterLoading(true);
        this.initClusterCreateProgress();
        this.appendInfoLine(`[Swarm] Iniciando creación del cluster '${name}' (Manager: ${managerVm}, Workers: ${workerVms.length})...`);

        try {
            const res = await fetch('/api/clusters', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'Accept': 'text/event-stream'
                },
                body: JSON.stringify({ name, managerVm, workerVms })
            });

            const contentType = res.headers.get('Content-Type') || '';
            if (res.ok && contentType.includes('text/event-stream') && res.body) {
                const reader = res.body.getReader();
                const decoder = new TextDecoder('utf-8');
                let buffer = '';
                let finalCluster = null;
                let streamError = null;

                while (true) {
                    const { value, done } = await reader.read();
                    if (done) break;
                    buffer += decoder.decode(value, { stream: true });

                    const parts = buffer.split('\n\n');
                    buffer = parts.pop() || '';

                    for (const block of parts) {
                        const line = block.split('\n').find((l) => l.startsWith('data:'));
                        if (!line) continue;
                        try {
                            const ev = JSON.parse(line.slice(5).trim());
                            this.updateClusterCreateProgress(ev);
                            if (ev.stage === 'error') {
                                streamError = ev.error || ev.message || 'Error al crear el cluster';
                            } else if (ev.stage === 'done' && ev.cluster) {
                                finalCluster = ev.cluster;
                            }
                        } catch (parseErr) {
                            console.warn('Error parseando evento SSE de cluster:', parseErr);
                        }
                    }
                }

                if (streamError) {
                    this.showCreateClusterFeedback(`Error: ${streamError}`, 'error');
                    this.appendErrorLine(`[Swarm] Error al crear cluster '${name}': ${streamError}`);
                    return;
                }

                if (finalCluster) {
                    const isDegraded = finalCluster.status === 'Degraded';
                    this.showCreateClusterFeedback(
                        isDegraded
                            ? `Cluster '${finalCluster.name}' creado en estado Degraded (${finalCluster.statusReason || 'revisa los Workers en la vista de detalles'}).`
                            : `¡Cluster '${finalCluster.name}' creado y verificado con éxito!`,
                        isDegraded ? 'info' : 'success'
                    );
                    this.appendInfoLine(`[Swarm] Cluster '${finalCluster.name}' creado (${finalCluster.status}). Manager: ${finalCluster.managerVmName} (${finalCluster.managerIp}), Nodos: ${finalCluster.totalNodes}.`);

                    setTimeout(async () => {
                        this.isCreatingCluster = false;
                        this.closeCreateClusterModal();
                        await this.openClusterDetails(finalCluster.id);
                    }, 1400);
                }
                return;
            }

            if (!res.ok) {
                const text = await res.text();
                const errMsg = this.parseApiError(text);
                this.updateClusterCreateProgress({ stage: 'error', step: 1, error: errMsg, message: errMsg });
                this.showCreateClusterFeedback(`Error: ${errMsg}`, 'error');
                this.appendErrorLine(`[Swarm] Error al crear cluster '${name}': ${errMsg}`);
                return;
            }

            const created = await res.json();
            this.updateClusterCreateProgress({ stage: 'done', step: 11, total: 11, cluster: created });
            this.appendInfoLine(`[Swarm] Cluster '${created.name}' creado exitosamente.`);
            this.isCreatingCluster = false;
            this.closeCreateClusterModal();
            await this.openClusterDetails(created.id);
        } catch (err) {
            console.warn('Error al crear cluster:', err);
            this.updateClusterCreateProgress({ stage: 'error', step: 1, error: err.message, message: 'No se pudo conectar al servidor.' });
            this.showCreateClusterFeedback('No se pudo conectar al servidor para crear el cluster.', 'error');
        } finally {
            this.isCreatingCluster = false;
            this.setCreateClusterLoading(false);
        }
    }

    async retryClusterWorker(clusterId, vmName, btn) {
        if (!clusterId || !vmName) return;
        const spinner = btn ? btn.querySelector('.btn-spinner') : null;
        const text = btn ? btn.querySelector('.btn-text') : null;
        if (btn) btn.disabled = true;
        if (spinner) spinner.classList.remove('hidden');
        if (text) text.textContent = 'Uniendo...';

        try {
            const res = await fetch(`/api/clusters/${encodeURIComponent(clusterId)}/workers`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ vm: vmName })
            });
            if (!res.ok) {
                const errText = await res.text();
                const errMsg = this.parseApiError(errText);
                this.appendErrorLine(`[Swarm] Error al reintentar unión de Worker '${vmName}': ${errMsg}`);
                alert(`No se pudo incorporar el Worker '${vmName}':\n${errMsg}`);
            } else {
                this.appendInfoLine(`[Swarm] Worker '${vmName}' incorporado exitosamente al cluster.`);
            }
        } catch (e) {
            this.appendErrorLine(`[Swarm] Error de red al reintentar unión de Worker '${vmName}'.`);
        } finally {
            if (btn) btn.disabled = false;
            if (spinner) spinner.classList.add('hidden');
            if (text) text.textContent = 'Reintentar unión';
            await this.openClusterDetails(clusterId);
        }
    }

    openRemoveClusterNodeModal(clusterId, vmName, role, clusterName) {
        this.closeCardDropdowns();
        if (!this.removeClusterNodeModal) return;

        const isMgr = (role || '').toLowerCase() === 'manager';
        const activeWorkersCount = this.activeClusterData && Array.isArray(this.activeClusterData.nodes)
            ? this.activeClusterData.nodes.filter((n) => (n.role || '').toLowerCase() === 'worker').length
            : 0;

        this.pendingRemoveClusterNode = {
            clusterId,
            vmName,
            role: isMgr ? 'manager' : 'worker',
            clusterName: clusterName || (this.activeClusterData && this.activeClusterData.name) || clusterId
        };

        if (this.removeClusterNodeVmNameEl) {
            this.removeClusterNodeVmNameEl.textContent = vmName;
        }
        if (this.removeClusterNodeRoleBadgeEl) {
            this.removeClusterNodeRoleBadgeEl.textContent = isMgr ? 'MANAGER' : 'WORKER';
            this.removeClusterNodeRoleBadgeEl.className = `cluster-role-badge ${isMgr ? 'role-manager' : 'role-worker'}`;
        }
        if (this.removeClusterNodeClusterNameEl) {
            this.removeClusterNodeClusterNameEl.textContent = this.pendingRemoveClusterNode.clusterName;
        }

        if (this.removeClusterNodeConsequenceEl) {
            if (isMgr && activeWorkersCount > 0) {
                this.removeClusterNodeConsequenceEl.innerHTML = `
                    <strong style="color:var(--accent-yellow)">Acción bloqueada:</strong> Este nodo es el único <strong>Manager</strong> y el cluster aún tiene <strong>${activeWorkersCount} Worker(s)</strong> registrados. Debes retirar todos los Workers antes de poder retirar el Manager.
                `;
            } else if (isMgr) {
                this.removeClusterNodeConsequenceEl.innerHTML = `
                    Al retirar el último <strong>Manager</strong>, se ejecutará <code>docker swarm leave --force</code> en la VM y se eliminará el registro del cluster en Vessel. <strong>No se eliminará</strong> la máquina virtual de VirtualBox ni sus contenedores/imágenes locales.
                `;
            } else {
                this.removeClusterNodeConsequenceEl.innerHTML = `
                    Se pondrá el nodo en modo <code>drain</code>, ejecutará <code>docker swarm leave</code> y se eliminará de la lista de nodos del Manager. <strong>No se eliminará</strong> la máquina virtual de VirtualBox, su sistema operativo, Docker Engine ni sus imágenes o volúmenes locales.
                `;
            }
        }

        if (this.removeClusterNodeForceBox) {
            this.removeClusterNodeForceBox.classList.add('hidden');
        }
        if (this.removeClusterNodeForceCheck) {
            this.removeClusterNodeForceCheck.checked = false;
        }
        if (this.removeClusterNodeFeedback) {
            this.removeClusterNodeFeedback.textContent = '';
            this.removeClusterNodeFeedback.className = 'modal-feedback hidden';
        }

        this.setRemoveClusterNodeLoading(false);
        if (this.confirmRemoveClusterNodeBtn) {
            this.confirmRemoveClusterNodeBtn.disabled = isMgr && activeWorkersCount > 0;
        }

        this.removeClusterNodeModal.classList.remove('hidden');
        this.removeClusterNodeModal.setAttribute('aria-hidden', 'false');
    }

    closeRemoveClusterNodeModal() {
        if (!this.removeClusterNodeModal) return;
        this.removeClusterNodeModal.classList.add('hidden');
        this.removeClusterNodeModal.setAttribute('aria-hidden', 'true');
        this.pendingRemoveClusterNode = null;
        this.setRemoveClusterNodeLoading(false);
    }

    setRemoveClusterNodeLoading(loading) {
        if (!this.confirmRemoveClusterNodeBtn) return;
        const spinner = this.confirmRemoveClusterNodeBtn.querySelector('.btn-spinner');
        const text = this.confirmRemoveClusterNodeBtn.querySelector('.btn-text');

        this.confirmRemoveClusterNodeBtn.disabled = loading;
        if (this.cancelRemoveClusterNodeBtn) this.cancelRemoveClusterNodeBtn.disabled = loading;
        if (this.closeRemoveClusterNodeModalBtn) this.closeRemoveClusterNodeModalBtn.disabled = loading;

        if (spinner) spinner.classList.toggle('hidden', !loading);
        if (text) text.textContent = loading ? 'Retirando nodo...' : 'Eliminar del cluster';
    }

    async executeRemoveClusterNode() {
        if (!this.pendingRemoveClusterNode) return;
        const { clusterId, vmName, clusterName } = this.pendingRemoveClusterNode;
        const force = Boolean(this.removeClusterNodeForceCheck && this.removeClusterNodeForceCheck.checked);

        this.setRemoveClusterNodeLoading(true);
        if (this.removeClusterNodeFeedback) {
            this.removeClusterNodeFeedback.textContent = '';
            this.removeClusterNodeFeedback.className = 'modal-feedback hidden';
        }

        try {
            const url = `/api/clusters/${encodeURIComponent(clusterId)}/nodes/${encodeURIComponent(vmName)}${force ? '?force=true' : ''}`;
            const res = await fetch(url, { method: 'DELETE' });

            if (res.status === 409) {
                const data = await res.json().catch(() => ({}));
                if (data && data.workerUnreachable) {
                    if (this.removeClusterNodeForceBox) {
                        this.removeClusterNodeForceBox.classList.remove('hidden');
                    }
                    if (this.removeClusterNodeForceCheck) {
                        this.removeClusterNodeForceCheck.checked = true;
                    }
                    if (this.removeClusterNodeFeedback) {
                        this.removeClusterNodeFeedback.textContent = data.error || 'El nodo está desconectado. Marca la casilla superior y pulsa nuevamente para forzar su retiro.';
                        this.removeClusterNodeFeedback.className = 'modal-feedback error';
                    }
                    this.setRemoveClusterNodeLoading(false);
                    return;
                }
            }

            if (!res.ok) {
                const text = await res.text();
                const errMsg = this.parseApiError(text);
                if (this.removeClusterNodeFeedback) {
                    this.removeClusterNodeFeedback.textContent = `Error: ${errMsg}`;
                    this.removeClusterNodeFeedback.className = 'modal-feedback error';
                }
                this.appendErrorLine(`[Swarm] Error al retirar '${vmName}' del cluster '${clusterName}': ${errMsg}`);
                this.setRemoveClusterNodeLoading(false);
                return;
            }

            const result = await res.json();
            this.appendInfoLine(`[Swarm] ${result.message || `Nodo '${vmName}' retirado del cluster '${clusterName}'.`}`);
            if (result.warning) {
                this.appendInfoLine(`[Swarm Aviso] ${result.warning}`);
            }

            this.closeRemoveClusterNodeModal();
            if (result.clusterDeleted) {
                this.backToClustersList();
            } else {
                await this.openClusterDetails(clusterId);
            }
        } catch (e) {
            console.warn('Error al retirar nodo del cluster:', e);
            if (this.removeClusterNodeFeedback) {
                this.removeClusterNodeFeedback.textContent = 'No se pudo conectar al servidor para retirar el nodo.';
                this.removeClusterNodeFeedback.className = 'modal-feedback error';
            }
            this.setRemoveClusterNodeLoading(false);
        }
    }

    escapeAttr(str) {
        if (!str) return '';
        return String(str).replace(/["'<>]/g, '');
    }
}

document.addEventListener('DOMContentLoaded', () => {
    new TerminalApp();
});
