<<<<<<< HEAD
# Dockerfile for tmux-intray testing isolation
# Provides an isolated environment to run tests and linting

FROM debian:bookworm-slim AS builder

# Install dependencies
RUN DEBIAN_FRONTEND=noninteractive apt-get update && apt-get install -y --no-install-recommends \
    bash \
    bats \
    make \
    shellcheck \
    shfmt \
    tmux \
    && rm -rf /var/lib/apt/lists/*

# Create a non-root user for safer execution
RUN useradd -m -s /bin/bash tester
USER tester
WORKDIR /home/tester/tmux-intray

# Set up temporary directories for isolated testing
ENV XDG_STATE_HOME=/tmp/xdg_state
ENV XDG_CONFIG_HOME=/tmp/xdg_config
ENV HOME=/tmp/home
ENV TERM=screen

# Copy project files (as root, then chown)
USER root
COPY . .
RUN chown -R tester:tester .
USER tester

# Default command: run tests
CMD ["make", "tests"]
=======
# Dockerfile for tmux-intray CLI
# Provides a containerized version of the tmux-intray CLI that can be run
# without installing dependencies on the host.

FROM alpine:3.20

# Install bash and other dependencies
RUN apk add --no-cache bash

# Create directory for tmux-intray
WORKDIR /usr/local/tmux-intray

# Copy entire project (excluding .dockerignore patterns)
COPY . .

# Make the main script executable
RUN chmod +x bin/tmux-intray

# Add tmux-intray bin to PATH
ENV PATH="/usr/local/tmux-intray/bin:$PATH"

# Verify installation
RUN /usr/local/tmux-intray/bin/tmux-intray --help

# Set default command to show help
ENTRYPOINT ["/usr/local/tmux-intray/bin/tmux-intray"]
CMD ["--help"]
>>>>>>> 07bda1f (feat(install): add multiple installation methods for tmux-intray CLI)
