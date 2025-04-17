# Release Checklist

## Simple flow

* Bump versions in:
  * `charts/*/Chart.yaml` - to e.g: `0.3.0`
  * `kof-operator/go.mod` for `github.com/K0rdent/kcm` to e.g: `v0.3.0`
  * `cd kof-operator && go mod tidy && make test`
* Get this to `main` branch using PR as usual.
* Open https://github.com/k0rdent/kof/releases and click:
  * Draft a new release.
  * Choose a tag - Find or create - e.g: `v0.3.0` - Create new tag.
  * Target - `main`
  * Previous tag - the latest full release (not RC), e.g. `v0.2.0`
  * Generate release notes.
  * Set as a pre-release.
  * Publish release.
* Open https://github.com/k0rdent/kof/actions and verify that CI created the artifacts.
* Update the docs using PR to https://github.com/k0rdent/docs
* Test the artifacts end-to-end by the docs.
* If the fix is needed, get it to `main`, delete the pre-release and its tag, draft it again.
* Check the team agrees that `kof` release is ready.
* Open https://github.com/k0rdent/kof/releases - e.g. `v0.3.0` - Edit, and click:
  * Set as the latest release
  * Update release.

## Complex flow

* Open https://github.com/k0rdent/kof/branches and click:
  * New branch - name e.g: `release/v0.2.0`
  * Source: `main`
  * Create new branch.
* Create an RC (Release Candidate) branch in your forked repo,
  based on upstream release branch, e.g:
  ```bash
  git remote add upstream git@github.com:k0rdent/kof.git
  git fetch upstream
  git checkout -b kof-v0.2.0-rc1 upstream/release/v0.2.0
  ```
* Bump versions in:
  * `charts/*/Chart.yaml` - to e.g: `0.2.0-rc1`
  * `kof-operator/go.mod` for `github.com/K0rdent/kcm` to e.g: `v0.2.0-rc1`
  * `cd kof-operator && go mod tidy && make test`
* Push, e.g: `git commit -am 'chore: kof v0.2.0-rc1' && git push -u origin kof-v0.2.0-rc1`
* Create a PR, selecting the base branch e.g: `release/v0.2.0`, get it approved and merged.
* Open https://github.com/k0rdent/kof/releases and click:
  * Draft a new release.
  * Choose a tag - Find or create - e.g: `v0.2.0-rc1` - Create new tag.
  * Target - e.g: `release/v0.2.0`
  * Previous tag - if this is `rc1`, then select the latest non-candidate,
    else select the latest release candidate for incremental notes.
  * Generate release notes.
  * Set as a pre-release.
  * Publish release.
* Open https://github.com/k0rdent/kof/actions and verify that CI created the artifacts.
* Update the docs using PR to https://github.com/k0rdent/docs
* Test the artifacts end-to-end by the docs.
* To fix something do e.g:
  ```bash
  git fetch upstream
  git checkout -b fix-something upstream/release/v0.2.0
  ```
  * Commit and push the fix, create a PR selecting the base branch e.g. `release/v0.2.0`, merge it.
  * Create one more PR via https://github.com/k0rdent/kof/compare
    e.g: `Syncing changes from release/v0.2.0 to main`
    using a regular merge commit (no squash) to keep the metadata of the original commits.
* Once there are enough fixes, create the next release candidate.
* Check the team agrees that `kof` release is ready.
* Bump to the final versions without `-rc`.
* Open https://github.com/k0rdent/kof/releases - and click:
  * Draft a new release.
  * Choose a tag - Find or create - e.g: `v0.2.0` - Create new tag.
  * Target - e.g: `release/v0.2.0`
  * Previous tag - e.g: `0.1.1` - the latest non-candidate for full release notes.
  * Generate release notes, and add headers for readability.
  * Set as the latest release
  * Publish release.
