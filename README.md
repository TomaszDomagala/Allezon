

# Allezon

Allezon is deployed on bare-metal kubernetes cluster, using helm charts (`/charts` directory).



* API Service - REST api that handles requests
  * `/user_tags` - adds the tag to user's profile, and sends kafka event to worker.
  * `/user_profiles/:cookie` - reads user profile from aerospike
  * `/aggregates` - reads aggregates from aerospike
* Worker Service - processes messages received from kafka and updates the aggregates in aerospike.
* ID Service - assignes and returns the numerical ID to elements from a given collection. Collecion are one of "origin", "brand", "category".



```mermaid


flowchart
    User --> LB[Load Balancer]
    LB --> API[API Service]
    API --> DB[Aerospike Cluster]
    API --> KAFKA[Kafka]
    API --> ID[ID Service]
    KAFKA --> WORKER[Worker Service]
    WORKER --> DB
    WORKER --> ID


```


# DB Model

We use 3 sets in out Aerospike db:
 - user_profiles:
  - COOKIE | (VIEWS) MAP[TIMESTAMP][USER_TAG] | (BUYS) MAP[TIMESTAMP][USER_TAG]
    - we use maps to mkae insertions atomic
 - aggregates:
   - stores aggregates in a format:
   TS-ORIGIN_ID-COLLECTION_ID-BRAND_ID | TS | (VIEWS)(count <<48 | sum)| (BUYS)(count <<48 | sum)
 - ids:
   - list of collections, brands and origin where idx in the list is id of corresponding collection. Used to make memory footprint of agggregates smaller


# Setup on RTB House provided vms
RTB House provides us with some vms to run our kubernetes cluster. This is a guide on how to set it up.

0. Install:
   - make
   - ansible
   - kubectl
   - optionally if you want to build yourself and not use prebuild images:
     - go(1.19)
     - docker

1. Create `.vms` file in the root of this repo following the example in `.vms_example` file.
In the `HOSTS` provide the ids of the vms that you want to use for your cluster (workers and masters).

2. You may want to re-deploy the vms to have a clean state. Go to the [jenkins](https://mimjenkins.rtb-lab.pl) and run the ReDeployVm job on the vms that you want to re-deploy.

3. Run to set up Kubernetes cluster and setup kubectl
   - adjust ips in `cluster/hosts.yaml`
   - ```bash
       make cluster-setup
       ./kubectl-setup.sh <username> <kubeadm address(node1)>
       ```

4. Create aerospike cluster
    - adjust ips in `aerospike/hosts`
    - ```bash
      ansible-playbook --extra-vars "ansible_user=<username> ansible_password=<password> ansible_ssh_extra_args='-o StrictHostKeyChecking=no'" -i aerospike/hosts aerospike/aerospike.yaml
      ```

5. Start service
    - adjust addresses in `charts/allezon/values.yaml` in all configs regarding DB(5) to those specified in `aerospike/hosts`
    - adjusts ip addresses in `charts/ippool/values.yaml`
      - addresses must be outside of address range that is taken by the vms.
    - ```bash
      make cluster-storage-install
      make helm-dependency-update helm-install
      make cluster-ippool-install
      ```
        - this will use pre-existing docker images if you want to build them yourself you would have to change the docker repo and use target `cluster-deploy` instead of `helm-dependency-update helm-install`

6. Run ELK for logs (optional)
   - ```bash
     make elk-operator-install
     make elk-install
     ```
   - port forward and get credentials(you may have to wait a bit as Kibana takes around minute to initiate)
     ```bash
     make elk-credentials elk-port-forward
     ```

7. To get IP address run, and look for external ip for allezon-api service.
    ```bash
    kubectl get svc
    ```   

The `cluster` ansible inventory was generated as specified in the [kubespray quick start guide](https://github.com/kubernetes-sigs/kubespray#quick-start).
You may want to generate your own (replace hosts.yaml with your own). The `hosts-ips.sh` script can help you with obtaining the ips of the vms.

