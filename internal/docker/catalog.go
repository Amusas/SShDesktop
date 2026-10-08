package docker

// GetDefaultCatalog retorna el catálogo de aplicaciones y plantillas de contenedores organizadas por categoría
func GetDefaultCatalog() []CategoryCatalog {
	return []CategoryCatalog{
		{
			ID:          "databases",
			Name:        "Bases de Datos",
			Description: "Sistemas SQL y NoSQL",
			Items: []CatalogItem{
				{
					ID:           "postgres",
					Name:         "PostgreSQL",
					Category:     "databases",
					Description:  "BD relacional SQL avanzada",
					DefaultImage: "postgres:16-alpine",
					RecommendedPorts: []PortMapping{
						{HostPort: 5432, ContainerPort: 5432, Protocol: "tcp"},
					},
					RecommendedEnv: []EnvVar{
						{Key: "POSTGRES_DB", Value: "app_db"},
						{Key: "POSTGRES_USER", Value: "postgres"},
						{Key: "POSTGRES_PASSWORD", Value: "postgres"},
					},
					RecommendedVolumes: []VolumeMount{
						{HostPath: "pgdata", ContainerPath: "/var/lib/postgresql/data"},
					},
					DefaultRestart: "unless-stopped",
				},
				{
					ID:           "mysql",
					Name:         "MySQL",
					Category:     "databases",
					Description:  "BD relacional SQL estándar",
					DefaultImage: "mysql:8.4",
					RecommendedPorts: []PortMapping{
						{HostPort: 3306, ContainerPort: 3306, Protocol: "tcp"},
					},
					RecommendedEnv: []EnvVar{
						{Key: "MYSQL_ROOT_PASSWORD", Value: "rootpassword"},
						{Key: "MYSQL_DATABASE", Value: "app_db"},
						{Key: "MYSQL_USER", Value: "app_user"},
						{Key: "MYSQL_PASSWORD", Value: "app_password"},
					},
					RecommendedVolumes: []VolumeMount{
						{HostPath: "mysqldata", ContainerPath: "/var/lib/mysql"},
					},
					DefaultRestart: "unless-stopped",
				},
				{
					ID:           "mariadb",
					Name:         "MariaDB",
					Category:     "databases",
					Description:  "BD relacional compatible MySQL",
					DefaultImage: "mariadb:11",
					RecommendedPorts: []PortMapping{
						{HostPort: 3306, ContainerPort: 3306, Protocol: "tcp"},
					},
					RecommendedEnv: []EnvVar{
						{Key: "MARIADB_ROOT_PASSWORD", Value: "rootpassword"},
						{Key: "MARIADB_DATABASE", Value: "app_db"},
					},
					RecommendedVolumes: []VolumeMount{
						{HostPath: "mariadbdata", ContainerPath: "/var/lib/mysql"},
					},
					DefaultRestart: "unless-stopped",
				},
				{
					ID:           "mongodb",
					Name:         "MongoDB",
					Category:     "databases",
					Description:  "BD NoSQL de documentos JSON",
					DefaultImage: "mongo:7",
					RecommendedPorts: []PortMapping{
						{HostPort: 27017, ContainerPort: 27017, Protocol: "tcp"},
					},
					RecommendedEnv: []EnvVar{
						{Key: "MONGO_INITDB_ROOT_USERNAME", Value: "root"},
						{Key: "MONGO_INITDB_ROOT_PASSWORD", Value: "mongopassword"},
					},
					RecommendedVolumes: []VolumeMount{
						{HostPath: "mongodata", ContainerPath: "/data/db"},
					},
					DefaultRestart: "unless-stopped",
				},
			},
		},
		{
			ID:          "web",
			Name:        "Servidores Web",
			Description: "Servidores HTTP y proxies",
			Items: []CatalogItem{
				{
					ID:           "nginx",
					Name:         "Nginx",
					Category:     "web",
					Description:  "Servidor web y proxy de alto rendimiento",
					DefaultImage: "nginx:alpine",
					RecommendedPorts: []PortMapping{
						{HostPort: 80, ContainerPort: 80, Protocol: "tcp"},
					},
					RecommendedEnv:     []EnvVar{},
					RecommendedVolumes: []VolumeMount{},
					DefaultRestart:     "unless-stopped",
				},
				{
					ID:           "apache",
					Name:         "Apache HTTP",
					Category:     "web",
					Description:  "Servidor web HTTP clásico",
					DefaultImage: "httpd:alpine",
					RecommendedPorts: []PortMapping{
						{HostPort: 8080, ContainerPort: 80, Protocol: "tcp"},
					},
					RecommendedEnv:     []EnvVar{},
					RecommendedVolumes: []VolumeMount{},
					DefaultRestart:     "unless-stopped",
				},
				{
					ID:           "traefik",
					Name:         "Traefik",
					Category:     "web",
					Description:  "Proxy inverso y balanceador",
					DefaultImage: "traefik:v3.1",
					RecommendedPorts: []PortMapping{
						{HostPort: 80, ContainerPort: 80, Protocol: "tcp"},
						{HostPort: 8080, ContainerPort: 8080, Protocol: "tcp"},
					},
					RecommendedEnv:     []EnvVar{},
					RecommendedVolumes: []VolumeMount{},
					DefaultRestart:     "unless-stopped",
				},
			},
		},
		{
			ID:          "messaging",
			Name:        "Mensajería y Eventos",
			Description: "Colas de mensajes y brokers",
			Items: []CatalogItem{
				{
					ID:           "rabbitmq",
					Name:         "RabbitMQ",
					Category:     "messaging",
					Description:  "Broker de colas AMQP con UI web",
					DefaultImage: "rabbitmq:3-management-alpine",
					RecommendedPorts: []PortMapping{
						{HostPort: 5672, ContainerPort: 5672, Protocol: "tcp"},
						{HostPort: 15672, ContainerPort: 15672, Protocol: "tcp"},
					},
					RecommendedEnv: []EnvVar{
						{Key: "RABBITMQ_DEFAULT_USER", Value: "guest"},
						{Key: "RABBITMQ_DEFAULT_PASS", Value: "guest"},
					},
					RecommendedVolumes: []VolumeMount{
						{HostPath: "rabbitmqdata", ContainerPath: "/var/lib/rabbitmq"},
					},
					DefaultRestart: "unless-stopped",
				},
				{
					ID:           "mosquitto",
					Name:         "Mosquitto (MQTT)",
					Category:     "messaging",
					Description:  "Broker MQTT ligero para IoT",
					DefaultImage: "eclipse-mosquitto:latest",
					RecommendedPorts: []PortMapping{
						{HostPort: 1883, ContainerPort: 1883, Protocol: "tcp"},
					},
					RecommendedEnv:     []EnvVar{},
					RecommendedVolumes: []VolumeMount{},
					DefaultRestart:     "unless-stopped",
				},
			},
		},
		{
			ID:          "cache",
			Name:        "Caché en Memoria",
			Description: "Almacenes clave-valor de alta velocidad",
			Items: []CatalogItem{
				{
					ID:           "redis",
					Name:         "Redis",
					Category:     "cache",
					Description:  "Caché y almacén clave-valor",
					DefaultImage: "redis:alpine",
					RecommendedPorts: []PortMapping{
						{HostPort: 6379, ContainerPort: 6379, Protocol: "tcp"},
					},
					RecommendedEnv: []EnvVar{},
					RecommendedVolumes: []VolumeMount{
						{HostPath: "redisdata", ContainerPath: "/data"},
					},
					DefaultRestart: "unless-stopped",
				},
				{
					ID:           "memcached",
					Name:         "Memcached",
					Category:     "cache",
					Description:  "Caché distribuida en memoria",
					DefaultImage: "memcached:alpine",
					RecommendedPorts: []PortMapping{
						{HostPort: 11211, ContainerPort: 11211, Protocol: "tcp"},
					},
					RecommendedEnv:     []EnvVar{},
					RecommendedVolumes: []VolumeMount{},
					DefaultRestart:     "unless-stopped",
				},
			},
		},
		{
			ID:          "development",
			Name:        "Runtimes y Dev",
			Description: "Entornos de ejecución listos",
			Items: []CatalogItem{
				{
					ID:           "node",
					Name:         "Node.js",
					Category:     "development",
					Description:  "Runtime JavaScript Node 20",
					DefaultImage: "node:20-alpine",
					RecommendedPorts: []PortMapping{
						{HostPort: 3000, ContainerPort: 3000, Protocol: "tcp"},
					},
					RecommendedEnv:     []EnvVar{},
					RecommendedVolumes: []VolumeMount{},
					DefaultRestart:     "no",
				},
				{
					ID:           "python",
					Name:         "Python",
					Category:     "development",
					Description:  "Runtime Python 3 con pip",
					DefaultImage: "python:3.12-alpine",
					RecommendedPorts: []PortMapping{
						{HostPort: 8000, ContainerPort: 8000, Protocol: "tcp"},
					},
					RecommendedEnv:     []EnvVar{},
					RecommendedVolumes: []VolumeMount{},
					DefaultRestart:     "no",
				},
			},
		},
		{
			ID:          "custom",
			Name:        "Personalizado",
			Description: "Cualquier imagen Docker",
			Items: []CatalogItem{
				{
					ID:                 "custom",
					Name:               "Imagen Libre",
					Category:           "custom",
					Description:        "Ingresa la imagen de Docker Hub o registro",
					DefaultImage:       "",
					RecommendedPorts:   []PortMapping{},
					RecommendedEnv:     []EnvVar{},
					RecommendedVolumes: []VolumeMount{},
					DefaultRestart:     "unless-stopped",
				},
			},
		},
	}
}
