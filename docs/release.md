# Release

* Update Makefile variable `VERSION` to the appropiate release version. Allowed formats:
  * alpha: `VERSION ?= 0.3.0-alpha.12`
  * stable: `VERSION ?= 0.3.0`

## Alpha
* If it is an **alpha** release, execute the following target to create appropiate `alpha` bundle files:
```bash
make prepare-alpha-release
```
* Then you can manually execute opeator, bundle and catalog build/push.

## Stable
* But if it is an **stable** release, execute the following target to create appropiate `alpha` and `stable` bundle files:
```bash
make prepare-stable-release
```
* Then open a [Pull Request](https://github.com/3scale-ops/marin3r/pulls), and a GitHub Action will automatically detect if it is new release or not, in order to create it by building/pushing new operator, bundle and catalog images, as well as creating a GitHub release draft.