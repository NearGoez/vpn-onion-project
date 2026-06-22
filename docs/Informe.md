## Informe Final

Integrantes: 
- Pedro González
- Denis González

### Seccion 1: Introducción y Relevancia del Problema

Esta sección define el problema de seguridad que aborda el proyecto: la interceptación de tráfico en redes locales públicas y la pérdida de anonimato del usuario.

- El problema de la privacidad en redes compartidas: cuando un usuario se conecta a WI-FI público (como el de un café), sus datos viajan a través del aire. Cualquier atacante con herramientas gratuitas (como Wireshark) puede interceptar y leer toda la información en texto plano (como contraseñas o chats sin cifrar), y rastrear exactamente quién la envió a través de su dirección IP de origen.
- Modelo de Amenazas del Proyecto:
  - Atacante: Un espía pasivo en la red local WI-FI
  - Objetivo del Atacante: Descubrir el contenido de los mensajes de los usuarios y asociar dichos mensajes a sus direcciones IP.
  - Objetivo de la Solución (Nuestra VPN): Asegurar la confidencialidad de los datos (que no se lean) y el anonimato del remitente (que no se sepa quién los mandó).

## Sección 2: Arquitectura General y Metodología

El sistema está diseñado bajo el principio de Separación de Políticas y Mecanismos, dividiendo el software en dos binarios independientes que corren en espacio de usuario:

### 1. Componentes del Monorepo: 

- El Mecanismo ( vpn-bridge  en C++): Su única responsabilidad es el manejo de bajo nivel con el sistema operativo Linux. Levanta la interfaz virtual TUN ( vpn0 ) en modo puro  IFF_TUN | IFF_NO_PI  (lo que garantiza que los paquetes leídos no contengan cabeceras extrañas). Lee los bytes de red y los inyecta en el canal local.
- La Política ( vpn-core  en Go): Es el motor de privacidad. Se encarga de la lógica criptográfica, la división de datos en celdas fijas, la negociación de llaves y el enrutamiento UDP multicapa.

### 2. La Frontera de Integración (IPC)
Ambos procesos se comunican exclusivamente a través de un **Unix Domain Socket** local de tipo datagrama (`SOCK_DGRAM`) en `/tmp/onion_vpn.sock`.
* **Uso de `SOCK_DGRAM`:** Elegido en lugar de `SOCK_STREAM` para mapear de forma atómica y 1-a-1 los paquetes de red IP. Cada lectura o escritura en el socket corresponde a exactamente un paquete IP completo, evitando la sobrecarga de buffers de flujo continuo.
* **El Contrato de Datos:** El buffer es el paquete IP crudo de Capa 3 sin metadatos propios de la VPN. El primer byte indica la versión (los 4 bits más significativos indican `0x4` para IPv4 o `0x6` para IPv6), con un tamaño máximo (MTU) de 1500 bytes.

---

## 3. Criptografía y Enrutamiento Multicapa (Onion Routing)

La seguridad del túnel Onion se basa en dos pilares fundamentales: el acuerdo de llaves simétricas efímeras y el cifrado por capas autenticado.

### 🔑 Intercambio de Llaves (ECDH P-256)
Para asegurar que ningún nodo intermedio conozca las claves de encriptación de los otros, el cliente realiza un intercambio de claves elípticas **Diffie-Hellman (ECDH)** usando la curva **P-256** de Go de forma independiente con cada nodo del circuito (Nodo 1, Nodo 2, Nodo 3).
1. El cliente genera un par de claves efímeras y obtiene el secreto compartido multiplicando su clave privada con la clave pública de un nodo.
2. A partir del secreto compartido, se aplica la función **SHA-256** para derivar una clave simétrica uniforme de exactamente **32 bytes** (256 bits), lista para el cifrado por bloques.

