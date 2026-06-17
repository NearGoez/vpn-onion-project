# 🧅 VPN con Cifrado Propio TUN/TAP - Root Repository

Este es el repositorio central del proyecto de VPN con enrutamiento multicapa (tipo Onion Routing) para el curso de Seguridad Computacional. El sistema está diseñado dividiendo estrictamente la gestión de bajo nivel de interfaces del sistema operativo y la lógica criptográfica pesada en dos subprocesos independientes que se comunican de forma asíncrona.

---

## 🌐 1. Arquitectura General del Sistema

El proyecto implementa la separación de políticas y mecanismos dividiendo el software en dos binarios independientes:

```
+------------------------------------------------------------------------+
|                           ESPACIO DE KERNEL                            |
|  [Aplicaciones del Sistema] ---> [Tabla de Ruteo] ---> [Interfaz TUN]  |
+------------------------------------------------------------------------+
                                                              |
                                                    Paquetes IP Crudos
                                                              v
+------------------------------------------------------------------------+
|                          ESPACIO DE USUARIO                            |
|                                                                        |
|  +---------------------------+             +------------------------+  |
|  |     src/vpn-bridge/       |             |     src/vpn-core/      |  |
|  |         (C++)             |             |          (Go)          |  |
|  |                           |  Datagramas |                        |  |
|  | Intercepta tráfico TUN y  | <=========> | Servidor IPC, Cifrado, |  |
|  | lo redirige al socket IPC.|  SOCK_DGRAM | Lógica Onion y Nodos.  |  |
|  +---------------------------+             +------------------------+  |
|                                                         |              |
|                                                 Datagramas UDP         |
|                                                         v              |
|                                                [Internet Pública]      |
+------------------------------------------------------------------------+

```

* **`vpn-bridge` (C++)**: Actúa como el capturador de red. Abre el clonador de dispositivos de Linux, levanta la interfaz virtual TUN, extrae los bytes crudos del tráfico y los inyecta en el canal local.
* **`vpn-core` (Go)**: Actúa como el motor inteligente. Es el encargado de inicializar el circuito de anonimato, negociar llaves criptográficas simétricas efímeras, envolver el tráfico en capas cifradas y enviarlo a los nodos intermedios vía UDP.

---

## 🔌 2. La Frontera de Integración (Fricción Cero)

Para permitir el desarrollo paralelo con acoplamiento nulo, ambos procesos se comunican exclusivamente a través de un **Unix Domain Socket** de tipo datagrama (`SOCK_DGRAM`) ubicado en `/tmp/onion_vpn.sock`.

### Reglas Técnicas Inmutables:

1. **Flujo de Datos Puro**: No existe un protocolo intermedio entre C++ y Go. El buffer que viaja por el socket es exactamente el paquete de Capa 3 (IP) tal cual sale del kernel.
2. **Sin Sincronización de Código**: Go no importa código C++ (no usa Cgo) y C++ no sabe nada del runtime de Go. Esto permite compilar, debugear y testear cada componente de manera autónoma con herramientas de simulación como `socat`.
3. **Manejo del Ciclo de Vida**: El binario en Go (`vpn-core`) actúa como servidor levantando el socket. El binario en C++ (`vpn-bridge`) actúa como cliente conectándose a él.

---

## 📂 3. Estructura de Directorios del Monorepo

El repositorio utiliza una estrategia de monorepo estructurada de forma corporativa para simplificar el despliegue y mantener centralizada la documentación de la interfaz:

```text
vpn-onion-project/
├── docs/
│   └── CONTRATO_IPC.md      # Especificación inmutable del socket y los buffers.
├── src/
│   ├── vpn-bridge/          # Código fuente en C++ (Captura de tráfico de red).
│   │   ├── include/         # Cabeceras (.hpp).
│   │   └── src/             # Implementaciones (.cpp).
│   └── vpn-core/            # Código fuente en Go (Criptografía y enrutamiento).
│       ├── cmd/             # Punto de entrada de la aplicación.
│       └── internal/        # Paquetes privados (crypto, onion, transport).
├── Makefile                 # Orquestador global de compilación.
└── README.md                # Este archivo de inducción.

```

---

## 📅 4. Roadmap de Desarrollo Cronológico

El flujo de trabajo conjunto está diseñado en 4 fases lógicas para evitar bloqueos mutuos durante el desarrollo del proyecto:

* **Fase 1: Fijación del Contrato (Día 1)**: Ambos integrantes validan el archivo `docs/CONTRATO_IPC.md`. A partir de este punto, las firmas de los buffers y las rutas quedan congeladas.
* **Fase 2: Implementación Aislada**:
* Denis avanza en `vpn-bridge` programando el bucle asíncrono con `poll()` para mover datos del dispositivo TUN al socket local.
* Pedro avanza en `vpn-core` programando las goroutines de red, el handshake criptográfico ECDH y el encapsulamiento de celdas onion.


* **Fase 3: Integración Local**: Se levantan ambos binarios en la misma máquina física. Se verifica que el tráfico interceptado por C++ llegue limpio a Go a través del socket local.
* **Fase 4: Despliegue del Circuito**: Se ejecutan múltiples instancias de `vpn-core` en puertos UDP diferentes para emular los nodos de salto intermedio y validar la política de anonimato.

---

## 🛠️ 5. Requisitos del Sistema y Compilación Global

### Requisitos:

* Sistema Operativo: GNU/Linux (Se requieren privilegios de `root` o capacidades `CAP_NET_ADMIN` para crear y manipular interfaces TUN/TAP).
* Compilador: `g++` con soporte para C++17 o superior.
* Entorno: `Go 1.18` o superior.
* Herramientas de depuración recomendadas: `wireshark`, `tcpdump`, `socat`.

### Automatización de Compilación:

El `Makefile` ubicado en la raíz permite gestionar la compilación de todo el ecosistema de forma centralizada sin necesidad de ingresar a cada subdirectorio manualmente:

```bash
# Compilar ambos binarios (C++ con optimizaciones -O3 y Go)
make build

# Limpiar los ejecutables y archivos temporales del sistema
make clean

```
