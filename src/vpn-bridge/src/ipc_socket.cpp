#include "../include/ipc_socket.hpp"

#include <iostream>
#include <stdexcept>
#include <cstring>
#include <unistd.h>
#include <sys/socket.h>
#include <sys/un.h>

IpcSocket::IpcSocket(const std::string& path) : socket_fd(-1), socket_path(path) {}

IpcSocket::~IpcSocket() {
    close_socket();
}

void IpcSocket::connect_server() {

    socket_fd = socket(AF_UNIX, SOCK_DGRAM, 0);

    if (socket_fd < 0) {
        throw std::runtime_error("ERROR: no se puede crear el socket IPC");
    }

    struct sockaddr_un addr;

    std::memset(&addr, 0, sizeof(addr));
    addr.sun_family = AF_UNIX;

    if (socket_path.size() >= sizeof(addr.sun_path)) {

        close_socket();
        throw std::runtime_error("ERROR: ruta del socket demasiado larga");
    }

    std::strncpy(addr.sun_path, socket_path.c_str(), sizeof(addr.sun_path) - 1);

    int err = connect(socket_fd, (struct sockaddr*)&addr, sizeof(addr));
    if (err < 0) {
        close_socket();
        throw std::runtime_error("ERR: no se pudo conectar" + socket_path + ", revise si vpn corriendo");
    }

    std::cout << "IPC: conexion exitosa con " << socket_fd << std::endl;
}

ssize_t IpcSocket::send_packet(const char* buffer, size_t length) {

    ssize_t bytes_sent = send(socket_fd, buffer, length, 0);
    if (bytes_sent < 0) {
        std::cerr << "IPC ERR: fallo envio de paquete" << std::endl;
    }

    return bytes_sent;
}

ssize_t IpcSocket::recv_packet(char* buffer, size_t max_length) {
    ssize_t bytes_received = recv(socket_fd, buffer, max_length, 0);
    if (bytes_received < 0) {
        std::cerr << "IPC ERR: fallo recibiendo el paquete." << std::endl;
    }
    return bytes_received;
}

void IpcSocket::close_socket() {
    if (socket_fd >= 0) {
        close(socket_fd);

        std::cout << "IPC: socket cerrado correctamente" << std::endl;
        socket_fd = -1;
    }
}
