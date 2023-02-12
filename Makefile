all: help

GIT_DIR := $(shell git rev-parse --show-toplevel)
WORK_DIR ?= $(GIT_DIR)
SRC_DIR ?= $(WORK_DIR)/src
DOCKER_DIR ?= $(WORK_DIR)/dockerfiles
CHARTS_DIR ?= $(WORK_DIR)/charts

# kind config.
KIND_SETUP_FILE ?= "kind.setup.yaml"
KIND_CLUSTER_NAME ?= "allezon-cluster"

# Helm config.
HELM_CHARTS ?= api allezon idgetter worker ippool
HELM_RELEASE_NAME ?= allezon

HELM_IPPOOL_RELEASE_NAME ?= $(HELM_RELEASE_NAME)-ippool

# DOCKER_BUILDKIT=1 is required to use the --mount option during docker build.
export DOCKER_BUILDKIT = 1

# https://gitlab.com/allezon/registry/container_registry
DOCKER_REPO ?= "registry.gitlab.com"

DOCKER_NAMESPACE ?= "registry.gitlab.com/allezon/registry"

# api service config
API_VERSION ?= "0.2.0"
API_DOCKER_REPO ?= "$(DOCKER_NAMESPACE)/api"
API_DOCKERFILE ?= "api.Dockerfile"

# id getter service config
ID_GETTER_VERSION ?= "0.2.0"
ID_GETTER_DOCKER_REPO ?= "$(DOCKER_NAMESPACE)/idgetter"
ID_GETTER_DOCKERFILE ?= "id_getter.Dockerfile"

# worker service config
WORKER_VERSION ?= "0.2.0"
WORKER_DOCKER_REPO ?= "$(DOCKER_NAMESPACE)/worker"
WORKER_DOCKERFILE ?= "worker.Dockerfile"

PORT_FORWARD_LOCAL_PORT ?= 8080
PORT_FORWARD_REMOTE_PORT ?= 8080
PORT_FORWARD_HOST ?= "rtb1"


# Development targets.

.PHONY: test
test: ## Run go tests.
	# Clean test cache as docker tests do not cache properly.
	go clean -testcache
	cd $(SRC_DIR) && go test -v ./...

.PHONY: lint
lint: ## Run golangci-lint.
	cd $(SRC_DIR) && golangci-lint run

# Docker build targets.

.PHONY: docker-build
docker-build: docker-build-api docker-build-idgetter docker-build-worker ## Build all docker images.

.PHONY: docker-build-api
docker-build-api: ## Build the API docker image.
	docker build -t "$(API_DOCKER_REPO):$(API_VERSION)" -t "$(API_DOCKER_REPO):latest" -f "$(DOCKER_DIR)/$(API_DOCKERFILE)" "$(SRC_DIR)"

.PHONY: docker-build-idgetter
docker-build-idgetter: ## Build the ID Getter docker image.
	docker build -t "$(ID_GETTER_DOCKER_REPO):$(ID_GETTER_VERSION)" -t "$(ID_GETTER_DOCKER_REPO):latest" -f "$(DOCKER_DIR)/$(ID_GETTER_DOCKERFILE)" "$(SRC_DIR)"

.PHONY: docker-build-worker
docker-build-worker: ## Build the Worker docker image.
	docker build -t "$(WORKER_DOCKER_REPO):$(WORKER_VERSION)" -t "$(WORKER_DOCKER_REPO):latest" -f "$(DOCKER_DIR)/$(WORKER_DOCKERFILE)" "$(SRC_DIR)"

.PHONY: docker-push
docker-push: docker-push-api docker-push-idgetter docker-push-worker ## Push all docker images.

.PHONY: docker-push-api
docker-push-api: ## Push the API docker image.
	docker push "$(API_DOCKER_REPO):$(API_VERSION)"

.PHONY: docker-push-idgetter
docker-push-idgetter: ## Push the ID Getter docker image.
	docker push "$(ID_GETTER_DOCKER_REPO):$(ID_GETTER_VERSION)"

.PHONY: docker-push-worker
docker-push-worker: ## Push the Worker docker image.
	docker push "$(WORKER_DOCKER_REPO):$(WORKER_VERSION)"

# Required once before push
.PHONY: docker-login
docker-login:
	docker login $(DOCKER_REPO)


# Kind targets. Kind is a tool for running local Kubernetes clusters using Docker container "nodes".
# It is used for local development and testing.

.PHONY: kind-create-cluster
kind-create-cluster: ## Create a kind cluster.
	kind create cluster --name "$(KIND_CLUSTER_NAME)" --config "$(WORK_DIR)/$(KIND_SETUP_FILE)"
	kubectl cluster-info --context kind-$(KIND_CLUSTER_NAME)

