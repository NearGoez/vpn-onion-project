import os
import socket
import sys

SOCKET_PATH = "/tmp/onion_vpn.sock"


def main():
    if os.path.exists(SOCKET_PATH):
        os.remove(SOCKET_PATH)
    print(f"[MOCK GO] Creando servidor Unix Domain Socket (SOCK_DGRAM) en: {SOCKET_PATH}")
    server = socket.socket(socket.AF_UNIX, socket.SOCK_DGRAM)
    server.bind(SOCKET_PATH)
    print("[MOCK GO] Esperando datagramas del cliente C++... (Presiona Ctrl+C para salir)")
    try:
        while True:
            data, _ = server.recvfrom(2048)
            if not data:
                break
            print(f"\n[MOCK GO] Datagrama recibido! Tamaño: {len(data)} bytes")
            hex_payload = data.hex()
            print(f"[MOCK GO] Payload (HEX): {hex_payload}")
            
            if data[0] == 0x45:
                print("[MOCK GO] Validación: Cabecera IPv4 correcta (0x45). Contrato respetado.")
            else:
                print("[MOCK GO] ¡ALERTA! El primer byte no es 0x45. Contrato violado.")
                
    except KeyboardInterrupt:
        print("\n[MOCK GO] Apagando servidor de pruebas...")
    finally:
        server.close()
        if os.path.exists(SOCKET_PATH):
            os.remove(SOCKET_PATH)
            print("[MOCK GO] Archivo de socket eliminado limpiamente.")

if __name__ == "__main__":
    main()
