#include "tun_device.hpp"
#include <iostream>
#include <chrono>
#include <thread>

int main() {
    try {
        std::cout << "[MAIN] Iniciando componente vpn-bridge..."  << std::endl;

        TunDevice tun("vpn0");
        tun.initialize();

        std::cout << "[MAIN] Manteniendo proceso vivo. Verifica con: 'ip link show vpn0'" << std::endl;
        std::cout << "Presione Ctrl+C para destruir el proceso..." << std::endl;
        

        // loop temporal para mantener vivo el proceso.
        while (true) {
            std::this_thread::sleep_for(std::chrono::seconds(1));
        }
    } catch (const std::exception& e) {
        std::cerr << "[CRITICAL] Exception: " << e.what() << std::endl;
        return 1;
    }

    return 0;
}
