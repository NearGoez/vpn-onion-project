#!/usr/bin/env python3
# -*- coding: utf-8 -*-

import os
import sys
import socket
import struct

# Colores para la terminal
C_RESET = "\033[0m"
C_BOLD = "\033[1m"
C_GREEN = "\033[32m"
C_RED = "\033[31m"
C_BLUE = "\033[34m"
C_CYAN = "\033[36m"
C_YELLOW = "\033[33m"

def check_root():
    if os.getuid() != 0:
        print(f"{C_BOLD}{C_RED}[ERROR] Este script de auditoría requiere privilegios de ROOT (sudo) para abrir sockets RAW.{C_RESET}")
        sys.exit(1)

def main():
    check_root()
    
    print(f"{C_BOLD}{C_CYAN}=================================================={C_RESET}")
    print(f"{C_BOLD}{C_CYAN}    ONION VPN - TRAZABILIDAD MULTINODO EN VIVO    {C_RESET}")
    print(f"{C_BOLD}{C_CYAN}=================================================={C_RESET}")
    print("Monitoreando puertos: 9000 (Cliente), 9001 (N1), 9002 (N2), 9003 (N3) y 9999 (Inseguro).\n"
          "Envía un mensaje desde la terminal interactiva para ver el salto del paquete...\n")
    
    try:
        # Abrir socket RAW para escuchar paquetes UDP
        sock = socket.socket(socket.AF_INET, socket.SOCK_RAW, socket.IPPROTO_UDP)
    except PermissionError:
        print(f"{C_BOLD}{C_RED}[ERROR] Permiso denegado. Ejecuta con 'sudo'.{C_RESET}")
        sys.exit(1)
        
    print(f"{C_GREEN}✔ Monitor RAW de Circuito activo. Esperando ráfaga...{C_RESET}")
    print("-" * 75)
    
    while True:
        try:
            packet, addr = sock.recvfrom(65535)
            
            # Decodificar cabecera IP
            version_ihl = packet[0]
            ihl = (version_ihl & 0x0F) * 4
            
            if len(packet) < ihl + 8:
                continue
                
            # Decodificar cabecera UDP
            udp_header = packet[ihl : ihl+8]
            src_port, dest_port, length, checksum = struct.unpack("!HHHH", udp_header)
            
            # Extraer payload
            payload = packet[ihl + 8 :]
            
            # Identificar comandos Onion si aplica
            onion_info = ""
            if len(payload) >= 7 and dest_port in (9000, 9001, 9002, 9003):
                circuit_id = struct.unpack("!I", payload[0:4])[0]
                command = payload[4]
                payload_len = struct.unpack("!H", payload[5:7])[0]
                onion_payload = payload[7:7+payload_len]
                onion_info = f" [Circuito: {circuit_id} | Cmd: {command} | Cifrado Hex: {onion_payload.hex()[:24]}...]"
            
            # --- MAPEO DE FLUJO DE TRÁFICO (Camino de ida y vuelta) ---
            
            # 1. Tráfico Inseguro
            if dest_port == 9999:
                msg = payload.decode('utf-8', errors='replace')
                print(f"{C_BOLD}{C_RED}🔴 [SIN VPN - INSECURO] Cliente ──────> Servidor Final (Texto Claro){C_RESET}")
                print(f"   ├─ Puertos: {src_port} ──> {dest_port}")
                print(f"   └─ Contenido espiado: {C_BOLD}{C_YELLOW}'{msg}'{C_RESET}\n")

            # 2. Camino de subida (Upstream - Encriptando capas)
            elif dest_port == 9001 and src_port not in (9002, 9003):
                # Cliente o simulador enviando a Nodo 1
                print(f"{C_BOLD}{C_GREEN}🟢 [VPN - SUBIDA] Cliente ──────> Nodo 1 (Entrada de la Red){C_RESET}")
                print(f"   ├─ Puertos: {src_port} ──> {dest_port} (Cebolla con 3 capas de cifrado)")
                print(f"   └─ Datos:{C_CYAN}{onion_info}{C_RESET}\n")
                
            elif dest_port == 9002 and src_port == 9001:
                # Nodo 1 enviando a Nodo 2
                print(f"{C_BOLD}{C_GREEN}🟢 [VPN - SUBIDA] Nodo 1  ──────> Nodo 2 (Salto Intermedio){C_RESET}")
                print(f"   ├─ Puertos: {src_port} ──> {dest_port} (Quitada Capa 1 de 3)")
                print(f"   └─ Datos:{C_CYAN}{onion_info}{C_RESET}\n")
                
            elif dest_port == 9003 and src_port == 9002:
                # Nodo 2 enviando a Nodo Exit (3)
                print(f"{C_BOLD}{C_GREEN}🟢 [VPN - SUBIDA] Nodo 2  ──────> Nodo 3 (Exit Node - Salida){C_RESET}")
                print(f"   ├─ Puertos: {src_port} ──> {dest_port} (Quitada Capa 2 de 3 - Listo para entregar al destino)")
                print(f"   └─ Datos:{C_CYAN}{onion_info}{C_RESET}\n")

            # 3. Camino de bajada (Downstream - Envolviendo capas de respuesta)
            elif dest_port == 9002 and src_port == 9003:
                # Nodo Exit (3) enviando de regreso a Nodo 2
                print(f"{C_BOLD}{C_BLUE}🔵 [VPN - BAJADA] Nodo 3 (Exit) ──> Nodo 2 (Salto Intermedio){C_RESET}")
                print(f"   ├─ Puertos: {src_port} ──> {dest_port} (Añadida Capa 3 de cifrado)")
                print(f"   └─ Datos:{C_CYAN}{onion_info}{C_RESET}\n")
                
            elif dest_port == 9001 and src_port == 9002:
                # Nodo 2 enviando de regreso a Nodo 1
                print(f"{C_BOLD}{C_BLUE}🔵 [VPN - BAJADA] Nodo 2  ──────> Nodo 1 (Entrada de la Red){C_RESET}")
                print(f"   ├─ Puertos: {src_port} ──> {dest_port} (Añadida Capa 2 de cifrado)")
                print(f"   └─ Datos:{C_CYAN}{onion_info}{C_RESET}\n")
                
            elif dest_port == 9000 and src_port == 9001:
                # Nodo 1 enviando de regreso al Cliente
                print(f"{C_BOLD}{C_BLUE}🔵 [VPN - BAJADA] Nodo 1  ──────> Cliente (Túnel Finalizado){C_RESET}")
                print(f"   ├─ Puertos: {src_port} ──> {dest_port} (Añadida Capa 1 de cifrado - Cliente descifra todo)")
                print(f"   └─ Datos:{C_CYAN}{onion_info}{C_RESET}\n")
                
        except KeyboardInterrupt:
            print("\nDeteniendo sniffer...")
            break
        except Exception:
            continue

if __name__ == "__main__":
    main()
