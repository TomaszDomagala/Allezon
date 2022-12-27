# Kubernetes setup on RTB House provided vms
RTB House provides us with some vms to run our kubernetes cluster. This is a guide on how to set it up.

1. Create `.vms` file in the root of this repo following the example in `.vms_example` file.
In the `HOSTS` provide the ids of the vms that you want to use for your cluster (workers and masters).

2. You may want to re-deploy the vms to have a clean state. Go to the [jenkins](https://mimjenkins.rtb-lab.pl) and run the ReDeployVm job on the vms that you want to re-deploy.

3. Run the `./ssh-reset-all.sh` script to reset the ssh keys on the vms. At this point you should be able to ssh into the vms using the `ssh` command.

4. Build kubespray image. We will use the kubespray container to run ansible playbooks that will set up the cluster.
```
docker build -t kubespray kubespray/
```

5. Run the `setup-cluster.sh` script. This will run the kubespray container and run the ansible playbooks that will set up the cluster (it may take a while, around 20 minutes).
After successful setup there is one more script to execute on your master node, `ubuntu-fix.sh`. Without it you wont be able to run `kubectl get nodes` for example. 


The `cluster` ansible inventory was generated as specified in the [kubespray quick start guide](https://github.com/kubernetes-sigs/kubespray#quick-start).
You may want to generate your own (replace hosts.yaml with your own). The `hosts-ips.sh` script can help you with obtaining the ips of the vms.

