# Pod dive

![lists all restarts in your cluster and pods last start time](logo-256.png)

A `kubectl` [Krew](https://krew.dev) plugin to lists all restarts in your cluster and show you pods last start times.

Icon art made by [Pixelmeetup](https://www.flaticon.com/authors/pixelmeetup) from [Flaticon](https://www.flaticon.com/). [We had one before Krew itself](https://github.com/kubernetes-sigs/krew/issues/437), go figure.

## Quick Start

If you don't use Krew to manage `kubectl` plugins [you can simply download the binary here](https://github.com/caiobegotti/pod-restarts/releases) and put it in your PATH.

```
kubectl krew install pod-restarts
kubectl pod-restarts
```

## Why use it

Sometimes using a `kubectl` command is much faster than running a bunch of Prometheus graphs and only need a quick glance of what has been restarting lately or during a migration.

## What does it look like

```
$ kubectl pod-restarts --help

Usage:
  pod-restarts [flags]

Examples:
```