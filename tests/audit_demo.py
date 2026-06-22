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
import re

# --- CONFIGURACIÓN ---
GO_BINARY = "./src/vpn-core/vpn-core"
CPP_BINARY = "./src/vpn-bridge/vpn-bridge"
SOCKET_PATH = "/tmp/onion_vpn.sock"
WEB_URL = "http://localhost:8080/api/simulate?message="
TEST_MESSAGE = "SECURE_VPN_TEST_PEDRO_Y_DENIS"

# Colores para la terminal
C_RESET = "\033[0m"
C_BOLD = "\033[1m"
C_GREEN = "\033[32m"
C_RED = "\033[31m"
C_BLUE = "\033[34m"
C_CYAN = "\033[36m"
C_YELLOW = "\033[33m"

def print_header(title):
    print(f"\n{C_BOLD}{C_CYAN}=== {title} ==={C_RESET}")

def print_step(step, desc):
    print(f"{C_BOLD}{C_BLUE}[PASO {step}]{C_RESET} {desc}")

def check_requirements():
    # 1. Verificar ROOT
    if os.getuid() != 0:
        print(f"{C_BOLD}{C_RED}[ERROR] Este script requiere privilegios de ROOT (sudo) para crear interfaces TUN (vpn0) y capturar tráfico con tcpdump.{C_RESET}")
        sys.exit(1)
    
    # 2. Verificar tcpdump
    if not shutil.which("tcpdump"):
        print(f"{C_BOLD}{C_RED}[ERROR] tcpdump no está instalado. Instálalo con: sudo apt install tcpdump{C_RESET}")
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
    
    print_header("INICIANDO PRUEBA CENTRALIZADA Y AUDITORÍA DE RED")
    print(f"Este script levantará la infraestructura real (vpn0 + C++ + Go), auditará tráfico seguro/inseguro y comprobará el túnel.\n")
    
    run_cleanup()
    
    # 1. Compilación
    print_step(1, "Compilando binarios de C++ y Go...")
    build_res = subprocess.run(["make", "build"], stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True)
    if build_res.returncode != 0:
        print(f"{C_BOLD}{C_RED}[ERROR] Error de compilación:\n{build_res.stderr}{C_RESET}")
        sys.exit(1)
    print(f"{C_GREEN} -> Compilación exitosa.{C_RESET}")
    
    go_proc = None
    cpp_proc = None
    tcpdump_insecure = None
    tcpdump_secure = None
    
    try:
        # 2. Levantar Go
        print_step(2, "Levantando vpn-core de Go (Modo all: Cliente + Nodos + WebServer)...")
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
        print(f"{C_GREEN} -> Motor Go listo (Socket Unix creado y puertos UDP 9000-9003 listos).{C_RESET}")
        
        # 3. Levantar C++
        print_step(3, "Levantando vpn-bridge de C++...")
        cpp_proc = subprocess.Popen(
            [CPP_BINARY],
            stdout=subprocess.DEVNULL,
            stderr=subprocess.DEVNULL
        )
        time.sleep(1.0) # Esperar inicialización de la interfaz
        
        # 4. Configurar vpn0
        print_step(4, "Configurando la interfaz de red virtual vpn0 (10.8.0.1)...")
        subprocess.run(["ip", "addr", "add", "10.8.0.1/24", "dev", "vpn0"], check=True)
        subprocess.run(["ip", "link", "set", "vpn0", "up"], check=True)
        print(f"{C_GREEN} -> Interfaz vpn0 configurada y activa.{C_RESET}")
        
        # 5. Iniciar sniffers de red en segundo plano
        print_step(5, "Iniciando sniffers tcpdump en segundo plano (escuchando loopback 'lo')...")
        
        # Sniffer para el canal inseguro (UDP 9999) - Captura 1 paquete y muestra ASCII (-A)
        tcpdump_insecure = subprocess.Popen(
            ["tcpdump", "-i", "lo", "-nn", "-A", "-c", "1", "udp port 9999"],
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            text=True
        )
        
        # Sniffer para el canal seguro (UDP 9001 - Nodo 1) - Captura 1 paquete y muestra Hex+ASCII (-X)
        tcpdump_secure = subprocess.Popen(
            ["tcpdump", "-i", "lo", "-nn", "-X", "-c", "1", "udp port 9001"],
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            text=True
        )
        time.sleep(1.5) # Esperar a que tcpdump empiece a escuchar
        print(f"{C_GREEN} -> Capturadores de tráfico listos y a la ecuación.{C_RESET}")
        
        # 6. Disparar simulación (Tráfico del simulador)
        print_step(6, f"Simulando envío de datos: '{TEST_MESSAGE}'...")
        # Hacemos una petición HTTP al servidor de simulación
        url = WEB_URL + urllib.parse.quote(TEST_MESSAGE)
        req = urllib.request.Request(url)
        with urllib.request.urlopen(req) as response:
            res_data = response.read().decode('utf-8')
            simulation_logs = json.loads(res_data)
        
        print(f"{C_GREEN} -> Petición HTTP completada.{C_RESET}")
        print("\n⏳ Esperando capturas de red...")
        
        # Esperar a que terminen de capturar
        try:
            ins_out, ins_err = tcpdump_insecure.communicate(timeout=4)
            sec_out, sec_err = tcpdump_secure.communicate(timeout=4)
        except subprocess.TimeoutExpired:
            tcpdump_insecure.kill()
            tcpdump_secure.kill()
            ins_out, _ = tcpdump_insecure.communicate()
            sec_out, _ = tcpdump_secure.communicate()
            print(f"{C_RED}[WARN] Expiró el tiempo de espera para tcpdump.{C_RESET}")
        
        # 7. CONTRASTE DE TRÁFICO
        print_header("AUDITORÍA: TRÁFICO INSEGURO vs TRÁFICO SEGURO (ONION)")
        
        # Mostrar Inseguro
        print(f"{C_BOLD}{C_RED}🔴 TRÁFICO INSEGURO CAPTURADO EN PUERTO 9999 (Sin VPN):{C_RESET}")
        lines = ins_out.splitlines()
        # Buscamos líneas que contengan el mensaje de prueba
        payload_found = False
        for line in lines:
            if TEST_MESSAGE in line:
                print(f"  {C_YELLOW}>>> Mensaje en Texto Plano Detectado: {C_BOLD}'{line.strip()}'{C_RESET}")
                payload_found = True
        
        if not payload_found:
            # Imprimir salida cruda recortada si no encontramos el string directo
            print("\n".join(lines[-8:]))
        else:
            # Imprimir parte de la cabecera del paquete
            print(f"  Info de Red: {lines[0] if len(lines) > 0 else 'N/A'}")
            
        print("\n" + "-"*60 + "\n")
        
        # Mostrar Seguro
        print(f"{C_BOLD}{C_GREEN}🟢 TRÁFICO SEGURO CAPTURADO EN PUERTO 9001 (Onion Routing):{C_RESET}")
        lines_sec = sec_out.splitlines()
        
        # Mostrar las líneas que tienen el hex dump
        hex_lines = [l for l in lines_sec if re.match(r'^\s+0x', l)]
        header_line = lines_sec[0] if len(lines_sec) > 0 else "N/A"
        print(f"  Info de Red: {header_line}")
        print(f"  {C_CYAN}>>> Celda Onion Serializada (CircuitID + Comando + Longitud + Payload AES-GCM):{C_RESET}")
        for hl in hex_lines[:6]:
            print(f"    {hl}")
            
        # Comprobar que no exista el texto plano en el tráfico seguro
        if TEST_MESSAGE in sec_out:
            print(f"\n  {C_RED}[ALERTA] ¡El mensaje secreto se filtró en el puerto seguro!{C_RESET}")
        else:
            print(f"\n  {C_GREEN}✔ ÉXITO: El mensaje secreto '{TEST_MESSAGE}' NO aparece en texto plano en la captura del puerto 9001 (está cifrado).{C_RESET}")
            
        # 8. PRUEBA DEL TÚNEL REAL (PING)
        print_header("VERIFICACIÓN: PRUEBA DE FUEGO DEL TÚNEL vpn0")
        print_step(7, "Enviando pings reales del sistema a través de la interfaz virtual vpn0...")
        ping_res = subprocess.run(
            ["ping", "-I", "vpn0", "-c", "3", "-W", "2", "10.8.0.5"],
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            text=True
        )
        print("\n=== SALIDA DE PING ===")
        print(ping_res.stdout)
        print("======================\n")
        
        if "0% packet loss" in ping_res.stdout:
            print(f"{C_BOLD}{C_GREEN}✔ PRUEBA DEL TÚNEL vpn0: EXITOSA (0% de pérdida de paquetes).{C_RESET}")
        else:
            print(f"{C_BOLD}{C_RED}❌ PRUEBA DEL TÚNEL vpn0: FALLIDA (Pérdida de paquetes detectada).{C_RESET}")

    except Exception as e:
        print(f"{C_RED}[ERROR] Ocurrió un error en la ejecución: {e}{C_RESET}")
    
    finally:
        # Limpieza final
        print_header("LIMPIEZA DE ENTORNO")
        print("Apagando procesos de fondo y eliminando vpn0...")
        
        if tcpdump_insecure and tcpdump_insecure.poll() is None:
            tcpdump_insecure.kill()
        if tcpdump_secure and tcpdump_secure.poll() is None:
            tcpdump_secure.kill()
            
        if cpp_proc:
            cpp_proc.terminate()
            cpp_proc.wait()
            print(" -> vpn-bridge (C++) apagado.")
        if go_proc:
            go_proc.terminate()
            go_proc.wait()
            print(" -> vpn-core (Go) apagado.")
            
        run_cleanup()
        print(f"{C_GREEN} -> Limpieza completada. Sistema en estado original.{C_RESET}")

if __name__ == "__main__":
    main()
