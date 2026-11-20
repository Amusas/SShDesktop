# Vessel: SSH Web Terminal, Virtual Machine & Docker Manager

Aplicación web interactiva en tiempo real para emulación de terminal SSH, aprovisionamiento automatizado y gestión integral de **Máquinas Virtuales en Oracle VirtualBox 7.2** sobre host remoto Windows, y una **plataforma completa de contenerización y orquestación Docker nativa dentro de las máquinas virtuales Linux**.

El sistema implementa aprovisionamiento instantáneo a partir de **plantillas base con discos diferenciales**, generación criptográfica nativa de claves **Ed25519**, verificación estricta de `known_hosts`, túnel SSH transparente con PTY completa, red privada compartida inter-VM (**vessel-net**), gestión de **Docker Swarm Clusters** multi-nodo, y un sistema de **Docker Registry Privado Local con Pull-through Cache inteligente**, todo operando 100% dentro de las VMs invitadas sin requerir Docker en el host Windows.

---

## Características Principales

### 1. Gestión Integral de Docker en VMs Linux
- **Cero Docker en el Host Windows**: Todos los contenedores, imágenes, volúmenes y redes se construyen y ejecutan exclusivamente dentro del motor Docker de las máquinas virtuales Linux. El host Windows no requiere tener Docker instalado.
- **Túnel SSH y Ejecución Segura**: Las operaciones Docker se canalizan mediante el túnel SSH hacia la máquina virtual con validación estricta contra inyección de comandos, escapado POSIX con comillas simples y ejecución controlada vía `sudo docker`.
- **Navegación Docker Integrada con Submenú**:
  - Al acceder a **Docker** en el menú de navegación, se despliega el submenú con acceso directo a:
    - **Contenedores**: Ciclo de vida, inspección, logs y consola.
    - **Imágenes**: Catálogo local, compilación Dockerfile, commit, exportación tar y push a Docker Hub.
    - **Clusters**: Creación, administración y topología de Docker Swarm.
    - **Registry**: Registry privado local y proxy caché de Docker Hub.
  - **Selector Dinámico por Máquina Virtual**: Selector de VM con estado inicial claro (*"Selecciona MV"*) para dirigir operaciones puntuales a cada nodo.
  - **Indicador de Salud Docker en Vivo**: Muestra en tiempo real si el daemon Docker está activo, su versión y conteo de recursos en la VM seleccionada.

#### Contenedores y Asistente Visual (Wizard)
- **Asistente de Creación Paso a Paso**:
  - **Paso 1 (VM y Origen)**: Selecciona entre imágenes locales existentes o el catálogo de aplicaciones recomendadas.
  - **Paso 2 (Catálogo / Plantilla)**: Categorías visuales (*Bases de Datos, Servidores Web, Mensajería, Caché, Desarrollo o Personalizado*) con puertos y volúmenes preconfigurados.
  - **Paso 3 (Configuración Dinámica)**: Nombre, política de reinicio (`unless-stopped`, `always`, `on-failure`, `no`), mapeo de puertos (`Host -> Contenedor`), variables de entorno (`KEY=VALUE`), montajes de volúmenes persistentes y asignación a redes Docker existentes.
  - **Paso 4 (Resumen)**: Tarjeta de revisión de parámetros antes de la creación.
- **Ciclo de Vida Completo**: Iniciar, detener, reiniciar y eliminar contenedores (con opción `--force`).
- **Visor de Logs en Vivo**: Consulta de líneas recientes (`100`, `200`, `500`, `1000`) con refresco dinámico.
- **Inspección Profunda**: Consulta de ID, IPs internas, redes conectadas, mapeos de puertos, variables de entorno y montajes.
- **Terminal Interactiva Directa (`docker exec`)**: Consola interactiva Bash/Sh dentro del contenedor mediante WebSocket y PTY con soporte ANSI completo.

