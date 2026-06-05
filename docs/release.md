# Release Checklist

* The examples below use a release branch `release/v1.9.0` and tags like `v1.9.0-rc1`.
  Please adjust to the current release version.

## Date of rc1

* Configure the `upstream`:
  ```
  git remote add upstream git@github.com:k0rdent/kof.git
  ```
* Sync the `main` branch of your fork:
  ```
  git fetch upstream
  git checkout main
  git merge --ff-only upstream/main
  ```
* Create a release branch, e.g:
  ```
  git checkout -b release/v1.9.0
  git push upstream release/v1.9.0
  ```

## Iteration

* Create a PR targeting the release branch, not `main`:
  * Create RC branch, e.g:
    ```
    git checkout release/v1.9.0
    git checkout -b kof-1-9-0-rc1
    ```
  * Bump the versions in the charts,
    especially to use correct version of `kof-opentelemetry-collector-contrib` image:
    ```
    make set-charts-version V=1.9.0-rc1
    ```
  * Bump the `github.com/K0rdent/kcm` version in `kof-operator/go.mod` to e.g. `v1.9.0-rc1`
    if it is available at https://github.com/k0rdent/kcm/tags already.
  * Run:
    ```
    cd kof-operator
    go mod tidy
    make test
    ```
  * Commit, push, create PR, and get it merged to the release branch, not `main`.
* Sync the release branch of your fork, e.g:
  ```
  git fetch upstream
  git checkout release/v1.9.0
  git merge --ff-only upstream/release/v1.9.0
  ```
* Create and push the tag, e.g:
  ```
  git tag v1.9.0-rc1
  git push upstream v1.9.0-rc1
  ```
* Open https://github.com/k0rdent/kof/actions and wait
  until CI creates the artifacts and the release draft.
* Open https://github.com/k0rdent/kof/releases and edit the release draft.
* Verify the auto-generated sections look OK.
* Ensure the `Set as a pre-release` is checked.
* Click the `Publish release`.
* Update the docs using PR to https://github.com/k0rdent/docs
  * Bump `kofDotVersion` in the `mkdocs.yml` to the final version,
    e.g. `1.9.0` without `-rc` to avoid updating it every time.
  * Add e.g. `Upgrade to v1.9.0` section to the [kof-upgrade-includes.md](https://github.com/k0rdent/docs/blob/main/includes/kof-upgrade-includes.md?plain=1) file
    if any special steps are needed for this upgrade.
  * Add a link to this section (once published)
    to the `## ❗ Upgrade Instructions ❗` section in the top of the GitHub release notes.
  * Ensure each new feature of the release is documented, if it makes sense.
  * Add the links to the added/updated docs/sections
    to the `## 📚 New Docs 📚` section, second in the GitHub release notes.
  * Start with the [next version](https://docs.k0rdent.io/next/admin/kof/) links.
  * Replace them with the final version links once they are available.
* Test the RC artifacts end-to-end by the docs.
* If the fix is needed:
  * Apply the [Iteration](#iteration) including the fix, `rc2`, and so on.

## Final release date

* Apply the [Iteration](#iteration), but use the final version, e.g. `v1.9.0` without `-rc`.
* Ensure the `Set as the latest release` is checked.
* Click the `Publish release`.
* Notify in Slack.
* Merge the release branch back to the `main` branch, e.g:
  ```
  git fetch upstream
  git checkout main
  git merge upstream/release/v1.9.0  # No --ff-only this time.
  ```
  * If the git conflicts are too risky:
    * Then consider making a PR to test the result with CI, e.g:
      ```
      git merge --abort
      git checkout -b kof-1-9-0
      git merge upstream/release/v1.9.0
      # Resolve git conflicts.
      git commit
      ```
    * Else resolve the simple git conflicts and run:
      ```
      git commit
      ```
  * For the cloud clusters to use the latest version,
    and for CI to test the upgrade from the latest to the future version, run e.g:
    ```
    make set-charts-version LATEST_V=1.9.0 V=1.10.0-rc0
    git commit
    ```
  * If you're creating the PR:
    * Then `git push origin kof-1-9-0`
    * Else `git push upstream main`
* Delete the outdated RCs (not the final releases) from the [GitHub releases page](https://github.com/k0rdent/kof/releases).
