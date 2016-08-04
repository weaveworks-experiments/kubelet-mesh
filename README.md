### Intial features to enable cluster bootsrap without external dependencies

Kubelets can use Mesh for simple and secure discovery of API server URLs and root CA certs.

### Other potential features that Weave Mesh could enable

Rotation of root CA certs should be possible.

It should also be possible to pass initial componentconfig, and later updates it for all kubelets.

Additionally, it should be possible to use mesh to bootstrap HA control plane, full certificate rotation can be implemented also, and it could be leveraged for componentconfig as well.

Clients could also use Mesh, but it may be not exactly the best fit due to shortlived connections.

Mesh data path could be used for cheaper port forwarding.

Federation discovery can also be implement using Mesh channel per region.