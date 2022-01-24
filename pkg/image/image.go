package image

import "os"

//go:generate echo "[INFO] Generating files for pkg/image package"
//go:generate gen-pkg-image --image ${IMAGE}

// Current returns the current marin3r operator image
func Current() string {

	if imageOverride, ok := os.LookupEnv("MARIN3R_IMAGE"); image != "" && ok {
		return imageOverride
	}

	return image
}
