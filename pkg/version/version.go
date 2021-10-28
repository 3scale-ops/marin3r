package version

//go:generate echo "[INFO] Generating files for pkg/version package"
//go:generate gen-pkg-version --version ${VERSION}

// Current returns the current marin3r operator version
func Current() string { return version }
