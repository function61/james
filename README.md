[![Build Status](https://travis-ci.org/function61/james.svg?branch=master)](https://travis-ci.org/function61/james)
[![Download](https://api.bintray.com/packages/function61/james/main/images/download.svg)](https://bintray.com/function61/james/main/_latestVersion#files)

James is your friendly toolbox for infrastructure management.

Features:

- Build and deploy VM images (via Packer + Terraform) in immutable infrastructure manner
- Manage DNS entries (via Cloudflare)
- Bootstrap Docker Swarm cluster
- List and acknowledge alerts (via lambda-alertmanager)
- Deploy Portainer to manage your cluster
- Overlay networking (= all containers in cluster can communicate via encrypted channel)
- Connect via SSH to your cluster nodes

Currently highly specific to needs of function61.com, might change in the future or might not.

The stack is highly opinionated - it focuses on simplicity/immutability/security instead of flexibility.


Procedure for bringing up a cluster from zero to production
-----------------------------------------------------------

1. [Install, configure James](#install-configure-james)
2. [Create VM image](#create-vm-image)
3. [Create VMs](#create-vms)
4. [Bootstrap cluster](#bootstrap-cluster)
5. [Deploy system services](#deploy-system-services)


Install, configure James
------------------------

```
$ VERSION_TO_DOWNLOAD="..." # find this from Bintray link
$ sudo curl --location --fail --output /usr/local/bin/james "https://dl.bintray.com/function61/james/$VERSION_TO_DOWNLOAD/james" && sudo chmod +x /usr/local/bin/james
```

Create & fill details in `jamesfile.json` (TODO: document)


Create VM image
---------------

We could just use ready-made images from the cloud provider, but that is a moving target.
We use Packer to at least get a snapshot of the image, so we know exactly what we're going to get.

```
$ james iac
Entering infrastructure-as-code container. Press ctrl+c to exit
$ bin/build-digitalocean-coreos.sh
...
--> digitalocean: A snapshot was created: 'fn61-coreos-2018-09-24-08-55' (ID: 38442790) in regions 'ams3'
```

Take note of these things that you'll need later:

- the snapshot name (`fn61-coreos-2018-09-24-08-55`) created above - you'll need it later.
- CoreOS `PRETTY_NAME`. It's a good idea to document this in your Terraform config.

You should now be able to see the image in your DigitalOcean control panel, and it should be
under 100 megabytes (they probably use copy-on-write to provide deduplication).


Create VMs
----------

Randomize a name for your new VM:

```
# first enter infrastructure-as-code container
$ james iac
$ bin/randomservername.sh
misty-crushinator
```

Create `nodes.tf` with content (this will create your first VM):

```
locals {
	cluster = "prod4"
}

data "digitalocean_image" "fn61-coreos-1855-4-0-stable-2" {
	name = "fn61-coreos-2018-09-24-08-55"
}

module "misty_crushinator" {
	box = "misty-crushinator"
	size = "s-1vcpu-1gb"
	region = "ams3"
	cluster = "${local.cluster}"
	image = "${data.digitalocean_image.fn61-coreos-1855-4-0-stable-2.image}"

	source = "./droplet"
}
```

Then generate an execution plan (still in `iac` container)::

```
$ bin/plan.sh
Plan: 1 to add, 0 to change, 0 to destroy.
```

Then execute that very same plan (still in `iac` container):

```
$ bin/apply.sh
```

Terraform's strength are its execution plans - you can see exactly what it tries to do,
and when you execute it, Terraform doesn't do anything that's not in the plan. This
prevents "terrorform" (if you use it carefully).


Bootstrap cluster
-----------------

Import Terraform's box state into James (to let James know about boxes Terraform just created):

```
$ james boxes import
Updated Jamesfile with 1 boxes
```

Then bootstrap your cluster:

```
$ james boxes
prod4-misty-crushinator.do-ams3.fn61.net
$ james bootstrap prod4-misty-crushinator.do-ams3.fn61.net
...
```

James now bootstrapped your Docker Swarm cluster, configured an
[overlay network](https://docs.docker.com/network/overlay/) and deployed
[dockersockproxy](https://github.com/function61/dockersockproxy). You are now ready to
deploy Portainer (on your local computer for improved security):

```
$ james portainer deploy
...
Portainer should now be usable at http://localhost:9000/
```

Now enter Portainer and add your cluster's details:

```
$ james portainer details
Portainer connection details:
                  Name: prod4
          Endpoint URL: dockersockproxy.prod4.fn61.net:4431
                   TLS: Yes
              TLS mode: TLS with server and client verification
    TLS CA certificate: Download from https://function61.com/ca-certificate.crt
       TLS certificate: client-bundle.crt
               TLS key: client-bundle.crt
```


Deploy system services
----------------------

To bring your cluster up to game, deploy these stacks (TODO: document):

- Monitoring
- Traefik
