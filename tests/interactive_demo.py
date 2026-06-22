#!/usr/bin/env python3
# -*- coding: utf-8 -*-

import os
import sys
import time
import subprocess
import urllib.request
import urllib.parse
import json
import shutil

# --- CONFIGURACIÓN ---
GO_BINARY = "./src/vpn-core/vpn-core"
CPP_BINARY = "./src/vpn-bridge/vpn-bridge"
SOCKET_PATH = "/tmp/onion_vpn.sock"
WEB_URL = "http://localhost:8080/api/simulate?message="

# Colores para la terminal
C_RESET = "\033[0m"
C_BOLD = "\033[1m"
C_GREEN = "\033[32m"
C_RED = "\033[31m"
C_BLUE = "\033[34m"
C_CYAN = "\033[36m"
C_YELLOW = "\033[33m"

def check_requirements():
    if os.getuid() != 0:
        print(f"{C_BOLD}{C_RED}[ERROR] Este script requiere privilegios de ROOT (sudo) para crear la interfaz TUN (vpn0) y configurar el ruteo.{C_RESET}")
        sys.exit(1)

def run_cleanup():
    # Remover socket e interfaces residuales
    if os.path.exists(SOCKET_PATH):
        try:
            os.remove(SOCKET_PATH)
        except OSError:
            pass
    subprocess.run(["ip", "link", "set", "vpn0", "down"], stderr=subprocess.DEVNULL)
    subprocess.run(["ip", "link", "delete", "vpn0"], stderr=subprocess.DEVNULL)

def main():
    check_requirements()
    
    print(f"{C_BOLD}{C_CYAN}=================================================={C_RESET}")
    print(f"{C_BOLD}{C_CYAN}       ONION VPN - CONSOLA INTERACTIVA            {C_RESET}")
    print(f"{C_BOLD}{C_CYAN}=================================================={C_RESET}")
    print("Este script compilará el proyecto, levantará la interfaz virtual vpn0,\n"
          "conectará el puente C++ y el motor Go, e iniciará un bucle de envío.\n")
    
    run_cleanup()
    
    # 1. Compilación
    print(f"{C_BOLD}{C_BLUE}[1/4]{C_RESET} Compilando C++ y Go...")
    build_res = subprocess.run(["make", "build"], stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True)
    if build_res.returncode != 0:
        print(f"{C_BOLD}{C_RED}[ERROR] Falló la compilación:\n{build_res.stderr}{C_RESET}")
        sys.exit(1)
    print(f"{C_GREEN} -> Compilación exitosa.{C_RESET}")
    
    go_proc = None
    cpp_proc = None
    
    try:
        # 2. Levantar Go
        print(f"{C_BOLD}{C_BLUE}[2/4]{C_RESET} Levantando vpn-core de Go (Modo simulador completo)...")
        go_proc = subprocess.Popen(
            [GO_BINARY, "-mode", "all"],
            stdout=subprocess.DEVNULL,
            stderr=subprocess.DEVNULL
        )
        
        # Esperar a que se cree el socket IPC
        timeout = 5.0
        while not os.path.exists(SOCKET_PATH):
            time.sleep(0.2)
            timeout -= 0.2
            if timeout <= 0:
                raise TimeoutError("El binario de Go no creó el socket IPC a tiempo.")
        print(f"{C_GREEN} -> Motor Go listo (puertos UDP 9000-9003 y servidor web iniciados).{C_RESET}")
        
        # 3. Levantar C++
        print(f"{C_BOLD}{C_BLUE}[3/4]{C_RESET} Levantando vpn-bridge de C++...")
        cpp_proc = subprocess.Popen(
            [CPP_BINARY],
            stdout=subprocess.DEVNULL,
            stderr=subprocess.DEVNULL
        )
        time.sleep(1.0)
        
        # 4. Configurar vpn0
        print(f"{C_BOLD}{C_BLUE}[4/4]{C_RESET} Creando y configurando interfaz TUN 'vpn0'...")
        subprocess.run(["ip", "addr", "add", "10.8.0.1/24", "dev", "vpn0"], check=True)
        subprocess.run(["ip", "link", "set", "vpn0", "up"], check=True)
        print(f"{C_GREEN} -> Interfaz vpn0 activa en 10.8.0.1.{C_RESET}")
        
        time.sleep(1.0) # Asegurar estabilidad de la red
        
        print(f"\n{C_BOLD}{C_GREEN}✔ ESTRUCTURA INICIALIZADA CORRECTAMENTE.{C_RESET}")
        print("La VPN real está operativa en background mediante vpn0.")
        print("Escribe tus mensajes a continuación. El script los enviará en paralelo:")
        print(f"  - Por el canal seguro {C_GREEN}Onion (UDP 9001 -> 9002 -> 9003){C_RESET}")
        print(f"  - Por el canal inseguro {C_RED}Texto Plano (UDP 9999){C_RESET}")
        print("(Escribe 'exit' o presiona Ctrl+C para salir)\n")
        
        while True:
            try:
                message = input(f"{C_BOLD}{C_YELLOW}Mensaje a transmitir > {C_RESET}").strip()
                if not message:
                    continue
                if message.lower() in ("exit", "quit"):
                    break
                
                # Consumir el API de simulación
                url = WEB_URL + urllib.parse.quote(message)
                req = urllib.request.Request(url)
                
                try:
                    with urllib.request.urlopen(req) as response:
                        response.read() # Consumir la respuesta del servidor
                        print(f"{C_GREEN}✔ Mensaje enviado con éxito (Canal Seguro Onion y Canal Inseguro disparados).{C_RESET}\n")
                except Exception as api_err:
                    print(f"{C_RED}[ERROR] No se pudo conectar al servidor de simulación: {api_err}{C_RESET}")
                    
            except KeyboardInterrupt:
                print("\nSaliendo...")
                break
                
    except Exception as e:
        print(f"{C_RED}[ERROR] Ocurrió una falla crítica: {e}{C_RESET}")
        
    finally:
        print(f"\n{C_BOLD}{C_BLUE}[LIMPIEZA]{C_RESET} Desmantelando procesos de background e interfaces...")
        if cpp_proc:
            cpp_proc.terminate()
            cpp_proc.wait()
        if go_proc:
            go_proc.terminate()
            go_proc.wait()
        run_cleanup()
        print(f"{C_GREEN} -> Workspace limpio y vpn0 removida.{C_RESET}")

if __name__ == "__main__":
    main()
