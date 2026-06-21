#include "tun_device.hpp"
#include "ipc_socket.hpp"
#include "event_loop.hpp"

#include <iostream>
#include <csignal>
#include <memory>

// puntero global para poder acceder desde el loop
std::unique_ptr<EventLoop> global_loop = nullptr;

void signal_handler(int signum) {
    std::cout << "\n[MAIN] signal (" << signum << ") recibida. Matando componentes" << std::endl;
    if (global_loop) {
        global_loop->stop();
    }
}

int main() {
    struct sigaction sa;
    sa.sa_handler = signal_handler;
    sigemptyset(&sa.sa_mask);
    sa.sa_flags = 0;
    if (sigaction(SIGINT, &sa, nullptr) < 0 || sigaction(SIGTERM, &sa, nullptr) < 0) {
        std::cerr << "[CRITICAL] No se pudo iniciar el signar handler" << std::endl;
        return 1;
    }
    try {
        std::cout << "[MAIN] Iniciando.." << std::endl;

        TunDevice tun("vpn0");
        tun.initialize();

        IpcSocket ipc("/tmp/onion_vpn.sock");
        ipc.connect_server();

        global_loop = std::make_unique<EventLoop>(tun.get_fd(), ipc.get_fd());
        
        global_loop->start();

        global_loop.reset();
        ipc.close_socket();
        tun.close_device();
        std::cout << "[MAIN] finalizado" << std::endl;
    } catch (const std::exception& e) {
        std::cerr << "[CRITICAL] error en ejecucion" << e.what() << std::endl;
        return 1;
    }
    return 0;
}
