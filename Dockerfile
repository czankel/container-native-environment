FROM ubuntu:focal-20221130

RUN	apt update && apt install -y ca-certificates curl gnupg lsb-release && \
	mkdir -p /etc/apt/keyrings && \
	curl -fsSL https://download.docker.com/linux/ubuntu/gpg | gpg --dearmor -o /etc/apt/keyrings/docker.gpg && \
	echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable" | tee /etc/apt/sources.list.d/docker.list > /dev/null && \
	apt update && \
	apt install -y containerd.io && \
	rm -rf /var/lib/apt/lists/*

# Copy CNE tool
COPY cne /cne

# Run CNE
ENTRYPOINT ["/cne"]
