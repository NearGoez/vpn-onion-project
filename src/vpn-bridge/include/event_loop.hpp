#pragma once

#include <atomic>


class EventLoop { 
    private:
        int tun_fd;
        int ipc_fd;

        std::atomic<bool> running;

        void process_channels();

    public:
        EventLoop(int tun_fd, int ipc_fd);
        ~EventLoop();

        void start();

        void stop();
};



