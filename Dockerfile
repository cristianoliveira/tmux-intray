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