#### Gestión de Imágenes Docker
- **Listado y Filtro**: Visualización de Repository, Tag, Image ID, Tamaño y fecha de creación.
- **Compilación de Imágenes (`+ Crear imagen`)**: Soporte para subir archivos Dockerfile (dropzone interactivo) o escribir instrucciones en editor integrado.
- **Commit de Contenedores**: Creación de nuevas imágenes a partir del estado de ejecución de un contenedor existente (`docker commit`).
- **Exportación a `.tar` y Descarga Directa**: Exporta imágenes a archivos tar dentro de la VM y permite descargarlas directamente al navegador del usuario.
- **Publicación en Docker Hub**: Gestión segura de credenciales de Docker Hub (`username` y `access_token`) con guardado persistente y subida autenticada (`docker push`).

#### Volúmenes y Redes Docker
- **Volúmenes**: Listado, creación de volúmenes persistentes con drivers configurables, inspección de puntos de montaje y eliminación segura.
- **Redes Docker**: Administración de redes puente (`bridge`), superpuestas (`overlay`) y de host; creación con subredes y gateways personalizados, inspección de contenedores adjuntos y desconexión/conexión dinámica.

---

### 2. Clusters Docker Swarm (Orquestación Multi-Nodo)
- **Orquestación sobre VMs Existentes**: Permite formar clusters Docker Swarm utilizando exclusivamente las máquinas virtuales creadas y administradas en Vessel, sin modificar el flujo de creación de VMs.
- **Asistente de Creación con Progreso en Tiempo Real (SSE)**:
  - Definición del nombre del cluster.
  - Selección guiada de un nodo **Manager / Leader** y múltiples nodos **Workers**.
  - Inicialización automática en el Manager (`docker swarm init --advertise-addr <IP_PRIVADA>`).
  - Emisión de Server-Sent Events (SSE) para reflejar paso a paso el progreso: validación de red, inicialización del Manager, obtención de tokens y unión de cada Worker (`docker swarm join`).
- **Gestión Avanzada del Ciclo de Vida del Cluster**:
  - **Topología en Vivo**: Visualización del Leader, Managers y Workers con sus IPs privadas en `vessel-net`, estado del motor Docker y disponibilidad (`Active`, `Drain`, `Pause`).
  - **Expulsión y Drenado Seguro de Nodos**: Pone al nodo en modo `drain` para reubicar tareas, ejecuta `docker swarm leave` y lo elimina del inventario del Manager (`docker node rm`).
  - **Expulsión Forzada de Nodos Desconectados**: Opción de forzar la eliminación desde el Manager si la VM Worker está apagada o sin conexión SSH.
  - **Reincorporación y Reintento de Workers**: Posibilidad de reintentar la unión de nodos fallidos o incorporar nuevos Workers a un cluster activo.
  - **Desmantelamiento Completo y Limpio**: Destrucción segura del cluster con liberación de recursos y restauración del estado individual de cada nodo.

---

### 3. Docker Registry Privado Local & Pull-through Cache
- **Arquitectura de Doble Instancia Aislada (`registry:2.8.3`)**:
  - **Puerto 5000 (`vessel-registry`) - Registry Local Escribible**:
    - Instancia dedicada exclusivamente al almacenamiento y distribución de imágenes propias de proyectos o compilaciones locales.
    - Soporte completo para operaciones `docker push` y `docker pull`.
    - Persistencia en volumen dedicado `vessel-registry-data`.
    - Explorador visual del catálogo de repositorios (`/v2/_catalog`), etiquetas, digests y tamaño de capas.
    - Eliminación de imágenes con limpieza física de blobs mediante recolección de basura (*Garbage Collection*).
  - **Puerto 5001 (`vessel-registry-cache`) - Pull-through Cache Transparente**:
    - Proxy caché conectado como espejo de Docker Hub (`REGISTRY_PROXY_REMOTEURL: https://registry-1.docker.io`).
    - **Interceptación y Almacenamiento Automático**: Intercepta solicitudes de descarga de Docker Hub, guarda las capas localmente en el volumen `vessel-registry-cache-data` y las distribuye a las demás VMs a velocidad de red local gigabit sin consumir internet.
    - Modo estrictamente desacoplado para no interferir con las imágenes escribibles locales.
    - Purga y limpieza de caché con un solo clic sin afectar el Registry privado local.
- **Sincronización Automática de Nodos (`/etc/docker/daemon.json`)**:
  - Modal inteligente para configurar todas las VMs de Vessel con:
    - `registry-mirrors`: Registro del puerto 5001 como mirror de Docker Hub.
    - `insecure-registries`: Permiso HTTP plano para los puertos 5000 y 5001 (evitando errores de certificados TLS).
  - Opciones de aplicación: **Recarga en caliente (`systemctl reload docker`)** sin detener contenedores activos o **Reinicio completo (`systemctl restart docker`)**.
