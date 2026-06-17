# 🚀 Guía de Desarrollo: Motor de Red Criptográfico (vpn-core)

Este documento detalla el plan de trabajo modular dividido en hitos cronológicos para la implementación de la lógica en Go (`vpn-core`). Cada hito está diseñado para ser programado y testeado de forma 100% aislada de la interfaz en C++, respetando estrictamente el archivo `docs/CONTRATO_IPC.md`.

---

## 🔌 Hito 1: Servidor IPC (Unix Domain Socket)

* **Tiempo Estimado:** 6 horas.
* **Objetivo:** Establecer el canal de entrada y salida local actuando como servidor del socket de datagramas.
* **Directorios e Involucrados:**
* `src/vpn-core/internal/transport/ipc.go` (Crear este archivo)
* `src/vpn-core/cmd/main.go`


* **Mecanismo:**
* Implementar la escucha del socket usando `net.ListenUnixgram("unixgram", &net.UnixAddr{Name: "/tmp/onion_vpn.sock", Net: "unixgram"})`.
* Configurar un bucle de lectura asíncrono (*goroutine*) que lea datagramas en un buffer estático de `1500` o `2048` bytes.
* Implementar el manejo de señales (`os/signal`) en `main.go` para que, al cerrar el programa con `Ctrl+C`, se ejecute de forma obligatoria un defer con `os.Remove("/tmp/onion_vpn.sock")` para limpiar el entorno.


* **Sugerencia de Pruebas Aisladas:**
Levantar el binario de Go. Verificar que el archivo `/tmp/onion_vpn.sock` aparezca en el sistema. Usar `socat - UNIX-SENDTO:/tmp/onion_vpn.sock` desde otra terminal para inyectar texto crudo y verificar en los logs de Go que se reciben los bytes de forma atómica.

---

## 🔐 Hito 2: Motor Criptográfico (Primitivas Base)

* **Tiempo Estimado:** 8 horas.
* **Objetivo:** Implementar el cifrado/descifrado y el intercambio de claves que asegurarán el túnel.
* **Directorios e Involucrados:**
* `src/vpn-core/internal/crypto/handshake.go`
* `src/vpn-core/internal/crypto/cipher.go`


* **Mecanismo:**
* **`handshake.go`**: Utilizar la librería estándar `crypto/ecdh` (Curva P-256 o X25519) para generar el par de claves efímeras del nodo y computar el secreto compartido del Handshake. Usar `crypto/sha256` o HKDF para derivar las llaves simétricas finales de sesión.
* **`cipher.go`**: Implementar funciones de cifrado y descifrado simétrico utilizando un modo autenticado (AEAD) como `crypto/cipher.NewGCM` (AES-GCM) o ChaCha20-Poly1305. Cada paquete cifrado debe incluir su respectivo Nonce único (Vector de Inicialización).


* **Sugerencia de Pruebas Aisladas:**
Escribir pruebas unitarias estándar de Go (`*_test.go`) dentro del directorio `internal/crypto/`. Probar que un string cifrado con una clave derivada pueda ser descifrado correctamente, y que si se altera un solo bit del payload cifrado, la verificación de integridad de AES-GCM/ChaCha20 falle explícitamente.

---

## 🧅 Hito 3: Lógica Onion (Circuitos y Manejo de Celdas)

* **Tiempo Estimado:** 10 horas.
* **Objetivo:** Programar el empaquetado multicapa (encapsulamiento) y la toma de decisiones de la ruta de saltos.
* **Directorios e Involucrados:**
* `src/vpn-core/internal/onion/circuit.go`
* `src/vpn-core/internal/onion/cell.go`


* **Mecanismo:**
* Definir la estructura de una **Celda** del protocolo (el formato del paquete que viaja por internet pública: `CircuitID`, `CommandType`, `PayloadCifrado`).
* **`circuit.go`**: Mantener un mapa en memoria tipo `map[uint32]*CircuitState` que rastree los circuitos activos. Cada estado de circuito debe guardar el puntero de red al siguiente salto y las claves criptográficas simétricas asociadas a ese tramo.
* Implementar la envoltura (añadir capas de cifrado para subida) y el desempaque (remover capas de cifrado para bajada) según la posición del nodo en el circuito.


* **Sugerencia de Pruebas Aisladas:**
Simular un circuito de 3 saltos en una prueba unitaria creando 3 instancias separadas de las estructuras de cifrado. Pasar un buffer simulado (un array de bytes dummy) y verificar que tras aplicar las 3 operaciones de cifrado en cadena, se puedan revertir secuencialmente en el orden inverso exacto para recuperar los bytes originales.

---

## 🌐 Hito 4: Capa de Transporte UDP Global e Integración

* **Tiempo Estimado:** 10 horas.
* **Objetivo:** Unir el socket IPC local con sockets de red UDP públicos para realizar el reenvío de paquetes real entre proxies intermedios.
* **Directorios e Involucrados:**
* `src/vpn-core/internal/transport/udp.go`
* `src/vpn-core/cmd/main.go`


* **Mecanismo:**
* Levantar un socket de escucha pública mediante `net.ListenUDP("udp", &net.UDPAddr{Port: 9000})`.
* Configurar la orquestación de las dos *goroutines* core:
* **Goroutine IPC-to-UDP (Subida):** Lee paquete IP crudo del socket UNIX $\rightarrow$ cifra recursivamente con la lógica de `internal/onion/` $\rightarrow$ envía por el socket UDP público al primer proxy.
* **Goroutine UDP-to-IPC (Bajada):** Recibe del puerto UDP público $\rightarrow$ descifra/desenvuelve capas $\rightarrow$ si el paquete llegó a su destino final, escribe el paquete IP limpio resultante directo en el socket UNIX para C++.




* **Sugerencia de Pruebas Aisladas:**
Levantar dos instancias del proceso de Go en la misma máquina en puertos UDP distintos (ej. 9000 y 9001). Usar `socat` para enviar un mensaje simulado al socket IPC de la instancia 1, verificar mediante logs que viaja cifrado vía UDP a la instancia 2, se descifra, y se escribe en el socket IPC de la instancia 2 de forma transparente.
