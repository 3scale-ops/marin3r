# Release

* Update Makefile variable `VERSION` to the appropiate release version. Allowed formats:
  * alpha: `VERSION ?= 0.12.1-alpha.1`
  * stable: `VERSION ?= 0.12.1`

## Alpha

* If it is an **alpha** release, execute the following target to create appropiate `alpha` bundle files:

```bash
make prepare-alpha-release
```

* Then you can manually execute operator, bundle and catalog build/push targets.

```bash
make bundle-publish
```

```bash
make catalog-add-bundle-to-alpha
```

```bash
make catalog-publish
```

## Stable

* If it is an **stable** release, execute the following target to create appropiate `alpha` and `stable` bundle files:

```bash
make prepare-stable-release
```

* Then open a [Pull Request](https://github.com/3scale-ops/marin3r/pulls), and a GitHub Action will automatically detect if it is new release or not, in order to create it by building/pushing new operator and bundle images, as well as creating a GitHub release draft.

* After the release of a stable version, you need to update the catalog. To do so execute the following targets:

```bash
make catalog-add-bundle-to-alpha && make catalog-add-bundle-to-stable
```

Then commit the changes and open a [Pull Request](https://github.com/3scale-ops/marin3r/pulls). A GitHub action will pick up the change in the stable channel and build & push a new catalog image.
