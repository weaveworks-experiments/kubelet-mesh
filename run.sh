#!/bin/bash -x
./kubelet-mesh -hwaddr 6c:40:08:94:9e:01 -mesh 0.0.0.0:6783 -password VerySecure -root-ca ca.crt &
./kubelet-mesh -hwaddr 6c:40:08:94:9e:02 -mesh 0.0.0.0:6784 -password VerySecure -peer 127.0.0.1:6783
until killall kubelet-mesh ; do sleep 1 ; done