.PHONY: kind-delete-cluster
kind-delete-cluster: ## Delete a kind cluster.
	kind delete cluster --name "$(KIND_CLUSTER_NAME)"

.PHONY: kind-load
kind-load: kind-load-api kind-load-idgetter kind-load-worker ## Load all docker images into kind.

.PHONY: kind-load-api
kind-load-api: ## Load the API docker image into kind.
	kind load docker-image "$(API_DOCKER_REPO):$(API_VERSION)" --name "$(KIND_CLUSTER_NAME)"

.PHONY: kind-load-idgetter
kind-load-idgetter: ## Load the ID Getter docker image into kind.
	kind load docker-image "$(ID_GETTER_DOCKER_REPO):$(ID_GETTER_VERSION)" --name "$(KIND_CLUSTER_NAME)"

.PHONY: kind-load-worker
kind-load-worker: ## Load the Worker docker image into kind.
	kind load docker-image "$(WORKER_DOCKER_REPO):$(WORKER_VERSION)" --name "$(KIND_CLUSTER_NAME)"


# Helm targets. Helm is a package manager for Kubernetes.

.PHONY: helm-dependency-update
helm-dependency-update: ## Update all helm dependencies.
	$(foreach chart,$(HELM_CHARTS),helm dependency update $(CHARTS_DIR)/$(chart);)

.PHONY: helm-install
helm-install: ## Install allezon helm chart.
	helm install $(HELM_RELEASE_NAME) $(CHARTS_DIR)/allezon

.PHONY: helm-install-local
helm-install-local: ## Install allezon helm chart using local setup.
	helm install $(HELM_RELEASE_NAME) $(CHARTS_DIR)/allezon -f $(CHARTS_DIR)/local_deploy.yaml

.PHONY: helm-uninstall
helm-uninstall: ## Uninstall allezon helm chart.
	helm uninstall $(HELM_RELEASE_NAME)

.PHONY: helm-upgrade
helm-upgrade: ## Upgrade allezon helm chart.
	helm upgrade $(HELM_RELEASE_NAME) $(CHARTS_DIR)/allezon

.PHONY: helm-upgrade-local
helm-upgrade-local: ## Upgrade allezon helm chart using local setup.
	helm upgrade $(HELM_RELEASE_NAME) $(CHARTS_DIR)/allezon -f $(CHARTS_DIR)/local_deploy.yaml

# Local deployment targets. This is probably the most useful section of this Makefile.
# Use local-deploy to deploy allezon locally.
# For configuration changes, use local-deploy-update-helm to update the helm charts.

.PHONY: local-deploy
local-deploy: docker-build kind-delete-cluster kind-create-cluster kind-load helm-dependency-update helm-install-local ## Deploy allezon locally. Will delete the kind cluster if it already exists.

.PHONY: local-deploy-update
local-deploy-update: docker-build kind-load helm-dependency-update helm-upgrade-local ## Build and load docker images into kind and update helm charts on already running kind cluster.

.PHONY: local-deploy-update-helm
local-deploy-update-helm: helm-dependency-update helm-upgrade-local ## Update and install helm charts on already running kind cluster.


.PHONY: remote-port-forward
remote-port-forward: ## Forward the local kind cluster port to the remote VM.
	ssh -R  $(PORT_FORWARD_REMOTE_PORT):localhost:$(PORT_FORWARD_LOCAL_PORT) -N $(PORT_FORWARD_HOST)

# Real cluster deployment targets. These targets are used to deploy allezon to a remote cluster.

.PHONY: cluster-deploy
cluster-deploy: helm-dependency-update helm-install ## Deploy allezon to a remote cluster.

.PHONY: cluster-deploy-update
cluster-deploy-update: docker-build docker-push helm-dependency-update helm-upgrade ## Update allezon on a remote cluster.

.PHONY: cluster-uninstall
cluster-uninstall: helm-uninstall ## Uninstall allezon from a remote cluster.

.PHONY: cluster-loadbalancer-ip-install
cluster-loadbalancer-ip-install: ## Install the LoadBalancer IP address on the remote cluster.
	helm install $(HELM_IPPOOL_RELEASE_NAME) $(CHARTS_DIR)/ippool

.PHONE: cluster-loadbalancer-ip-uninstall
cluster-loadbalancer-ip-uninstall: ## Uninstall the LoadBalancer IP address from the remote cluster.
	helm uninstall $(HELM_IPPOOL_RELEASE_NAME)

# Misc targets.

.PHONY: help
help: ## Show Makefile help.
	@awk -F ':|##' '/^[^\t].+?:.*?##/ {printf "\033[36m%-25s\033[0m %s\n", $$1, $$NF}' $(MAKEFILE_LIST)
