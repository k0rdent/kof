# Release Checklist

* Bump versions in:
  * `charts/*/Chart.yaml` - to e.g. `1.3.0`
  * `Makefile` in `svctmpls` - to e.g. `1-3-0`
  * `kof-operator/go.mod` for `github.com/K0rdent/kcm` to e.g. `v1.3.0`
  * `cd kof-operator && go mod tidy && make test`
* Get this to `main` branch using PR as usual.
* Sync your fork and run e.g:
  ```
  git checkout main
  git pull
  git tag v1.3.0-rc1
  git remote add upstream git@github.com:k0rdent/kof.git
  git push upstream v1.3.0-rc1
  ```
* Open https://github.com/k0rdent/kof/actions and wait
  until CI creates the artifacts and the release draft.
* Open https://github.com/k0rdent/kof/releases and edit the release draft.
* Add the `## ❗ Upgrade Instructions ❗` section to the top of releases notes if needed.
* Once new docs are ready, add the `## 📚 New Docs 📚` section
  with the link to e.g. https://docs.k0rdent.io/v1.3.0/admin/kof/
  and the list of added/updated docs.
* Verify that auto-generated sections looks OK.
* Click "Publish release".
* Update the docs using PR to https://github.com/k0rdent/docs
  and make sure to copy "Upgrade Instructions" if any to the "Upgrading KOF" page.
* Add comment to internal issue with the link to this docs PR.
* Test the artifacts end-to-end by the docs.
* If the fix is needed, get it to `main` branch and restart with new RC.
* Check kof team and QA agrees that release is ready.
* Create and push the final tag, e.g. `v1.3.0` (without `-rc`).
* Verify artifacts, release notes, publish release, notify teams in Slack.
* If you've created a release branch earlier, delete it at GitHub and locally.
* Create the release branch, e.g:
  ```
  git checkout -b release/v1.3.0 v1.3.0
  git push -u upstream release/v1.3.0
  ```
* As we have a code freeze for features in `main` during RC testing,
  there is no need to create release branch before the final release is done.
