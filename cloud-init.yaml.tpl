#cloud-config

package_update: true
package_upgrade: true

packages:
  - curl
  - git
  - iptables-persistent

runcmd:
  # Open ports in OCI's host-level iptables (security list alone is not enough on Ubuntu)
  - iptables -I INPUT 6 -m state --state NEW -p tcp --dport 22 -j ACCEPT
  - iptables -I INPUT 6 -m state --state NEW -p tcp --dport 80 -j ACCEPT
  - iptables -I INPUT 6 -m state --state NEW -p tcp --dport 443 -j ACCEPT
  - netfilter-persistent save

  # Install Docker via official script (includes Compose plugin)
  - curl -fsSL https://get.docker.com | sh
  - systemctl enable docker
  - systemctl start docker
  - usermod -aG docker ubuntu

  # Generate Appwrite config files without starting containers
  - mkdir -p /opt/appwrite
  - |
    docker run --rm \
      --volume /var/run/docker.sock:/var/run/docker.sock \
      --volume /opt/appwrite:/usr/src/code/appwrite:rw \
      --entrypoint="install" \
      appwrite/appwrite:1.9 \
      --http-port=80 \
      --https-port=443 \
      --interactive=N \
      --database=mongodb \
      --no-start

  # Patch domain in the generated .env
  - sed -i 's|^_APP_DOMAIN=.*|_APP_DOMAIN="${appwrite_domain}"|' /opt/appwrite/.env
  - sed -i 's|^_APP_DOMAIN_TARGET=.*|_APP_DOMAIN_TARGET="${appwrite_domain}"|' /opt/appwrite/.env

  # Start the stack
  - cd /opt/appwrite && docker compose up -d

  # Log completion marker for debugging
  - echo "Appwrite cloud-init complete $(date)" >> /var/log/appwrite-init.log
