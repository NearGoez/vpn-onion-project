import socket
import os
import sys
import time
import threading

SERVER_SOCKET_PATH = "/tmp/onion_vpn.sock"
CLIENT_SOCKET_PATH = "/tmp/onion_vpn_client.sock"

def listen_responses(client_socket):
    print("[TEST CLIENT] Escuchando respuestas de Go en segundo plano...")
    client_socket.settimeout(3.0)
    try:
        while True:
            data, _ = client_socket.recvfrom(2048)
            if not data:
                break
            # Intentamos ver si es texto
            try:
                msg = data.decode()
                print(f"\n[TEST CLIENT] ¡Respuesta recibida desde la VPN! -> '{msg}'")
            except UnicodeDecodeError:
                print(f"\n[TEST CLIENT] ¡Paquete IP recibido desde la VPN! -> Tamaño: {len(data)} bytes | Hex: {data.hex()[:20]}...")
    except socket.timeout:
        print("[TEST CLIENT] Fin de la escucha (tiempo de espera agotado).")
    except Exception as e:
        print(f"[TEST CLIENT] Error escuchando: {e}")

def main():
    if not os.path.exists(SERVER_SOCKET_PATH):
        print(f"[TEST CLIENT] Error: El socket {SERVER_SOCKET_PATH} no existe. ¿Arrancaste vpn-core en Go?")
        sys.exit(1)

    # Limpieza del socket de este cliente si quedó de antes
    if os.path.exists(CLIENT_SOCKET_PATH):
        os.remove(CLIENT_SOCKET_PATH)

    client = socket.socket(socket.AF_UNIX, socket.SOCK_DGRAM)
    
    # IMPORTANTE para Mac: Nos enlazamos (bind) a una ruta temporal propia
    # Esto le da una "dirección física" a nuestro cliente para que Go sepa cómo responderle.
    client.bind(CLIENT_SOCKET_PATH)
    
    print(f"[TEST CLIENT] Conectándose al socket de Go: {SERVER_SOCKET_PATH}")
    client.connect(SERVER_SOCKET_PATH)

    # Iniciamos un hilo para escuchar las respuestas descifradas que Go nos mande de vuelta
    listener_thread = threading.Thread(target=listen_responses, args=(client,), daemon=True)
    listener_thread.start()

    # Caso 1: Enviamos un mensaje de texto para demostrar Onion Routing de forma visual
    secret_message = "Hola, esta es una prueba ultra secreta en capas."
    print(f"\n[TEST CLIENT] Enviando mensaje de texto secreto: '{secret_message}'")
    client.send(secret_message.encode())
    time.sleep(1)

    # Caso 2: Enviamos un paquete IPv4 falso (para emular un ping interceptado)
    # Empieza con 0x45, longitud 28 bytes. Contiene una cabecera ICMP Echo Request (Tipo 8 en byte 20)
    fake_ping_packet = bytearray([
        0x45, 0x00, 0x00, 0x1c, # Versión/IHL, ToS, Longitud (28 bytes)
        0x00, 0x01, 0x00, 0x00, # ID, Flags/Fragmento
        0x40, 0x01, 0x00, 0x00, # TTL (64), Protocolo (1 = ICMP), Checksum IP (vacío)
        0x0a, 0x08, 0x00, 0x01, # IP Origen: 10.8.0.1
        0x0a, 0x08, 0x00, 0x02, # IP Destino: 10.8.0.2
        0x08, 0x00, 0x00, 0x00, # ICMP: Tipo (8 = Request), Código (0), Checksum (se calcula)
        0x12, 0x34, 0x56, 0x78  # Datos dummy del ICMP
    ])
    
    print(f"\n[TEST CLIENT] Enviando paquete IP falso (ICMP Echo Request)...")
    client.send(fake_ping_packet)
    time.sleep(1.5)

    client.close()
    if os.path.exists(CLIENT_SOCKET_PATH):
        os.remove(CLIENT_SOCKET_PATH)
    print("\n[TEST CLIENT] Finalizado.")

if __name__ == "__main__":
    main()
