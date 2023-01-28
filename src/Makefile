all: help

GIT_DIR := $(shell git rev-parse --show-toplevel)
WORK_DIR ?= $(GIT_DIR)
SRC_DIR ?= $(WORK_DIR)/src
DOCKER_DIR ?= $(WORK_DIR)/dockerfiles
HELM_DIR ?= $(WORK_DIR)/helm

# kind config.
KIND_SETUP_FILE ?= "kind.setup.yaml"
KIND_CLUSTER_NAME ?= "allezon-cluster"

# Helm config.
HELM_CHARTS ?= api allezon
HELM_RELEASE_NAME ?= allezon

# DOCKER_BUILDKIT=1 is required to use the --mount option during docker build.
export DOCKER_BUILDKIT = 1

# api service config
API_VERSION ?= "0.1.0"
API_DOCKER_REPO ?= "allezon/api"
API_DOCKERFILE ?= "api.Dockerfile"

# id getter service config
ID_GETTER_VERSION ?= "0.1.0"
ID_GETTER_DOCKER_REPO ?= "allezon/idgetter"
ID_GETTER_DOCKERFILE ?= "id_getter.Dockerfile"

# worker service config
WORKER_VERSION ?= "0.1.0"
WORKER_DOCKER_REPO ?= "allezon/worker"
WORKER_DOCKERFILE ?= "worker.Dockerfile"


# Docker build targets.

.PHONY: docker-build
docker-build: docker-build-api docker-build-idgetter docker-build-worker ## Build all docker images.

.PHONY: docker-build-api
docker-build-api: ## Build the API docker image.
	docker build -t "$(API_DOCKER_REPO):$(API_VERSION)" -f "$(DOCKER_DIR)/$(API_DOCKERFILE)" "$(SRC_DIR)"

.PHONY: docker-build-idgetter
docker-build-idgetter: ## Build the ID Getter docker image.
	docker build -t "$(ID_GETTER_DOCKER_REPO):$(ID_GETTER_VERSION)" -f "$(DOCKER_DIR)/$(ID_GETTER_DOCKERFILE)" "$(SRC_DIR)"

.PHONY: docker-build-worker
docker-build-worker: ## Build the Worker docker image.
	docker build -t "$(WORKER_DOCKER_REPO):$(WORKER_VERSION)" -f "$(DOCKER_DIR)/$(WORKER_DOCKERFILE)" "$(SRC_DIR)"


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
	$(foreach chart,$(HELM_CHARTS),helm dependency update $(HELM_DIR)/$(chart);)

.PHONY: helm-install
helm-install: ## Install allezon helm chart.
	helm install $(HELM_RELEASE_NAME) $(HELM_DIR)/allezon

.PHONY: helm-uninstall
helm-uninstall: ## Uninstall allezon helm chart.
	helm uninstall $(HELM_RELEASE_NAME)

.PHONY: helm-upgrade
helm-upgrade: ## Upgrade allezon helm chart.
	helm upgrade $(HELM_RELEASE_NAME) $(HELM_DIR)/allezon

# Local deployment targets.

.PHONY: local-deploy
local-deploy: docker-build kind-delete-cluster kind-create-cluster kind-load helm-dependency-update helm-install ## Deploy allezon locally. Will delete the kind cluster if it already exists.

.PHONY: local-deploy-update
local-deploy-update: docker-build kind-load helm-dependency-update helm-upgrade ## Update the local deployment. This will not delete the kind cluster.


# Misc targets.

.PHONY: help
help: ## Show Makefile help.
	@awk -F ':|##' '/^[^\t].+?:.*?##/ {printf "\033[36m%-25s\033[0m %s\n", $$1, $$NF}' $(MAKEFILE_LIST)
