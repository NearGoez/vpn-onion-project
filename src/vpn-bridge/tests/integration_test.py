import os
import socket
import subprocess
import time
import sys

SOCKET_PATH = "/tmp/onion_vpn.sock"
BRIDGE_PATH = "./vpn-bridge"

def setup_network():
    print("[TEST] Configurando interfaz virtual vpn0...")
    # Forzar que la interfaz esté abajo primero
    subprocess.run(["sudo", "ip", "link", "set", "vpn0", "down"], stderr=subprocess.DEVNULL)
    
    # NUEVO: Desactivar IPv6 en esta interfaz específica para mitigar paquetes automáticos del kernel
    subprocess.run(["sudo", "sysctl", "-w", "net.ipv6.conf.vpn0.disable_ipv6=1"], check=True, stdout=subprocess.DEVNULL)
    
    # Asignar direccionamiento IP y levantar la interfaz
    subprocess.run(["sudo", "ip", "addr", "add", "10.8.0.1/24", "dev", "vpn0"], check=True)
    subprocess.run(["sudo", "ip", "link", "set", "vpn0", "up"], check=True)

def run_integration_test():
    if os.getuid() != 0:
        print("[CRITICAL] El script de pruebas debe ejecutarse como root (sudo).")
        sys.exit(1)

    if os.path.exists(SOCKET_PATH):
        os.remove(SOCKET_PATH)

    server = socket.socket(socket.AF_UNIX, socket.SOCK_DGRAM)
    server.bind(SOCKET_PATH)
    server.settimeout(5.0) 

    bridge_process = None
    try:
        print("[TEST] Lanzando binario vpn-bridge...")
        bridge_process = subprocess.Popen(
            ["sudo", BRIDGE_PATH],
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            text=True
        )
        
        time.sleep(0.5)

        setup_network()

        print("[TEST] Inyectando paquete ICMP echo request...")
        subprocess.run(["ping", "-I", "vpn0", "-c", "1", "-W", "1", "10.8.0.5"], stdout=subprocess.DEVNULL)

        data, _ = server.recvfrom(2048)
        
        print(f"\n[TEST_RESULT] Datagrama interceptado exitosamente.")
        print(f" -> Tamaño del paquete: {len(data)} bytes")
        print(f" -> Payload en Hexadecimal: {data.hex()[:30]}...")

        assert len(data) == 84, f"Error: Tamaño esperado de ping estándar es 84 bytes, se obtuvo {len(data)}"
        assert data[0] == 0x45, f"Error: Cabecera IP corrupta o con PI flags activos, primer byte: {hex(data[0])}"
        
        print("\n[PASSED] Caso estándar: Interceptación correcta y cumplimiento de contrato.")

        print("\n[TEST] Iniciando caso borde: Paquete con tamaño MTU límite (1500 bytes)...")
        subprocess.run(["ping", "-I", "vpn0", "-c", "1", "-s", "1472", "-W", "1", "10.8.0.5"], stdout=subprocess.DEVNULL)
        
        data_mtu, _ = server.recvfrom(2048)
        print(f"[TEST_RESULT] Datagrama MTU interceptado.")
        print(f" -> Tamaño del paquete MTU: {len(data_mtu)} bytes")
        
        assert len(data_mtu) == 1500, f"Error: Se esperaban 1500 bytes exactos, se obtuvieron {len(data_mtu)}"
        assert data_mtu[0] == 0x45, "Error: Cabecera IP corrupta en paquete MTU."
        
        print("[PASSED] Caso borde MTU: Tráfico pesado transferido sin truncamiento.")

    except socket.timeout:
        print("\n[FAILED] Error: Tiempo de espera agotado. El bridge de C++ nunca envió datos al socket.")
    except AssertionError as e:
        print(f"\n[FAILED] Violación de aserción técnica: {e}")
    except Exception  as e:
        print(f"\n[ERROR] Fallo inesperado en el arnés de pruebas: {e}")
    finally:
        # 9. Desmantelamiento seguro y limpieza del entorno del sistema operativo
        print("\n[TEST] Iniciando desmontaje del entorno...")
        if bridge_process:
            bridge_process.terminate()
            try:
                bridge_process.wait(timeout=2)
                print("[TEST] Proceso vpn-bridge finalizado limpiamente.")
            except subprocess.TimeoutExpired:
                bridge_process.kill()
                print("[TEST] Forzado cierre de vpn-bridge de manera abrupta.")

        server.close()
        if os.path.exists(SOCKET_PATH):
            os.remove(SOCKET_PATH)
        print("[TEST] Entorno limpio de archivos socket e interfaces.")

if __name__ == "__main__":
    run_integration_test()
