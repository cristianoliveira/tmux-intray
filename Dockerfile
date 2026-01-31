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
