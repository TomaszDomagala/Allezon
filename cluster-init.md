# Start cluster locally
```bash
kind create cluster --config kind.setup.yaml

# Metal https://kind.sigs.k8s.io/docs/user/loadbalancer/
kubectl apply -f https://raw.githubusercontent.com/metallb/metallb/v0.13.7/config/manifests/metallb-native.yaml

kubectl wait --namespace metallb-system \
                --for=condition=ready pod \
                --selector=app=metallb \
                --timeout=90s
                
docker network inspect -f '{{.IPAM.Config}}' kind

# Adjust ips in metallb.yaml accordingly
kubectl apply -f metallb.yaml
```
hisotry
# Install Redpanda
```bash
helm repo add redpanda https://charts.redpanda.com/
helm repo update
helm install redpanda redpanda/redpanda \
    --namespace redpanda \
    --create-namespace

# New topic for user tags called user-tags. Replication factor 3.
kubectl -n redpanda exec -ti redpanda-0 -c redpanda -- rpk topic create user-tags --replicas 3 --brokers redpanda-0.redpanda.redpanda.svc.cluster.local.:9093,redpanda-1.redpanda.redpanda.svc.cluster.local.:9093,redpanda-2.redpanda.redpanda.svc.cluster.local.:9093
```