- **Descarga Inteligente (*Smart Pull*)**:
  - Asistente para descargar imágenes con selector de origen: priorizar Registry local, Pull-through Cache o fallback directo.

---

### 4. Red Privada Inter-VM (`vessel-net`) y Doble Adaptador de Red
- **Red NAT Compartida (NAT Network)**: Configuración en VirtualBox de la red compartida `vessel-net` (`10.250.0.0/24`, Gateway `10.250.0.1`, servidor DHCP integrado).
- **Arquitectura de Doble NIC por VM**:
  - **Adaptador 1 (NAT simple)**: Exclusivo para reenvío de puertos NAT SSH hacia el host Windows (`127.0.0.1:2200X -> 22`), garantizando el túnel de control del backend.
  - **Adaptador 2 (NAT Network `vessel-net`)**: Comunicación IP directa L3/L4 entre todas las máquinas virtuales invitadas sin salir a la red física externa.
- **Descubrimiento y Diagnóstico de Red**:
  - Mapeo automático de direcciones MAC (`MACAddress2`) contra la tabla DHCP y las interfaces del sistema invitado para registrar la IP privada exacta de cada VM.
  - Herramienta de verificación de conectividad bidireccional inter-VM (ICMP Ping con fallback a comprobación TCP en puertos de servicio).

---

