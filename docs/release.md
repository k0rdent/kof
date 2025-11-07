# Release Checklist

* Bump versions,
  especially to use correct `-rc` version of `kof-opentelemetry-collector-contrib` image:
  ```
  make set-charts-version V=1.4.0-rc1
  ```
* Bump the `github.com/K0rdent/kcm` version in `kof-operator/go.mod` to e.g. `v1.4.0-rc1`
* Run: `cd kof-operator && go mod tidy && make test`
* Get this to `main` branch using PR as usual.
* Sync your fork and run e.g:
  ```
  git checkout main
  git pull
  git tag v1.4.0-rc1
  git remote add upstream git@github.com:k0rdent/kof.git
  git push upstream v1.4.0-rc1
  ```
* Open https://github.com/k0rdent/kof/actions and wait
  until CI creates the artifacts and the release draft.
* Open https://github.com/k0rdent/kof/releases and edit the release draft.
* Add the `## ❗ Upgrade Instructions ❗` section to the top of releases notes if needed.
* Once new docs are ready, add the `## 📚 New Docs 📚` section
  with the link to e.g. https://docs.k0rdent.io/v1.4.0/admin/kof/
  and the list of added/updated docs.
* Verify that auto-generated sections looks OK.
* Ensure the "Set as a pre-release" is checked and then "Publish release".
* Delete outdated RC if any. We need one final release per version + current pre-releases.
* Update the docs using PR to https://github.com/k0rdent/docs
  and make sure to copy "Upgrade Instructions" if any to the "Upgrading KOF" page.
* Add comment to internal issue with the link to this docs PR.
* Test the artifacts end-to-end by the docs.
* If the fix is needed, get it to `main` branch and restart with new RC.
* Check kof team and QA agrees that release is ready.
* Create and push the final tag, e.g. `v1.4.0` (without `-rc`).
* Verify artifacts, release notes, click "Publish release" this time, notify in Slack.
* If you've created a release branch earlier, delete it at GitHub and locally.
* Create the release branch, e.g:
  ```
  git checkout -b release/v1.4.0 v1.4.0
  git push -u upstream release/v1.4.0
  ```
* As we have a code freeze for features in `main` during RC testing,
  there is no need to create release branch before the final release is done.
* For CI to test upgrade from latest to future release, bump KOF charts version, e.g:
  ```
  make set-charts-version V=1.5.0-rc0
  ```
* Get this to `main` branch using PR as usual.
