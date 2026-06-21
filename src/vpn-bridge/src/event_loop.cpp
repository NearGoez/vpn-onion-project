#include "event_loop.hpp"

#include <iostream>
#include <vector>
#include <unistd.h>
#include <sys/poll.h>
#include <sys/socket.h>
#include <cerrno>

EventLoop::EventLoop(int t_fd, int i_fd) : tun_fd(t_fd), ipc_fd(i_fd), running(false) {}

EventLoop::~EventLoop() {
    stop();
}

void EventLoop::start() {
    if (running) return;
    running = true;
    std::cout << "Loop iniciado de eventos asincronos." << std::endl;
    process_channels();
}

void EventLoop::stop() {
    if (!running) return;
    running = false;
}

void EventLoop::process_channels() {
    struct pollfd fds[2];
    fds[0].fd = tun_fd;
    fds[0].events = POLLIN;
    fds[1].fd = ipc_fd;
    fds[1].events = POLLIN;

    std::vector<char> buffer(2048);

    while (running) {
        // Ejecutar poll
        int ready = poll(fds, 2, -1);
        
        if (ready < 0) {
            if (errno == EINTR) continue;
            std::perror("[DEBUG ERROR] poll() falló");
            break;
        }

        // LOG 1: Saber si poll() al menos despierta con el ping
        std::cout << "[DEBUG] poll() despertó. Descriptor listo = " << ready 
                  << " | tun_revents: " << fds[0].revents 
                  << " | ipc_revents: " << fds[1].revents << std::endl;

        // Canal 1: Subida (TUN -> IPC)
        if (fds[0].revents & POLLIN) {
            ssize_t bytes_read = read(tun_fd, buffer.data(), buffer.size());
            if (bytes_read < 0) {
                std::perror("[DEBUG ERROR] read() desde TUN falló");
                break;
            }
            
            // LOG 2: Saber si pudimos leer los bytes del kernel
            std::cout << "[DEBUG] TUN activo: Leídos " << bytes_read << " bytes. Enviando a Go..." << std::endl;

            if (bytes_read > 0) {
                ssize_t bytes_sent = send(ipc_fd, buffer.data(), bytes_read, 0);
                if (bytes_sent < 0) {
                    // LOG 3: Saber si el socket UNIX escupe un error
                    std::perror("[DEBUG ERROR] send() hacia IPC falló");
                } else {
                    std::cout << "[DEBUG] IPC activo: " << bytes_sent << " bytes enviados exitosamente." << std::endl;
                }
            }
        }

        // Canal 2: Bajada (IPC -> TUN)
        if (fds[1].revents & POLLIN) {
            ssize_t bytes_received = recv(ipc_fd, buffer.data(), buffer.size(), 0);
            if (bytes_received < 0) {
                std::perror("[DEBUG ERROR] recv() desde IPC falló");
                break;
            }
            if (bytes_received > 0) {
                std::cout << "[DEBUG] IPC activo: Recibidos " << bytes_received << " bytes. Inyectando a TUN..." << std::endl;
                write(tun_fd, buffer.data(), bytes_received);
            }
        }

        if ((fds[0].revents & (POLLERR | POLLHUP)) || (fds[1].revents & (POLLERR | POLLHUP))) {
            std::cerr << "[LOOP ERROR] Un descriptor se colgó o cerró." << std::endl;
            break;
        }
    }
    running = false;
   

}