### 5. Gestión Integral de Máquinas Virtuales (VirtualBox 7.2)
- **Aprovisionamiento Instantáneo mediante Plantillas Base**: Múltiples máquinas virtuales comparten un único disco base en modo lectura con discos diferenciales.
- **Compatibilidad con VirtualBox 7.2**: Si VirtualBox restringe la asociación directa por estado del medio (`MultiAttach compatibility`), el sistema aprovisiona automáticamente un disco diferencial hijo (`--diffparent`), garantizando que la plantilla original jamás se modifique ni se corrompa.
- **Aprovisionamiento de Identidad Criptográfica (Zero-Touch Provisioning)**:
  - Generación nativa en Go de pares de claves **Ed25519** (OpenSSH / PKCS#8 PEM) exclusivas para cada VM.
  - Derivación de usuario Linux único no colisionable (ej. `vm_servidor_8c40`).
  - Inicialización automática de llaves de host SSH en el primer arranque mediante `VBoxManage guestcontrol` o secuencia en consola virtual TTY2.
  - Conexión inicial mediante credenciales temporales de `bootstrap`, creación del usuario único con privilegios `sudo`, autorización de su clave pública y **revocación inmediata del acceso bootstrap**.
  - Optimización del arranque cambiando el target a `multi-user.target` (modo servidor de alta velocidad).
- **Operaciones de Ciclo de Vida**:
  - Creación con configuración dinámica de CPU cores y memoria RAM.
  - Arranque headless en segundo plano.
  - Apagado ordenado mediante señal ACPI (`acpipowerbutton`).
  - Apagado forzado inmediato (`poweroff`).
  - Eliminación en cascada: destrucción de discos diferenciales, liberación de puertos NAT, eliminación de llaves asociadas y limpieza de `known_hosts`.
- **Sincronización Bidireccional**: Sincronización en vivo con `VBoxManage` y depuración automática de máquinas huérfanas si fueron borradas externamente.

---

### 6. Terminal Web SSH Interactivo
- **Conexión Directa al Host o a VMs**: Conéctate a la máquina Windows anfitriona o salta por túnel SSH transparente hacia el puerto NAT de cualquier VM.
- **Emulación Completa ANSI / VT100**: Colores enriquecidos, soporte de teclas especiales, atajos interactivos (`Ctrl+C`, `Ctrl+D`, etc.) y redimensionamiento dinámico del terminal (`resize`).
- **Resaltado Semántico y Limpieza de Buffer**: Detección inteligente de prompts y rutas; manejo de buffers para evitar repeticiones visuales tras comandos de borrado como `cls` o `clear`.

---

### 7. Gestión Centralizada de Llaves SSH
- **Almacén Seguro de Identidades**: Listado completo de llaves vinculadas a cada máquina virtual en `keys.json`.
- **Descargas Directas**: Descarga de clave privada (`.pem`) y pública (`.pub`) para uso en clientes externos (OpenSSH, PuTTY, VS Code).
- **Autorecuperación Inteligente**: Si un archivo físico de clave privada se extravía en disco, el sistema lo regenera automáticamente desde el almacén local.

---

## Requisitos del Sistema

1. **Host Local (Servidor Web Vessel)**:
   - [Go 1.22+](https://go.dev/dl/) instalado.
   - Navegador web moderno (Chrome, Firefox, Edge).
2. **Host Remoto (Servidor de Virtualización)**:
   - Sistema operativo Windows con **Oracle VirtualBox 7.2** instalado y accesible en el `PATH` del sistema (`VBoxManage.exe`).
   - Red NAT de VirtualBox creada o configurable (`vessel-net` en `10.250.0.0/24`).
   - Servidor OpenSSH habilitado y configurado en el puerto `22` con acceso por clave privada.
   - Directorio configurado para almacenamiento de plantillas y máquinas virtuales.
3. **Máquinas Virtuales Invitadas (Docker)**:
   - Imagen Linux (Debian, Ubuntu, etc.) con el motor **Docker** instalado (`docker-ce` o `docker.io`) y servicio habilitado.
   - Acceso `sudo` sin contraseña para el usuario aprovisionado.

---

## Estructura del Proyecto

```
SshDesktop/
├── cmd/
│   └── ssh-web/                  # Punto de entrada principal (main.go)
├── internal/
│   ├── api/                      # Controladores HTTP REST API y WebSocket (/ws)
│   │   ├── clusters_handler.go   # Endpoints de Docker Swarm Clusters (SSE, ciclo de vida de nodos)
│   │   ├── handler.go            # Enrutador central HTTP y WebSocket
│   │   ├── hosts.go              # Gestión de perfiles de conexión SSH
│   │   ├── registry_handler.go   # Endpoints de Registry local y Pull-through Cache
│   │   └── vm_store.go           # Persistencia y adaptadores de VMs
│   ├── config/                   # Carga jerárquica de configuración (config.properties, .env y defaults)
│   ├── docker/                   # Motor integral de contenerización y orquestación
│   │   ├── catalog.go            # Catálogo de plantillas, categorías y configuraciones recomendadas
│   │   ├── registry_service.go   # Orquestador del Registry local (5000) y Cache (5001)
│   │   ├── registry_store.go     # Persistencia de estado de Registry (registry.json)
│   │   ├── registry_types.go     # Modelos de datos para Registry, Cache y daemon.json
│   │   ├── service.go            # Orquestador Docker vía SSH (contenedores, imágenes, volúmenes, redes)
│   │   ├── swarm_service.go      # Orquestador de Docker Swarm (init, join, drain, leave, topología)
│   │   ├── swarm_store.go        # Persistencia de clusters Docker Swarm (clusters.json)
│   │   ├── swarm_types.go        # Estructuras de datos para Swarm, nodos y progreso SSE
│   │   ├── types.go              # Modelos de datos principales (Container, Image, Volume, Network)
│   │   └── validator.go          # Validación con Regex y prevención de inyecciones de comandos POSIX
│   ├── keys/                     # Generador criptográfico Ed25519/RSA y almacén persistente (keys.json)
│   ├── ssh/                      # Cliente SSH, túnel PTY, integración con VBoxManage, provisioning y known_hosts
│   ├── template/                 # Descubrimiento, validación y sanitización de plantillas remotas
│   └── vm/                       # Lógica de ciclo de vida de VMs, asignador de puertos y almacén (vms.json)
├── pkg/
│   └── logger/                   # Logger estructurado JSON basado en log/slog
├── web/                          # Interfaz web cliente (HTML5, CSS3, JavaScript Vanilla)
│   ├── css/style.css             # Estilos oscuros, tarjetas, scrollbars personalizados y modales
│   ├── js/ansi.js                # Sanitizador ANSI, parser de colores y resaltador de rutas/prompts
│   ├── js/app.js                 # Controlador central de UI, wizard de Docker, Swarm y Registry
│   ├── js/websocket.js           # Cliente WebSocket robusto con reconexión limpia
│   └── index.html                # Estructura principal de la aplicación y modales
├── clusters.json                 # Almacén persistente de clusters Docker Swarm
├── config.properties             # Archivo de configuración base
├── hosts.json                    # Perfiles de hosts SSH guardados
├── keys.json                     # Almacén local de llaves criptográficas generadas
├── registry.json                 # Almacén de configuración de Registry y Cache
├── vms.json                      # Almacén local de máquinas virtuales registradas
└── README.md                     # Documentación completa del proyecto
```

---

## Configuración (`config.properties` / `.env`)

La aplicación lee su configuración de forma jerárquica: primero `config.properties`, luego `.env` (si existe), complementando con valores predeterminados.

| Propiedad (`config.properties`) | Variable (`.env`) | Requerido | Valor por Defecto | Descripción |
|---|---|---|---|---|
| `ssh.host` | `SSH_HOST` | Sí | - | IP o hostname del servidor Windows remoto con VirtualBox |
| `ssh.port` | `SSH_PORT` | No | `22` | Puerto SSH del host Windows |
| `ssh.user` | `SSH_USER` | Sí | `HP 245 RYZEN 3` | Usuario del host Windows remoto |
| `ssh.key_path` | `SSH_KEY_PATH` | Sí | - | Ruta absoluta a la clave privada SSH para conectar al host Windows |
| `ssh.shell` | `SSH_SHELL` | No | `powershell` | Shell por defecto del host (`powershell` o `cmd`) |
| `ssh.start_path` | `SSH_START_PATH` | No | `C:/Users/andres.HP245G7` | Directorio inicial predeterminado en el host |
| `vm.template_root` | `VM_TEMPLATE_ROOT` | No | `C:\VMManager\Templates` | Carpeta raíz en el host Windows donde residen las plantillas |
| `vm.vm_root` | `VM_VM_ROOT` | No | `C:\VMManager\VMs` | Directorio en el host donde VirtualBox almacena las VMs |
| `vm.ssh_root` | `VM_SSH_ROOT` | No | `C:\VMManager\SSH` | Directorio en el backend para identidades de VMs y `known_hosts` |
| `vm.bootstrap_key_path` | `VM_BOOTSTRAP_KEY_PATH` | No | `C:\VMManager\SSH\bootstrap_id_ed25519` | Clave privada de bootstrap para el primer aprovisionamiento |
| `vm.base_port` | `VM_BASE_PORT` | No | `22000` | Puerto base a partir del cual se asignan puertos NAT SSH |
| `vm.nat_network_name` | `VM_NAT_NETWORK_NAME` | No | `vessel-net` | Nombre de la NAT Network compartida en VirtualBox para comunicación entre VMs |
| `vm.nat_network_cidr` | `VM_NAT_NETWORK_CIDR` | No | `10.250.0.0/24` | Rango CIDR IPv4 privado de la red compartida `vessel-net` |
| `server.addr` | `SERVER_ADDR` | No | `:8080` | Dirección y puerto en el que escucha este servidor web |
| `log.level` | `LOG_LEVEL` | No | `info` | Nivel de logs: `debug`, `info`, `warn`, `error` |

---

## Arquitectura de Red y Virtualización Multi-Capa

La arquitectura desacopla por completo la virtualización del host y organiza las cargas mediante redes privadas dedicadas:

```
+-----------------------------------------------------------------------------------+
|                                  Navegador Web                                    |
|         (Terminal SSH, Wizard Contenedores, Clusters Swarm, Registry y Cache)     |
+-----------------------------------------+-----------------------------------------+
                                          | HTTP / WebSocket / SSE
                                          v
+-----------------------------------------------------------------------------------+
|                                   Backend Go                                      |
|               (API REST, Túneles SSH, Orquestador Docker, Swarm y Registry)       |
+-----------------------------------------+-----------------------------------------+
                                          | SSH Tunnel (Puerto 22 Host Windows)
                                          v
+-----------------------------------------------------------------------------------+
|                               Host Windows Remoto                                 |
|          (Oracle VirtualBox 7.2 - Red NAT vessel-net 10.250.0.0/24)               |
+--------------------+---------------------------------------------+----------------+
                     |                                             |
   [NIC 1: NAT Forwarding 22000]                 [NIC 1: NAT Forwarding 22001]
                     |                                             |
                     v                                             v
+-----------------------------------------+   +-------------------------------------+
|        VM 1: Manager / Registry         |   |            VM 2: Worker             |
|                                         |   |                                     |
|  [NIC 2: 10.250.0.5 en vessel-net]      |<->|  [NIC 2: 10.250.0.6 en vessel-net]  |
|                                         |   |                                     |
|  +-----------------------------------+  |   |  +-------------------------------+  |
|  |       Docker Engine (Daemon)      |  |   |  |     Docker Engine (Daemon)    |  |
|  |                                   |  |   |  |                               |  |
|  |  * Swarm Leader (Puerto 2377)     |  |   |  |  * Swarm Worker               |  |
|  |  * Local Registry (Puerto 5000)   |  |   |  |  * daemon.json configurado con|  |
|  |  * Pull-thru Cache (Puerto 5001)  |  |   |  |    mirrors hacia 10.250.0.5   |  |
|  |  * Contenedores de aplicaciones   |  |   |  |  * Contenedores de aplicacion |  |
|  +-----------------------------------+  |   |  +-------------------------------+  |
+-----------------------------------------+   +-------------------------------------+
```

---

## Referencia de la API REST y WebSocket

### Contenedores Docker (`/api/containers` y `/api/docker`)
| Método | Endpoint | Descripción | Cuerpo de Solicitud (JSON) |
|---|---|---|---|
| `GET` | `/api/docker/catalog` | Catálogo de aplicaciones recomendadas por categoría | - |
| `GET` | `/api/vms/{name}/docker/status` | Verifica la disponibilidad y versión de Docker en la VM | - |
| `GET` | `/api/containers` | Lista contenedores en todas las máquinas virtuales | - |
| `GET` | `/api/containers?vm={name}` | Filtra contenedores de una máquina virtual específica | - |
| `POST` | `/api/containers` | Crea y despliega un contenedor mediante el asistente visual | `{"vmName":"servidor","name":"mi-redis","image":"redis:alpine","ports":[{"hostPort":6379,"containerPort":6379,"protocol":"tcp"}],"restartPolicy":"unless-stopped"}` |
| `GET` | `/api/containers/{vm}/{id}` | Inspecciona la configuración, IP, puertos y montajes del contenedor | - |
| `POST` | `/api/containers/{vm}/{id}/start` | Inicia un contenedor detenido | - |
| `POST` | `/api/containers/{vm}/{id}/stop` | Detiene un contenedor en ejecución | - |
| `POST` | `/api/containers/{vm}/{id}/restart` | Reinicia un contenedor | - |
| `DELETE` | `/api/containers/{vm}/{id}?force=true` | Elimina el contenedor de la VM (soporta `--force`) | - |
| `GET` | `/api/containers/{vm}/{id}/logs?tail=200` | Obtiene las líneas recientes de logs emitidas por el contenedor | - |

### Imágenes Docker (`/api/images`)
| Método | Endpoint | Descripción | Cuerpo de Solicitud (JSON) |
|---|---|---|---|
| `GET` | `/api/images` | Lista imágenes Docker en todas las máquinas virtuales | - |
| `GET` | `/api/images?vm={name}` | Lista imágenes Docker en una máquina virtual específica | - |
| `POST` | `/api/images/build` | Construye una imagen a partir de un Dockerfile | `{"vmName":"servidor","imageName":"mi-app","tag":"1.0","dockerfile":"FROM alpine\n..."}` |
| `POST` | `/api/images/export` | Exporta una imagen Docker a archivo `.tar` en la VM | `{"vmName":"servidor","image":"nginx:alpine"}` |
| `GET` | `/api/images/exports?vm={name}` | Lista los archivos `.tar` exportados en la VM | - |
| `GET` | `/api/images/exports/download?vm={name}&file={fileName}` | Descarga directa del archivo `.tar` al navegador | - |
| `POST` | `/api/images/push` | Etiqueta y publica una imagen en Docker Hub | `{"vmName":"servidor","sourceImage":"mi-app:1.0","repository":"user/mi-app","tag":"1.0"}` |
| `GET` / `POST` | `/api/dockerhub/credentials` | Consulta estado o guarda credenciales para Docker Hub | `{"username":"user","access_token":"dckr_pat_..."}` |
| `DELETE` | `/api/images/{vm}/{id}?force=true` | Elimina una imagen Docker en la VM (soporta `--force`) | - |

### Volúmenes y Redes Docker (`/api/volumes` y `/api/networks`)
| Método | Endpoint | Descripción | Cuerpo de Solicitud (JSON) |
|---|---|---|---|
| `GET` | `/api/volumes?vm={name}` | Lista volúmenes Docker en la máquina virtual | - |
| `POST` | `/api/volumes` | Crea un volumen Docker persistente | `{"vmName":"servidor","name":"datos-app","driver":"local"}` |
| `DELETE` | `/api/volumes/{vm}/{name}` | Elimina un volumen Docker | - |
| `GET` | `/api/networks?vm={name}` | Lista redes Docker configuradas en la máquina virtual | - |
| `POST` | `/api/networks` | Crea una red Docker | `{"vmName":"servidor","name":"red-interna","driver":"bridge","subnet":"172.28.0.0/16"}` |
| `DELETE` | `/api/networks/{vm}/{id}` | Elimina una red Docker | - |

### Clusters Docker Swarm (`/api/clusters`)
| Método | Endpoint | Descripción | Cuerpo de Solicitud (JSON) |
|---|---|---|---|
| `GET` | `/api/clusters/eligible-vms` | Evalúa la elegibilidad de las VMs para formar clusters Swarm | - |
| `GET` | `/api/clusters` | Lista los clusters Docker Swarm registrados con estado en vivo | - |
| `POST` | `/api/clusters` | Crea un cluster Swarm (soporta `Accept: text/event-stream` para progreso SSE) | `{"name":"cluster-prod","managerVm":"nodo-manager","workerVms":["nodo-worker1","nodo-worker2"]}` |
| `GET` | `/api/clusters/{id}` | Inspecciona la topología, Leader, Managers y estado de los Workers | - |
| `POST` | `/api/clusters/{id}/workers` | Incorpora o reintenta la unión de un Worker al cluster | `{"vm":"nodo-worker3"}` |
| `DELETE` | `/api/clusters/{id}/nodes/{vm}?force=true` | Drena y expulsa un nodo del cluster (soporta forzado si está apagado) | - |

### Docker Registry & Pull-through Cache (`/api/registry`)
| Método | Endpoint | Descripción | Cuerpo de Solicitud (JSON) |
|---|---|---|---|
| `GET` | `/api/registry/status` | Estado global del Registry local, Pull-through Cache y nodos | - |
| `POST` | `/api/registry/deploy` | Despliega el Registry local (`target:"local"`, puerto 5000) o la Caché (`target:"cache"`, puerto 5001) | `{"target":"local","vmName":"nodo-1"}` |
| `POST` | `/api/registry/action` | Inicia, detiene, reinicia o purga datos de una instancia | `{"target":"cache","action":"purge"}` |
| `GET` | `/api/registry/logs?target={local\|cache}&tail=200` | Consulta los logs del contenedor del Registry o de la Caché | - |
| `GET` | `/api/registry/images?target=local` | Consulta el catálogo de imágenes y tags almacenados en el Registry | - |
| `DELETE` | `/api/registry/images?repository={repo}&tag={tag}&force=true` | Elimina una imagen del Registry y ejecuta Garbage Collection | - |
| `POST` | `/api/registry/push` | Etiqueta y sube una imagen de una VM hacia el Registry local | `{"vmName":"nodo-2","sourceImage":"mi-app:1.0","targetRepository":"mi-app","tag":"1.0"}` |
| `POST` | `/api/registry/pull` | Descarga una imagen desde el Registry local hacia una VM | `{"vmName":"nodo-2","repository":"mi-app","tag":"1.0"}` |
| `POST` | `/api/registry/smart-pull` | Descarga inteligente resolviendo desde Registry local, Cache o Hub | `{"vmName":"nodo-2","image":"nginx:alpine","preferRegistry":true}` |
| `POST` | `/api/registry/nodes/configure` | Aplica `registry-mirrors` e `insecure-registries` en `/etc/docker/daemon.json` de las VMs | `{"vms":["nodo-1","nodo-2"],"restartService":false}` |

### Máquinas Virtuales (`/api/vms`)
| Método | Endpoint | Descripción | Cuerpo de Solicitud (JSON) |
|---|---|---|---|
| `GET` | `/api/vms` | Lista todas las VMs registradas con estado en vivo | - |
| `POST` | `/api/vms` | Crea y aprovisiona una nueva VM a partir de plantilla | `{"name":"miVM","templateId":"debian-13.7.0","cores":2,"memoryMB":4096}` |
| `GET` | `/api/vms/{name}` | Obtiene los detalles y estado actual de una VM | - |
| `DELETE` | `/api/vms/{name}` | Elimina la VM de VirtualBox y limpia sus llaves y registros | - |
| `POST` | `/api/vms/{name}/start` | Inicia la VM en modo headless (y verifica adaptadores de red) | - |
| `POST` | `/api/vms/{name}/stop` | Apagado limpio de la VM mediante señal ACPI | - |
| `POST` | `/api/vms/{name}/poweroff` | Apagado forzado inmediato de la VM | - |
| `GET` / `POST` | `/api/vms/{name}/network-info` | Consulta o refresca el descubrimiento por MAC de la IP privada en `vessel-net` | - |
| `POST` | `/api/vms/{name}/connectivity` | Verifica la conectividad directa (ICMP + TCP) desde una VM hacia otra VM o IP en `vessel-net` | `{"target":"otraVM","targetPort":22}` |

### Plantillas (`/api/templates`)
| Método | Endpoint | Descripción |
|---|---|---|
| `GET` | `/api/templates` | Escanea y lista las plantillas válidas en el host remoto |
| `GET` | `/api/templates/{id}` | Obtiene la configuración de una plantilla específica |

### Llaves SSH (`/api/keys`)
| Método | Endpoint | Descripción |
|---|---|---|
| `GET` | `/api/keys` | Lista todas las llaves SSH generadas en el sistema |
| `GET` | `/api/keys/{id}/private` | Descarga el archivo de clave privada (`.pem`) |
| `GET` | `/api/keys/{id}/public` | Descarga el archivo de clave pública (`.pub`) |
| `DELETE` | `/api/keys/{id}` | Elimina el par de llaves del almacén |

### Terminal WebSocket (`/ws`)
| Protocolo | Endpoint | Parámetros Query | Descripción |
|---|---|---|---|
| `WSS` / `WS` | `/ws` | `host`, `port`, `user`, `shell`, `startPath` | Terminal interactivo contra el host Windows |
| `WSS` / `WS` | `/ws?vm={name}` | `vm={name}` | Terminal interactivo directo hacia la VM vía túnel SSH |
| `WSS` / `WS` | `/ws?vm={name}&container={id}` | `vm={name}`, `container={id}` | Consola interactiva directa dentro del contenedor (`docker exec -it <container> /bin/sh`) |

---

## Compilación y Puesta en Marcha

### 1. Compilación del Binario
```powershell
# Compilar el binario ejecutable optimizado
go build -o ssh-web.exe ./cmd/ssh-web
```

### 2. Ejecutar la Aplicación
```powershell
# Iniciar el servidor
.\ssh-web.exe
```

La consola mostrará el inicio del servidor:
```
{"time":"2026-09-27T13:30:00.000","level":"INFO","msg":"iniciando emulador de terminal ssh-web"}
{"time":"2026-09-27T13:30:00.050","level":"INFO","msg":"servidor HTTP a la escucha","direccion":":8080"}
```

### 3. Acceso a la Interfaz
Abre tu navegador en:
**http://localhost:8080**

- Haz clic en el **icono de engranaje** en la barra superior para abrir el panel de control.
- Alterna entre las pestañas:
  - **Hosts**: Conexión directa y perfiles de servidores SSH.
  - **Máquinas Virtuales**: Ciclo de vida, asignación de recursos y red privada `vessel-net`.
  - **Docker**: Submenú con **Contenedores**, **Imágenes**, **Clusters (Swarm)** y **Registry (Local & Cache)**.
  - **Llaves SSH**: Auditoría y descarga de claves Ed25519.
