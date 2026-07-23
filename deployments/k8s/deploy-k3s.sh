#!/bin/bash
set -e

echo "=========================================="
echo " Packaging Images for K3s on 172.16.60.190"
echo "=========================================="

# Build images locally
echo "[1/4] Building backend and frontend images..."
docker-compose -f ../docker-compose.yml build gateway frontend

# Save images to tarballs
echo "[2/4] Saving images to tarballs..."
docker save deployments-gateway:latest > gateway-image.tar
docker save deployments-frontend:latest > frontend-image.tar

echo "[3/4] Instructions for K3s server (172.16.60.190):"
echo "  1. Transfer these tarballs and the 'k8s' directory to the remote server:"
echo "     scp gateway-image.tar frontend-image.tar root@172.16.60.190:~/"
echo "     scp -r . root@172.16.60.190:~/k8s"
echo ""
echo "  2. SSH into the K3s server and import the images into containerd:"
echo "     ssh root@172.16.60.190"
echo "     k3s ctr images import gateway-image.tar"
echo "     k3s ctr images import frontend-image.tar"
echo ""
echo "  3. Apply the Kubernetes manifests:"
echo "     kubectl apply -f ~/k8s/"
echo ""
echo "[4/4] Done! The UI will be available via the K3s LoadBalancer on port 6081."
