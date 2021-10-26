package envoy

//go:generate echo "[INFO] Generating files for pkg/envoy/protos/v3 package"
//go:generate gen-pkg-envoy-proto --api-version v3 --package-file zz_generated.go --gomod-file ../../../../go.mod
