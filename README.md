[![Build Status](https://travis-ci.org/function61/james.svg?branch=master)](https://travis-ci.org/function61/james)
[![Download](https://api.bintray.com/packages/function61/james/main/images/download.svg)](https://bintray.com/function61/james/main/_latestVersion#files)

James is your friendly toolbox for infrastructure management.

Features:

- Build and deploy VM images (via Packer + Terraform)
- Manage DNS entries (via Cloudflare)
- Bootstrap Docker Swarm cluster
- List and acknowledge alerts (via lambda-alertmanager)
- Deploy Portainer to manage your cluster
- Connect via SSH to your cluster

Currently highly specific to needs of function61.com, might change in the future or might not.

Overview
--------

1. [Create VM image](#create-vm-image)
2. [Create VMs](#create-vms)
3. [Bootstrap cluster](#bootstrap-cluster)


Create VM image
---------------

We could just use ready-made images from the cloud provider, but that is a moving target.
We use Packer to at least get a snapshot of the image, so we know exactly what we're going to get.

```
$ james iac
$ 
```


Create VMs
----------

Use Terraform to bring up the VMs.


Bootstrap cluster
-----------------

Init swarm: (TODO: use James for this, release James)

- `$ docker swarm init`
- `$ docker swarm join`

Deploy dockersockproxy as a Swarm service

Use Portainer to:

- Deploy Monitoring
- Deploy Traefik

