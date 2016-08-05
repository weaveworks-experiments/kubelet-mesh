#!/bin/bash -x
./kubelet-mesh -nickname master -hwaddr 6c:40:08:94:9e:01 -mesh 0.0.0.0:6783 -password VerySecure -root-ca ca.crt -apiserver "https://k8s-1.example.org" &
./kubelet-mesh -nickname node01 -hwaddr 6c:40:08:94:9e:02 -mesh 0.0.0.0:6784 -password VerySecure -peer 127.0.0.1:6783 -apiserver "http://localhost:8080"
until killall kubelet-mesh ; do sleep 1 ; done
