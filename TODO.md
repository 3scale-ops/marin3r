TODO list:

- Add mTLS to marin3r (required by envoy sds api)
- Makefile that executes envoy with the config file to talk to the xds server
- Generate certificates for mTLS
- Code to push secret through sds:
    - Looks for the appropriate certificate in the cluster
    - Converts it to the secret envoy api
    - Updates the envoy xds cache
- Create a marin3r image
- Create the required yamls to deploy marin3r
- Deploy apicast-staging with an envoy sidecar using sds

The end!