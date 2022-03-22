# Draft Releaser Action

Repository for draft releaser action for GitHub Actions.

Use as follows:

```yaml
jobs:
  autorelease:
    name: Auto release drafts
    runs-on: ubuntu-latest
    steps:
      - name: Checkout Code
        uses: actions/checkout@v3
      - name: Auto Release
        uses: clarkjohnd/draft-releaser-action@v0.0.2
```

By default, it will publish draft releases created by the draft-releaser action that meet the following conditions:

- The draft release is the LATEST release entry, there are no more recent releases
- Draft release is over 7 days old
- It only consists of Dependency Upgrades

## Options

```release-days```: The minimum age of the draft release to publish (default 7)

```exclude-labels```: Labels to check for and if exist do not publish, instead notify etc. (default "Documentation,Features,Bug Fixes")

```all-labels```: If set to any value, will override ```exclude-labels``` and publish if any label found (default null)

```dry-run```: if set to any value, will run the checks but will not modify releases (default null)

## Other

Need to add functionality to do something when conditions are met but more than just dependency upgrades are met. Slack channel ping or something to remind a team to publish the release?

Action currently builds Docker image on runtime due to Mac OS building ARM64 images which fail to run in GitHub Actions. Update to add buildx etc. for Docker image publishing and update the ```action.yml``` ```runs.image``` reference to ```docker://registry:tag```.
