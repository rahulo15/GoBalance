# --------------------------------------------------------
# STAGE 1: The Builder (Compiles the code)
# --------------------------------------------------------
FROM golang:alpine AS builder

# Set the working directory inside the container
WORKDIR /app

# Copy the source code
COPY . .

# Build the binary.
# -o lb        : Name the output file "lb"
# CGO_ENABLED=0: Disables C dependencies (makes it a static binary)
RUN CGO_ENABLED=0 go build -o lb main.go

# --------------------------------------------------------
# STAGE 2: The Runner (The final production image)
# --------------------------------------------------------
FROM alpine:latest

WORKDIR /root/

# Copy only the compiled binary from the builder stage
COPY --from=builder /app/lb .

# Copy the config file (CRITICAL!)
COPY config.docker.json ./config.json

# Expose port 8080 so Docker knows this port is important
EXPOSE 8080

# Command to run when the container starts
CMD ["./lb"]