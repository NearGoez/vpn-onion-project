import os
import subprocess
import time
import sys
import re

GO_BINARY = "./src/vpn-core/vpn-core"
CPP_BINARY = "./src/vpn-bridge/vpn-bridge"
SOCKET_PATH = "/tmp/onion_vpn.sock"

def check_root():
    if os.getuid() != 0:
        print("[CRITICAL] Este test unificado modifica interfaces de red del kernel. Corre con 'sudo'.")
        sys.exit(1)

def run_test():
    check_root()
    
    # 1. Limpieza radical de entornos previos
    print("[INIT] Limpiando residuos de sockets e interfaces...")
    if os.path.exists(SOCKET_PATH):
        os.remove(SOCKET_PATH)
    subprocess.run(["ip", "link", "set", "vpn0", "down"], stderr=subprocess.DEVNULL)
    subprocess.run(["ip", "link", "delete", "vpn0"], stderr=subprocess.DEVNULL)

    go_process = None
    cpp_process = None

    try:
        # 2. Levantar el motor de Go en modo simulador completo (4 puertos UDP locales)
        print("[STEP 1] Levantando vpn-core en Go (Modo Simulador de Circuito Onion)...")
        go_process = subprocess.Popen(
            [GO_BINARY, "-mode", "all"],
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            text=True
        )
        
        # Esperar a que Go inicialice y cree el archivo de socket IPC
        timeout = 5
        while not os.path.exists(SOCKET_PATH):
            time.sleep(0.5)
            timeout -= 0.5
            if timeout <= 0:
                raise TimeoutError("El binario de Go no creó el socket IPC a tiempo.")

        # 3. Levantar el puente de C++ para interceptar el tráfico del kernel
        print("[STEP 2] Levantando vpn-bridge en C++ como proceso root...")
        cpp_process = subprocess.Popen(
            [CPP_BINARY],
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            text=True
        )
        time.sleep(1.0) # Permitir el ioctl de inicialización y el bind del socket

        # 4. Configurar el Stack de Red de Linux (Capa 3)
        print("[STEP 3] Configurando direccionamiento IP y activando vpn0...")
        subprocess.run(["ip", "addr", "add", "10.8.0.1/24", "dev", "vpn0"], check=True)
        subprocess.run(["ip", "link", "set", "vpn0", "up"], check=True)
        time.sleep(0.5)

        # 5. Inyección del Tráfico de Prueba (Ping Real)
        print("[STEP 4] Disparando ráfaga de pings reales a través del circuito cebolla...")
        ping_result = subprocess.run(
            ["ping", "-I", "vpn0", "-c", "4", "-W", "2", "10.8.0.5"],
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            text=True
        )

        print("\n=== SALIDA DEL COMANDO PING ===")
        print(ping_result.stdout)
        print("===============================\n")

        # 6. Análisis Métrico del Éxito del Test
        # Buscamos la línea de éxito: "4 packets transmitted, 4 received, 0% packet loss"
        if "0% packet loss" in ping_result.stdout:
            print("[SUCCESS] Test unificado completado. Tráfico de ida y vuelta verificado.")
            print("[INFO] El paquete IP pasó por C++, se cifró en 3 capas en Go, transitó localmente por 4 nodos UDP y retornó intacto.")
        else:
            print("[FAILED] Se detectó pérdida de paquetes en el circuito distribuido.")
            sys.exit(1)

    except Exception as e:
        print(f"[ERROR] Falla catastrófica en el arnés de pruebas: {e}")
        sys.exit(1)

    finally:
        # 7. Desmantelamiento seguro y ordenado de la infraestructura local
        print("\n[CLEANUP] Apagando procesos de forma controlada...")
        if cpp_process:
            cpp_process.terminate()
            cpp_process.wait()
            print(" -> vpn-bridge de C++ cerrado limpiamente.")
        if go_process:
            go_process.terminate()
            go_process.wait()
            print(" -> vpn-core de Go cerrado limpiamente.")
        
        if os.path.exists(SOCKET_PATH):
            os.remove(SOCKET_PATH)
        print("[CLEANUP] Entorno restaurado.")

if __name__ == "__main__":
    run_test()
