#pragma once

#include <string>
#include <sys/types.h>

class IpcSocket {
    private:
        int socket_fd;
        std::string socket_path;

    public:

        IpcSocket(const std::string& path = "/tmp/onion_vpn.sock");

        ~IpcSocket();

        void connect_server();

        ssize_t send_packet(const char* buffer, size_t length);
        ssize_t recv_packet(char* buffer, size_t max_length);
        void close_socket();
        int get_fd() const { return socket_fd; };
};


