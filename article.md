# How to build CF Eirini Extensions with Eirinix

Working with Eirini and Cloud Foundry is really exciting.

The possibility to leverage directly the features of Kubernetes with the Cloud Foundry
Ecosystem opens up a new wide scenarios and completely new ecosystems around Eirini.

When you want to simply provide new features around Eirini, probably means you need to interact with Kubernetes API.
This can be a complex and repetitive work, and which usually takes time from the Design to the testing process (expecially in real-world scenarios).

At SUSE we've built a framework around Eirini and Kubernetes, that we called "**Eirinix**" to provide a common way to build Extensions around Eirini, using Kubernetes API.

At beginning, the components that are now part of Eirinix, were used to provide persistence support to Cloud Foundry Eirini apps with the usage of Persistent Volume Claims.

Eirinix allows you to focus on the logic of your extension instead of loosing time with writing common bootstrap operation to get your logic to work in a real kubernetes cluster.

In this article we will see how to build a simple Extension for Eirini with Eirinix and how to test it locally.

---

**Note**: The full sample that will be realized in this article can be found [here](https://github.com/SUSE/eirinix-sample).

## Mutating webhooks

When dealing with the problem on how-to extend Eirini in a Kubernetes "native" way, we re-used part of the solutions already implemented by what then became the [Quarks Project in Cloud Foundry](https://www.cloudfoundry.org/project-quarks/).

Eirinix is building a [Kubernetes Mutating webhook](https://kubernetes.io/docs/reference/access-authn-authz/admission-controllers/#mutatingadmissionwebhook) ( see also [admission webhook](https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers/#what-are-admission-webhooks) ) from each Extension, so each Extension can be also seen as a standalone Kubernetes operator targetting Eirini applications.

For semplification, we can look at mutating webhooks as a way to intercept Kubernetes requests dynamically, and Eirinix as a way to handle this setup step for you.

When an apiserver receives a request that matches one of the rules provided by the registered mutating webhooks in a kubernetes cluster, the apiserver sends an admissionReview request to webhook, and the webhook replies with a response patch.

The core of an Eirini Extension is to provide the operations that would be provided with a given POD by the API. But we will look into that with a real example

## Create your first Extension

We are going to create our first Extension and we will later run it on minikube. Let's say that we want to create an Extension that will inject a new environment variable to every application that is being pushed on a Cloud Foundry instance.

If the user pushes a general application X, we want that at the runtime the application has a new environment variable, let's call it ```EXAMPLE```, in their pod definition.

The same approach could be taken, for example to inject sidecar containers for each application.

What the extension should do then, is just modifying the POD definition, going over all the existing containers, and append the environment variable.

Let's create a new project for our extension, here in after called _eirinix-helloworld_, and we assume the repository path is _github.com/eirinix/eirinix-helloworld_,
it basically will be composed of two files: one which is our ```main.go```, and will run the extension, and one that holds the Extension logic ```hello/helloworld.go```.

That's our tree structure:

```
eirinix-heloworld
├── hello
│   └── helloworld.go
└── main.go

1 directory, 2 files
```

### hello/helloworld.go:

```golang
package hello

import (
    "context"
    "errors"
    "net/http"

    eirinix "github.com/SUSE/eirinix"
    corev1 "k8s.io/api/core/v1"
    "sigs.k8s.io/controller-runtime/pkg/webhook/admission"
    "sigs.k8s.io/controller-runtime/pkg/webhook/admission/types"
)

type Extension struct{}

func (ext *Extension) Handle(ctx context.Context, eiriniManager eirinix.Manager, pod *corev1.Pod, req types.Request) types.Response {

    if pod == nil {
        return admission.ErrorResponse(http.StatusBadRequest, errors.New("No pod could be decoded from the request"))
    }
    podCopy := pod.DeepCopy()

    for i := range podCopy.Spec.Containers {
        c := &podCopy.Spec.Containers[i]
        c.Env = append(c.Env, corev1.EnvVar{Name: "EXAMPLE", Value: "Eirinix is awesome!"})
    }
    return admission.PatchResponse(pod, podCopy)
}
```

```helloworld.go``` will contain our Eirini Extension.

An Eirinix Extension is a structure which has a defined ```Handle``` method. It needs to accept ```ctx context.Context, eiriniManager eirinix.Manager, pod *corev1.Pod, req types.Request``` , which are the Kubernetes API requests plus the Eirinix Manager interface, and returns a ```types.Response``` structure, which is a response that Kubernetes can understand. That's the only requested method by [the Extension interface](https://godoc.org/github.com/SUSE/eirinix#Extension).

In our case, our response will be derived from a difference between two pods: the original POD  that we receive from the request, and how we want the POD to actually look like after our changes.

In ```podCopy := pod.DeepCopy()``` we are taking a copy of the original POD definition from the request, and we create a new pod data structure (this is not happening in runtime, we aren't duplicating any real POD running in the cluster!).

Every change that we want to commit over a POD will be done to ```podCopy```, and later on we will compute the patchset from the two POD structures by calling ``` admission.PatchResponse(pod, podCopy)```, where ```podCopy``` is including our changes.

To keep things simple, we will create a ```main.go``` files which just starts our Extension with hardcoded cluster connection values.

Our ```main.go``` will be very simple, we just need at this point to register our Extension, and start the Eirinix Extension Manager.

### main.go:

```golang
package main

import (
    eirinix "github.com/SUSE/eirinix"

    // Change your import path here!
    helloworld "github.com/eirinix/eirinix-helloworld/hello"
)

func main() {
    x := eirinix.NewManager(
            eirinix.ManagerOptions{
                Namespace:           "default",
                Host:                "10.0.2.2",
                Port:                3000,
                KubeConfig:          "~/.kube/config",
                OperatorFingerprint: "eirini-x-helloworld",
            })

    x.AddExtension(&helloworld.Extension{})
    x.Start()
}
````

Here we create a new Eirinix Manager passing [eirinix.ManagerOptions](https://godoc.org/github.com/SUSE/eirinix#ManagerOptions) with the connection parameters. ```Namespace``` is set to default, it means that we expect Eirini Application pods to be pushed here. ```Host``` is the ip, or hostname where the Manager server is listening to. You need to set this value to a IP/domain which is reachable from the kubernetes API server.

In our case, we will set it to ```10.0.2.2``` as we will use later minikube to test our extension (in minikube, that is the IP of the host reachable from the kubeapi server). 

```Port``` it is the port where the Manager server is listening to. Can be set to any value, just make sure that port is reachable if you run this inside a cluster or not declined by any fw rule.

```KubeConfig``` accepts a path to a Kubernetes config file, or otherwise omit it for in-cluster connection.

```OperatorFingerprint``` It's a unique identifier for your operator. It can be any arbitrary value. You need to set this only in the case you are planning to run more than one manager in the same cluster (in different PODs).

With ```x.AddExtension(&helloworld.Extension{})``` we are adding our Extension to the manager, later on the Manager will build a mutating webhook from it and will run it to serve requests. You can add more extension in the same process.

```x.Start()``` starts the main loop. It returns an error in case there were runtime issues (connection failures to k8s, etc. )