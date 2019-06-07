# eirinix-sample-extension


This is a sample extension for Eirinix

## Requirements

Go >= 1.12

## How to use

Simply build the project and run it.

    $> git clone https://github.com/mudler/eirinix-sample-extension
    $> cd eirinix-sample-extension
    $> go build

### External from a kube cluster

    $> ./eirinix-sample-extension --kubeconfig "path/to/my/kubeconfig" --namespace "eirini" --operator-webhook-host "my-ip-reacheable-from-the-cluster" --operator-webhook-port 8889

### In-cluster

You can run the extension from an application pod - just don't specify the kubeconfig as an option, it will connect with in-cluster credentials.

    $> ./eirinix-sample-extension --namespace "eirini" --operator-webhook-host "my-ip-reacheable-from-the-cluster" --operator-webhook-port 8889

The Host/Port where the extension is binding, needs to be accessible by the kubeapi server. One way to do this, for e.g. is creating a new service and re-use the ClusterIP as --operator-webhook-host.

## What does it do?

It does a really simple thing just to show off the usage of the library: the extension will add an environment variable to Eirini apps pushed in CF.