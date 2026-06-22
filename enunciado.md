# 🧅 Onion VPN  - Testing

## 1. SETUP

* **OS:** Linux nativo (manipulacion del kernel).
* **Dependencias:** `make`, `g++`, `go`, `python3`.
* **Privilegios:** Ejecutar scripts de testing y `make` con sudo.

---

##  2. Paso 1: Levantar el Monitor (Terminal 1)

Captura y decodifica el trafico UDP de la red en tiempo real a nivel de sockets RAW.

```bash
sudo python3 tests/run_audit.py

```
---

##  3. Paso 2: Iniciar la Consola (Terminal 2)

Compilara todo el proyecto, creara la interfaz  `vpn0` (`10.8.0.1`), conectara el puente C++ y arranca el motor Go.

```bash
sudo python3 tests/interactive_demo.py

```

---

##  4. Paso 3: Verificación del Cifrado

1. En la **Terminal 2**, ingresa cualquier mensaje de texto.
2. En la **Terminal 1**, observa la trazabilidad del circuito:
* 🔴 **Rojo (Puerto 9999):** Tráfico inseguro interceptado en texto claro fuera de la VPN.
* 🟢 **Verde (Subida):** Flujo por los puertos `9001 ──► 9002 ──► 9003`. El `Cifrado Hex` cambia en cada salto, demostrando la remoción secuencial de capas.
* 🔵 **Azul (Bajada):** Retorno simétrico del paquete de confirmación cifrado hacia el cliente.


3. Escribe `exit` en la Terminal 2 para cerrar los procesos y limpiar las interfaces del sistema automáticamente.
