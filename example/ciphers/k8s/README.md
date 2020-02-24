## K8S

This module allows you to run the example application on K8S using [ECK](https://www.elastic.co/guide/en/cloud-on-k8s/current/index.html).

The `Makefile` contains the needed targets, but in short:

1. `make install-eck`
2. `make deploy`
3. `make show-credentials` to show credentials for ES and APM. Copy and paste these to the tasques server and cipher 
   example server and worker configs. 
4. `make teardown` when done to teardown the env.

TLS has been disabled and anonymous access enabled for ease of playing around (but the setup uses the Kibana API, so
the credentials for the tasques ES cluster will still be needed)