### 🔐 Cifrado Autenticado (AES-GCM)
Una vez acordadas las claves de sesión con cada nodo, toda la comunicación se cifra utilizando **AES en modo GCM** (Galois/Counter Mode).
* **Por qué AES-GCM (AEAD):** A diferencia de modos tradicionales como CBC, GCM es un modo de Cifrado Autenticado con Datos Asociados (AEAD). Esto garantiza tanto la **Confidencialidad** (los datos no pueden leerse) como la **Integridad** (los datos no pueden modificarse en tránsito). Si un atacante altera un solo bit del paquete cifrado, la firma de autenticación del GCM fallará al descifrar y el paquete se descartará inmediatamente.
* **Uso del Nonce:** Cada paquete encriptado incluye un Nonce (vector de inicialización) de 12 bytes aleatorio único al inicio del payload. Esto evita ataques de análisis de patrones y ataques de reproducción de tráfico.

### 🧅 El Protocolo Onion Routing (Paso a Paso)

```
        CLIENTE                NODO 1 (Entrada)          NODO 2 (Medio)           NODO 3 (Exit)
   [ Paquete IP Crudo ]
            |
   Cifrar con K3 (Exit)
            |
   Cifrar con K2 (Medio)
            |
   Cifrar con K1 (Entry)
            |
   [ Celda Cifrada 3 Capas ]
            |------- UDP port 9001 ------>
                                        |
                              Descifrar Capa 1 (K1)
                                        |
                             [ Celda Cifrada 2 Capas ]
                                        |------- UDP port 9002 ------>
                                                                    |
                                                          Descifrar Capa 2 (K2)
                                                                    |
                                                         [ Celda Cifrada 1 Capa ]
                                                                    |------- UDP port 9003 ------>
                                                                                                |
                                                                                      Descifrar Capa 3 (K3)
                                                                                                |
                                                                                       [ Paquete IP Crudo ]
```

#### Camino de Subida (Upload):
1. **El Cliente** toma el paquete IP crudo y lo cifra en orden inverso a la ruta: primero con la llave del Nodo 3 ($K_3$), luego con la del Nodo 2 ($K_2$) y finalmente con la del Nodo 1 ($K_1$).
2. Envía la celda cifrada en 3 capas al **Nodo 1** por UDP.
3. **El Nodo 1** descifra la capa exterior usando $K_1$ y reenvía el payload restante al **Nodo 2** por UDP.
4. **El Nodo 2** descifra la segunda capa usando $K_2$ y reenvía al **Nodo 3** por UDP.
5. **El Nodo 3** (Exit Node) descifra la última capa usando $K_3$ y revela el paquete IP crudo original, entregándolo al destinatario final.

#### Camino de Bajada (Download):
El camino de regreso realiza la operación en sentido inverso. Cada nodo añade su respectiva capa de cifrado simétrico a los datos de respuesta a medida que el paquete ya viaja de regreso hacia el cliente (Nodo 3 $\rightarrow$ Nodo 2 $\rightarrow$ Nodo 1 $\rightarrow$ Cliente). El cliente recibe una celda cifrada en 3 capas y aplica las tres desencriptaciones consecutivas (Llave 1 $\rightarrow$ Llave 2 $\rightarrow$ Llave 3) para recuperar el paquete de respuesta limpio.

---

## 4. Resultados y Demostración Práctica

Para demostrar la efectividad del prototipo de la VPN, diseñamos dos metodologías de pruebas en vivo que validan de forma transparente la confidencialidad y el enrutamiento del sistema:

### A. Consola Visual Interactiva (Dashboard)
Desarrollamos una interfaz gráfica web alojada localmente en el puerto `8080`. Esta interfaz permite enviar un mensaje y observar en paralelo dos flujos de datos en tiempo real:
* **Túnel Seguro (Onion VPN):** Muestra el recorrido del paquete por cada nodo y visualiza en la consola integrada las firmas y payloads hexadecimales encriptados.
* **Canal Inseguro (Wi-Fi Público):** Muestra cómo el mismo paquete viaja de forma directa en texto claro y es interceptado por un hacker simulado en tránsito.

### B. Interceptación y Análisis de Tráfico Real con `tcpdump`
Para demostrar que el sistema no es una simulación visual aislada, ejecutamos capturas de paquetes de red directamente en los sockets de la computadora usando `tcpdump`. Los resultados fueron los siguientes:

#### 1. Prueba de Vulnerabilidad (Sin la VPN)
Escuchando en el puerto UDP `9999` y enviando el mensaje `"PEDRO_Y_DENIS_HACKED"` sin protección, `tcpdump` capturó de inmediato el paquete en la interfaz física:
```text
0x0000:  4500 003c e90e 0000 4011 0000 7f00 0001  E..<....@.......
0x0010:  7f00 0001 d960 270f 0028 fe3b 5045 4452  .....`'..(.;PEDR
0x0020:  4f5f 595f 4445 4e49 535f 4841 434b 4544  O_Y_DENIS_HACKED
```
**Resultado:** El mensaje es 100% legible a la derecha de la captura de bytes (en texto plano ASCII), demostrando la total vulnerabilidad de los datos en tránsito directo.

#### 2. Prueba de Seguridad y Anonimato (Con la VPN)
Escuchando en el puerto UDP `9001` (tránsito entre Cliente y Nodo 1) y enviando el mismo mensaje a través del túnel Onion, `tcpdump` capturó el siguiente paquete:
```text
0x0000:  4500 0099 c695 0000 4011 0000 7f00 0001  E.......@.......
0x0010:  7f00 0001 e2ea 2329 0085 fe98 0000 0064  ......#).......d
0x0020:  0300 76d7 2ad5 2b21 1e12 df78 dee8 43ce  ..v.*.+!...x..C.
0x0030:  18a1 19fb 741d c2c5 3b56 eff6 ac7f e98e  ....t...;V......
```
* **Confidencialidad:** El contenido del mensaje se ha transformado en bytes de alta entropía totalmente ilegibles.
* **Confirmación del Protocolo:** La cabecera binaria muestra el comando `0x03` (CmdRelay) e identifica de forma exacta el circuito `0x64` (ID 100 en hexadecimal).
* **Evidencia del Enrutamiento Onion:** Al medir el tráfico en los siguientes saltos, el tamaño del datagrama pasó de **125 bytes** (en el puerto 9001) a **97 bytes** (en el puerto 9002). Esta contracción matemática de exactamente 28 bytes demuestra que cada nodo está "pelando" exitosamente su respectiva capa criptográfica AES-GCM (que contiene 12 bytes de nonce y 16 bytes de tag de autenticación), confirmando el funcionamiento físico del enrutamiento cebolla.

---

## 5. Limitaciones y Aprendizajes

A través del desarrollo de este proyecto, identificamos desafíos técnicos clave que nos permitieron profundizar en el diseño de redes seguras y criptografía de sistemas:

### Desafíos y Limitaciones
1. **Divergencias en el Kernel (macOS vs Linux):** La creación de interfaces de red virtuales (TUN) está fuertemente acotada en macOS moderno por políticas de seguridad de Apple (requiriendo System Extensions firmadas). Esto impidió compilar nativamente `vpn-bridge` (C++) en macOS. Como mitigación, diseñamos un arnés de pruebas y simulación web independiente del hardware que nos permitió depurar el 100% de la lógica en Go antes de integrarlo en el entorno Linux final.
2. **Latencia y Overhead:** La encriptación multicapa de AES-GCM añade un costo computacional y bytes extra a cada paquete. Además, el enrutamiento a través de tres saltos UDP incrementa la latencia (RTT) en comparación con un túnel VPN directo. Este es el trade-off clásico entre seguridad/privacidad y rendimiento.

### Aprendizajes Técnicos Clave
* **La importancia de los Contratos de Integración:** Redactar el contrato `docs/CONTRATO_IPC.md` el primer día fue crucial. Nos permitió codificar de forma paralela e independiente en dos lenguajes distintos (C++ y Go) sin una sola línea de código acoplada, logrando una integración instantánea el día de las pruebas.
* **AEAD para Seguridad en Red:** Aprendimos que la confidencialidad no sirve de nada sin integridad. Elegir un cifrado autenticado (AES-GCM) en lugar de modos clásicos previene ataques de inyección y modificación activa de tráfico en tránsito.

### Trabajo Futuro
* **Protocolo de Handshake Dinámico:** Implementar una negociación de claves efímeras por red (DH) en tiempo de ejecución al inicializar el circuito, reemplazando la simulación estática precompartida.
* **Soporte Multicliente:** Ampliar el seguimiento de estados del circuito en los nodos intermediarios para soportar múltiples flujos de usuarios concurrentes de forma aislada.
